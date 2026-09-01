package controller

import (
	"context"
	"strings"
	"testing"

	palworldv1alpha1 "github.com/DataKnifeAI/palworld-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func testServerForMods(enabled bool) *palworldv1alpha1.PalworldServer {
	server := testServerForSecrets(false)
	server.Spec.StorageClassName = "truenas-csi-nfs"
	server.Spec.Mods.Enabled = enabled
	return server
}

func volumeByName(volumes []corev1.Volume, name string) *corev1.Volume {
	for i := range volumes {
		if volumes[i].Name == name {
			return &volumes[i]
		}
	}
	return nil
}

func mountByName(mounts []corev1.VolumeMount, name string) *corev1.VolumeMount {
	for i := range mounts {
		if mounts[i].Name == name {
			return &mounts[i]
		}
	}
	return nil
}

func mountByPath(mounts []corev1.VolumeMount, path string) *corev1.VolumeMount {
	for i := range mounts {
		if mounts[i].MountPath == path {
			return &mounts[i]
		}
	}
	return nil
}

func TestModsDisabledNoVolume(t *testing.T) {
	spec := palworldv1alpha1.PalworldServerSpec{
		Gateway: palworldv1alpha1.GatewayConfig{Address: testGatewayAddress},
	}
	if modsEnabled(spec) {
		t.Fatal("mods must default disabled")
	}
	server := &palworldv1alpha1.PalworldServer{
		ObjectMeta: metav1.ObjectMeta{Name: testCRName},
		Spec:       spec,
	}
	names := deriveNames(server)
	if volumeByName(gameVolumes(names, spec), volumeMods) != nil {
		t.Fatal("disabled mods must not add a mods volume")
	}
	if mountByName(gameVolumeMounts(spec), volumeMods) != nil {
		t.Fatal("disabled mods must not add a mods mount")
	}
	if mountByPath(gameVolumeMounts(spec), officialPaksRoot) != nil {
		t.Fatal("disabled mods must not mount Paks/")
	}
	if mountByName(seedInitVolumeMounts(spec), volumeMods) != nil {
		t.Fatal("disabled mods must not mount mods on seed-settings")
	}
	if seedPalModSettings(spec) {
		t.Fatal("disabled mods must not seed PalModSettings.ini")
	}
}

func TestModsEnabledVolumeAndMount(t *testing.T) {
	spec := palworldv1alpha1.PalworldServerSpec{
		Gateway: palworldv1alpha1.GatewayConfig{Address: testGatewayAddress},
		Mods: palworldv1alpha1.ModsConfig{
			Enabled: true,
		},
	}
	server := &palworldv1alpha1.PalworldServer{
		ObjectMeta: metav1.ObjectMeta{Name: testCRName},
		Spec:       spec,
	}
	names := deriveNames(server)
	vol := volumeByName(gameVolumes(names, spec), volumeMods)
	if vol == nil || vol.PersistentVolumeClaim == nil {
		t.Fatal("enabled mods must add a PVC volume")
	}
	if vol.PersistentVolumeClaim.ClaimName != "palworld-server-mods" {
		t.Fatalf("mods claim = %q, want palworld-server-mods", vol.PersistentVolumeClaim.ClaimName)
	}
	mounts := gameVolumeMounts(spec)
	root := mountByPath(mounts, officialModsMountPath)
	if root == nil || root.Name != volumeMods {
		t.Fatalf("enabled mods must mount PVC at %s", officialModsMountPath)
	}
	ws := mountByPath(mounts, officialPaksRoot+"/"+paksWorkshopModsDir)
	if ws == nil || ws.SubPath != paksOverlayWorkshopSub {
		t.Fatalf("~WorkshopMods overlay = %+v", ws)
	}
	logic := mountByPath(mounts, officialPaksRoot+"/"+paksLogicModsDir)
	if logic == nil || logic.SubPath != paksOverlayLogicSub {
		t.Fatalf("LogicMods overlay = %+v", logic)
	}
	if mountByPath(mounts, officialPaksRoot) != nil {
		t.Fatal("must not replace Pal/Content/Paks (would hide Pal-LinuxServer.pak)")
	}
	if mountByName(seedInitVolumeMounts(spec), volumeMods) == nil {
		t.Fatal("enabled mods must mount mods on seed-settings")
	}
	if modsWorkshopDir(spec) != officialModsMountPath+"/"+workshopSubdir {
		t.Fatalf("workshop dir = %q", modsWorkshopDir(spec))
	}
}

