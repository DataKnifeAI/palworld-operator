package controller

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	palworldv1alpha1 "github.com/DataKnifeAI/palworld-operator/api/v1alpha1"
)

type updateProbe struct {
	info    restInfo
	metrics restMetrics
	ok      bool
}

func (r *PalworldServerReconciler) tagLister() TagLister {
	if r.TagLister != nil {
		return r.TagLister
	}
	return &GHCRTagLister{}
}

func (r *PalworldServerReconciler) restClient() RESTClient {
	if r.RESTClient != nil {
		return r.RESTClient
	}
	return &HTTPRESTClient{}
}

func (r *PalworldServerReconciler) probeREST(
	ctx context.Context,
	server *palworldv1alpha1.PalworldServer,
	names derivedNames,
	adminPassword string,
) updateProbe {
	var out updateProbe
	if !restEnabled(server.Spec) || adminPassword == "" {
		return out
	}
	base := restBaseURL(names.serviceName, server.Namespace, restPort(server.Spec))
	info, err := r.restClient().GetInfo(ctx, base, adminPassword)
	if err != nil {
		logf.FromContext(ctx).V(1).Info("REST info unavailable", "error", err.Error())
		return out
	}
	out.info = info
	out.ok = true
	metrics, err := r.restClient().GetMetrics(ctx, base, adminPassword)
	if err != nil {
		logf.FromContext(ctx).V(1).Info("REST metrics unavailable", "error", err.Error())
		return out
	}
	out.metrics = metrics
	return out
}

func canSafelyAutoUpdate(server *palworldv1alpha1.PalworldServer, worldGUID string) bool {
	if dedicatedServerName(server) != "" || worldGUID != "" {
		return true
	}
	// No world pin yet — allow only before the server has been Ready with a world.
	return !server.Status.Ready
}