func TestModsStorageAndPathDefaults(t *testing.T) {
	spec := palworldv1alpha1.PalworldServerSpec{
		StorageClassName: "saves-class",
		Mods: palworldv1alpha1.ModsConfig{
			Enabled: true,
		},
	}
	if got := modsStorageSize(spec); got != defaultModsStorageSize {
		t.Fatalf("mods size = %q, want %q", got, defaultModsStorageSize)
	}
	if got := modsStorageClassName(spec); got != "saves-class" {
		t.Fatalf("mods class = %q, want saves-class", got)
	}

	spec.Mods.Storage.Size = "20Gi"
	spec.Mods.Storage.StorageClassName = "mods-class"
	spec.Mods.Path = "/custom/Mods"
	spec.Mods.WorkshopDir = "/custom/Workshop"
	if got := modsStorageSize(spec); got != "20Gi" {
		t.Fatalf("override size = %q", got)
	}
	if got := modsStorageClassName(spec); got != "mods-class" {
		t.Fatalf("override class = %q", got)
	}
	if got := modsMountPath(spec); got != "/custom/Mods" {
		t.Fatalf("override path = %q", got)
	}
	if got := modsWorkshopDir(spec); got != "/custom/Workshop" {
		t.Fatalf("override workshop = %q", got)
	}

	community := palworldv1alpha1.PalworldServerSpec{
		ServerImage: "thijsvanloef/palworld-server-docker:latest",
		Mods:        palworldv1alpha1.ModsConfig{Enabled: true},
	}
	if got := modsMountPath(community); got != communityModsMountPath {
		t.Fatalf("community mods path = %q, want %q", got, communityModsMountPath)
	}
	if got := paksWorkshopModsMount(community); got != communityPaksRoot+"/"+paksWorkshopModsDir {
		t.Fatalf("community ~WorkshopMods = %q", got)
	}

	off := false
	spec.Mods.PaksOverlay = &off
	if modsPaksOverlay(spec) {
		t.Fatal("paksOverlay=false must skip Paks subpath mounts")
	}
	if mountByPath(gameVolumeMounts(spec), officialPaksRoot+"/"+paksWorkshopModsDir) != nil {
		t.Fatal("paksOverlay=false must not mount ~WorkshopMods")
	}
	if mountByPath(gameVolumeMounts(spec), spec.Mods.Path) == nil {
		t.Fatal("paksOverlay=false must still mount Mods/")
	}
}

func TestSeedModsLayoutScriptQuotesTilde(t *testing.T) {
	script := seedModsLayoutScript()
	if !strings.Contains(script, `"/mods/paks/~WorkshopMods"`) {
		t.Fatalf("expected quoted ~WorkshopMods path in %s", script)
	}
	if !strings.Contains(script, "/mods/paks/LogicMods") {
		t.Fatalf("expected LogicMods dir in %s", script)
	}
}

func TestOfficialCommandArgsWorkshopDir(t *testing.T) {
	base := palworldv1alpha1.PalworldServerSpec{
		GamePort: 8211,
		Mods: palworldv1alpha1.ModsConfig{
			Enabled: true,
		},
	}
	joined := strings.Join(officialCommandArgs(base), " ")
	if strings.Contains(joined, workshopDirArgPrefix) {
		t.Fatalf("enabled mods must not pass workshopdir by default: %s", joined)
	}

	base.Mods.UseWorkshopDirArg = true
	args := officialCommandArgs(base)
	want := workshopDirArgPrefix + officialModsMountPath + "/" + workshopSubdir
	found := false
	for _, arg := range args {
		if arg == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected %q in %v", want, args)
	}
}

func TestBuildPalModSettingsINI(t *testing.T) {
	spec := palworldv1alpha1.PalworldServerSpec{
		Mods: palworldv1alpha1.ModsConfig{
			Enabled:       true,
			ActiveModList: []string{"GamingCattiva", " FarmingQuivern ", ""},
			WorkshopDir:   "/pal/Package/Mods/Workshop",
		},
	}
	body := buildPalModSettingsINI(spec)
	for _, want := range []string{
		palModSettingsSection,
		"bGlobalEnableMod=true",
		"ActiveModList=GamingCattiva",
		"ActiveModList=FarmingQuivern",
		"WorkshopRootDir=/pal/Package/Mods/Workshop",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in %s", want, body)
		}
	}
	if !seedPalModSettings(spec) {
		t.Fatal("non-empty activeModList should seed PalModSettings.ini")
	}
}