// maybeAutoUpdate checks GHCR (on schedule/interval), updates status fields, and
// optionally patches spec.serverImage. Returns requeue delay and whether the CR
// spec was mutated (caller should skip further work / requeue).
func (r *PalworldServerReconciler) maybeAutoUpdate(
	ctx context.Context,
	server *palworldv1alpha1.PalworldServer,
	names derivedNames,
	adminPassword string,
	probe updateProbe,
	now time.Time,
) (requeueAfter time.Duration, specMutated bool, err error) {
	log := logf.FromContext(ctx)
	desired := serverImage(server.Spec)
	server.Status.DesiredImage = desired

	if probe.ok {
		if probe.info.Version != "" {
			server.Status.RunningVersion = probe.info.Version
		}
		if probe.info.WorldGUID != "" {
			server.Status.DedicatedServerName = probe.info.WorldGUID
		}
		pc := probe.metrics.CurrentPlayerNum
		server.Status.PlayerCount = &pc
	}

	repo := imageRepository(server.Spec)
	requeueAfter = defaultUpdateRequeue(server.Spec)

	if !server.Spec.Update.AutoUpdateImage {
		// Lightweight status refresh of latest tag so updateAvailable is visible
		// without opting into mutation. Does not change Message.
		server.Status.Message = ""
		return r.refreshLatestStatus(ctx, server, repo, desired, now)
	}

	if !imageMatchesRepository(desired, repo) {
		server.Status.Message = fmt.Sprintf("auto-update skipped: spec.serverImage is not from %s", repo)
		return requeueAfter, false, nil
	}

	due, err := shouldCheckRegistry(server.Spec, timePtr(server.Status.LastImageCheckTime), now)
	if err != nil {
		return 0, false, err
	}

	latest := server.Status.LatestAvailableVersion
	if due {
		tags, listErr := r.tagLister().ListTags(ctx, repo)
		checkTime := metav1.NewTime(now)
		server.Status.LastImageCheckTime = &checkTime
		if listErr != nil {
			log.Error(listErr, "failed to list image tags", "repository", repo)
			server.Status.Message = fmt.Sprintf("image tag check failed: %v", listErr)
			return updateRequeueBusy, false, nil
		}
		if tag, ok := newestPalVersionTag(tags); ok {
			latest = tag
			server.Status.LatestAvailableVersion = tag
		}
	}

	updateAvail := shouldUpdateImage(desired, server.Status.RunningVersion, latest)
	server.Status.UpdateAvailable = updateAvail
	if !updateAvail {
		server.Status.PendingUpdateImage = ""
		server.Status.LastAnnounceTime = nil
		server.Status.Message = ""
		return requeueAfter, false, nil
	}

	targetImage := formatImageRef(repo, latest)
	worldGUID := ""
	if probe.ok {
		worldGUID = probe.info.WorldGUID
	}
	if !canSafelyAutoUpdate(server, worldGUID) {
		server.Status.Message = "update available; waiting to learn DedicatedServerName (world pin) before rolling"
		return updateRequeueBusy, false, nil
	}

	inWindow, err := inApplyWindow(server.Spec, now)
	if err != nil {
		return 0, false, err
	}
	if !inWindow {
		server.Status.Message = fmt.Sprintf("update available (%s); waiting for applySchedule window", latest)
		return updateRequeueBusy, false, nil
	}

	if updateOnlyWhenEmpty(server.Spec) && probe.ok && probe.metrics.CurrentPlayerNum > 0 {
		server.Status.Message = fmt.Sprintf("update available (%s); deferring while %d player(s) online", latest, probe.metrics.CurrentPlayerNum)
		return updateRequeueBusy, false, nil
	}
	if updateOnlyWhenEmpty(server.Spec) && !probe.ok && server.Status.Ready {
		server.Status.Message = fmt.Sprintf("update available (%s); waiting for REST player count before rolling", latest)
		return updateRequeueBusy, false, nil
	}

	// Notify path: announce then wait lead time.
	if server.Spec.Update.NotifyPlayers {
		lead := notifyLeadTime(server.Spec)
		needAnnounce := server.Status.PendingUpdateImage != targetImage ||
			server.Status.LastAnnounceTime == nil
		if needAnnounce {
			if !restEnabled(server.Spec) || adminPassword == "" {
				return 0, false, fmt.Errorf("notifyPlayers requires REST API and admin password")
			}
			msg := formatNotifyMessage(server.Spec, latest, targetImage)
			base := restBaseURL(names.serviceName, server.Namespace, restPort(server.Spec))
			if annErr := r.restClient().Announce(ctx, base, adminPassword, msg); annErr != nil {
				log.Error(annErr, "pre-update announce failed")
				server.Status.Message = fmt.Sprintf("update announce failed: %v", annErr)
				return updateRequeueBusy, false, nil
			}
			announced := metav1.NewTime(now)
			server.Status.LastAnnounceTime = &announced
			server.Status.PendingUpdateImage = targetImage
			server.Status.Message = fmt.Sprintf("announced update to %s; applying after %s", latest, lead)
			return lead, false, nil
		}
		if server.Status.LastAnnounceTime != nil && now.Before(server.Status.LastAnnounceTime.Add(lead)) {
			remaining := server.Status.LastAnnounceTime.Add(lead).Sub(now)
			server.Status.Message = fmt.Sprintf("announced update to %s; applying in %s", latest, remaining.Round(time.Second))
			return remaining, false, nil
		}
	}

	if desired == targetImage {
		return requeueAfter, false, nil
	}

	log.Info("auto-updating serverImage", "from", desired, "to", targetImage)
	patched := server.DeepCopy()
	patched.Spec.ServerImage = targetImage
	if err := r.Patch(ctx, patched, client.MergeFrom(server)); err != nil {
		return 0, false, fmt.Errorf("patch serverImage: %w", err)
	}
	server.Spec.ServerImage = targetImage
	server.Status.DesiredImage = targetImage
	server.Status.PendingUpdateImage = ""
	server.Status.LastAnnounceTime = nil
	server.Status.UpdateAvailable = false
	server.Status.Message = fmt.Sprintf("auto-updated serverImage to %s", targetImage)
	return updateRequeueSoon, true, nil
}

func defaultUpdateRequeue(spec palworldv1alpha1.PalworldServerSpec) time.Duration {
	if spec.Update.CheckSchedule != "" || spec.Update.ApplySchedule != "" {
		return updateRequeueBusy
	}
	return updateCheckInterval(spec)
}

func (r *PalworldServerReconciler) refreshLatestStatus(
	ctx context.Context,
	server *palworldv1alpha1.PalworldServer,
	repo, desired string,
	now time.Time,
) (time.Duration, bool, error) {
	requeueAfter := defaultUpdateRequeue(server.Spec)
	due, err := shouldCheckRegistry(server.Spec, timePtr(server.Status.LastImageCheckTime), now)
	if err != nil {
		return 0, false, err
	}
	if !due {
		server.Status.UpdateAvailable = shouldUpdateImage(desired, server.Status.RunningVersion, server.Status.LatestAvailableVersion)
		return requeueAfter, false, nil
	}
	tags, listErr := r.tagLister().ListTags(ctx, repo)
	checkTime := metav1.NewTime(now)
	server.Status.LastImageCheckTime = &checkTime
	if listErr != nil {
		logf.FromContext(ctx).V(1).Info("optional latest-tag refresh failed", "error", listErr.Error())
		return requeueAfter, false, nil
	}
	if tag, ok := newestPalVersionTag(tags); ok {
		server.Status.LatestAvailableVersion = tag
		server.Status.UpdateAvailable = shouldUpdateImage(desired, server.Status.RunningVersion, tag)
	}
	return requeueAfter, false, nil
}

func timePtr(t *metav1.Time) *time.Time {
	if t == nil {
		return nil
	}
	tt := t.Time
	return &tt
}