func TestReconcilePVCModsEnabledVsDisabled(t *testing.T) {
	scheme := secretsTestScheme(t)
	ctx := context.Background()

	disabled := testServerForMods(false)
	rDisabled := &PalworldServerReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(disabled).Build(),
		Scheme: scheme,
	}
	names := deriveNames(disabled)
	if err := rDisabled.reconcilePVC(ctx, disabled, names.pvcName, storageSize(disabled.Spec), disabled.Spec.StorageClassName); err != nil {
		t.Fatalf("saves PVC: %v", err)
	}
	if modsEnabled(disabled.Spec) {
		t.Fatal("fixture should be disabled")
	}
	saves := &corev1.PersistentVolumeClaim{}
	if err := rDisabled.Get(ctx, types.NamespacedName{Name: names.pvcName, Namespace: disabled.Namespace}, saves); err != nil {
		t.Fatalf("saves PVC missing: %v", err)
	}
	mods := &corev1.PersistentVolumeClaim{}
	err := rDisabled.Get(ctx, types.NamespacedName{Name: names.modsPVCName, Namespace: disabled.Namespace}, mods)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("mods PVC should be absent when disabled, got %v", err)
	}

	enabled := testServerForMods(true)
	enabled.Spec.Mods.Storage.Size = "10Gi"
	rEnabled := &PalworldServerReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(enabled).Build(),
		Scheme: scheme,
	}
	enNames := deriveNames(enabled)
	if err := rEnabled.reconcilePVC(ctx, enabled, enNames.pvcName, storageSize(enabled.Spec), enabled.Spec.StorageClassName); err != nil {
		t.Fatalf("saves PVC: %v", err)
	}
	if err := rEnabled.reconcilePVC(ctx, enabled, enNames.modsPVCName, modsStorageSize(enabled.Spec), modsStorageClassName(enabled.Spec)); err != nil {
		t.Fatalf("mods PVC: %v", err)
	}
	if err := rEnabled.Get(ctx, types.NamespacedName{Name: enNames.modsPVCName, Namespace: enabled.Namespace}, mods); err != nil {
		t.Fatalf("mods PVC missing when enabled: %v", err)
	}
	if mods.Name != "palworld-test-mods" {
		t.Fatalf("mods PVC name = %q", mods.Name)
	}
	if got := mods.Spec.Resources.Requests.Storage().String(); got != defaultModsStorageSize {
		t.Fatalf("mods PVC size = %q, want %q", got, defaultModsStorageSize)
	}
	if mods.Spec.StorageClassName == nil || *mods.Spec.StorageClassName != "truenas-csi-nfs" {
		t.Fatalf("mods StorageClass = %v, want saves class", mods.Spec.StorageClassName)
	}
	if len(mods.OwnerReferences) != 1 || mods.OwnerReferences[0].UID != enabled.UID {
		t.Fatalf("mods owner = %+v", mods.OwnerReferences)
	}
	savesEnabled := &corev1.PersistentVolumeClaim{}
	if err := rEnabled.Get(ctx, types.NamespacedName{Name: enNames.pvcName, Namespace: enabled.Namespace}, savesEnabled); err != nil {
		t.Fatalf("saves PVC must still exist: %v", err)
	}
}

func TestReconcileDeploymentModsVolume(t *testing.T) {
	scheme := secretsTestScheme(t)
	ctx := context.Background()

	disabled := testServerForMods(false)
	rDisabled := &PalworldServerReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(disabled).Build(),
		Scheme: scheme,
	}
	if err := rDisabled.reconcileDeployment(ctx, disabled, deriveNames(disabled), "admin", "join"); err != nil {
		t.Fatalf("deploy disabled: %v", err)
	}
	dep := &appsv1.Deployment{}
	if err := rDisabled.Get(ctx, types.NamespacedName{Name: disabled.Name, Namespace: disabled.Namespace}, dep); err != nil {
		t.Fatalf("get deploy: %v", err)
	}
	pod := dep.Spec.Template.Spec
	if volumeByName(pod.Volumes, volumeMods) != nil {
		t.Fatal("disabled deploy must not have mods volume")
	}
	if mountByName(pod.Containers[0].VolumeMounts, volumeMods) != nil {
		t.Fatal("disabled deploy must not mount mods")
	}
	if strings.Contains(strings.Join(pod.Containers[0].Args, " "), workshopDirArgPrefix) {
		t.Fatal("disabled deploy must not pass workshopdir")
	}

	enabled := testServerForMods(true)
	rEnabled := &PalworldServerReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(enabled).Build(),
		Scheme: scheme,
	}
	if err := rEnabled.reconcileDeployment(ctx, enabled, deriveNames(enabled), "admin", "join"); err != nil {
		t.Fatalf("deploy enabled: %v", err)
	}
	if err := rEnabled.Get(ctx, types.NamespacedName{Name: enabled.Name, Namespace: enabled.Namespace}, dep); err != nil {
		t.Fatalf("get enabled deploy: %v", err)
	}
	pod = dep.Spec.Template.Spec
	vol := volumeByName(pod.Volumes, volumeMods)
	if vol == nil || vol.PersistentVolumeClaim == nil || vol.PersistentVolumeClaim.ClaimName != "palworld-test-mods" {
		t.Fatalf("enabled deploy mods volume = %+v", vol)
	}
	if mountByPath(pod.Containers[0].VolumeMounts, officialModsMountPath) == nil {
		t.Fatal("enabled deploy must mount /pal/Package/Mods")
	}
	if mountByPath(pod.Containers[0].VolumeMounts, officialPaksRoot+"/"+paksWorkshopModsDir) == nil {
		t.Fatal("enabled deploy must overlay ~WorkshopMods")
	}
	if mountByPath(pod.Containers[0].VolumeMounts, officialPaksRoot) != nil {
		t.Fatal("enabled deploy must not replace whole Paks/")
	}
	if volumeByName(pod.Volumes, volumeSaves) == nil {
		t.Fatal("saves volume must remain")
	}
	if strings.Contains(strings.Join(pod.Containers[0].Args, " "), workshopDirArgPrefix) {
		t.Fatal("enabled deploy must not pass workshopdir unless opted in")
	}
	if len(pod.InitContainers) != 1 || mountByName(pod.InitContainers[0].VolumeMounts, volumeMods) == nil {
		t.Fatal("seed-settings must mount mods when enabled")
	}
}
