package controller

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	palworldv1alpha1 "github.com/DataKnifeAI/palworld-operator/api/v1alpha1"
)

// notifySleep is time.Sleep; tests may replace it to avoid real delays.
var notifySleep = time.Sleep

// notifyStage is one warning before plannedApplyTime (T=0).
type notifyStage struct {
	Key string
	At  time.Duration
}

// defaultNotifyScheduleKeys is the default multi-stage warning timeline.
var defaultNotifyScheduleKeys = []string{"60m", "30m", "15m", "5m", "1m", "30s", "10s"}

func parseNotifySchedule(spec palworldv1alpha1.PalworldServerSpec) []notifyStage {
	keys := spec.Update.NotifySchedule
	if len(keys) == 0 {
		if spec.Update.NotifyLeadTime != "" {
			keys = []string{spec.Update.NotifyLeadTime}
		} else {
			keys = append([]string(nil), defaultNotifyScheduleKeys...)
		}
	}
	stages := make([]notifyStage, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		d, err := time.ParseDuration(key)
		if err != nil || d < 0 {
			continue
		}
		// Normalize key to ParseDuration's string form for stable status tracking.
		norm := d.String()
		if _, ok := seen[norm]; ok {
			continue
		}
		seen[norm] = struct{}{}
		stages = append(stages, notifyStage{Key: norm, At: d})
	}
	if len(stages) == 0 {
		for _, key := range defaultNotifyScheduleKeys {
			d, _ := time.ParseDuration(key)
			stages = append(stages, notifyStage{Key: d.String(), At: d})
		}
	}
	sort.Slice(stages, func(i, j int) bool { return stages[i].At > stages[j].At })
	return stages
}

func maxNotifyLead(stages []notifyStage) time.Duration {
	if len(stages) == 0 {
		return defaultNotifyLeadTime
	}
	return stages[0].At // sorted descending
}

func announcedSet(keys []string) map[string]struct{} {
	out := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		out[k] = struct{}{}
	}
	return out
}

// nextDueNotifyStage returns the tightest (smallest) unannounced stage that is
// due (remaining <= stage.At). Larger overdue stages are returned in skip so
// the caller can mark them announced without spamming catch-up messages.
func nextDueNotifyStage(stages []notifyStage, remaining time.Duration, announced []string) (due notifyStage, skip []string, ok bool) {
	done := announcedSet(announced)
	// stages are sorted descending; walk ascending to find the tightest due stage.
	for i := len(stages) - 1; i >= 0; i-- {
		s := stages[i]
		if _, seen := done[s.Key]; seen {
			continue
		}
		if remaining > s.At {
			continue
		}
		due = s
		ok = true
		break
	}
	if !ok {
		return notifyStage{}, nil, false
	}
	for _, s := range stages {
		if s.At <= due.At {
			continue
		}
		if _, seen := done[s.Key]; seen {
			continue
		}
		if remaining <= s.At {
			skip = append(skip, s.Key)
		}
	}
	return due, skip, true
}

// requeueUntilNextNotifyStage returns how long to wait until the next unannounced
// stage becomes due, or until planned apply if none remain.
func requeueUntilNextNotifyStage(stages []notifyStage, remaining time.Duration, announced []string) time.Duration {
	if remaining <= 0 {
		return 0
	}
	done := announcedSet(announced)
	soonest := remaining
	for _, s := range stages {
		if _, seen := done[s.Key]; seen {
			continue
		}
		wait := remaining - s.At
		if wait <= 0 {
			return time.Second
		}
		if wait < soonest {
			soonest = wait
		}
	}
	if soonest < time.Second {
		return time.Second
	}
	return soonest
}

func humanizeDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	d = d.Round(time.Second)
	if d >= time.Hour {
		h := int(d / time.Hour)
		m := int((d % time.Hour) / time.Minute)
		if m == 0 {
			if h == 1 {
				return "1 hour"
			}
			return fmt.Sprintf("%d hours", h)
		}
		return fmt.Sprintf("%dh%dm", h, m)
	}
	if d >= time.Minute {
		m := int(d / time.Minute)
		sec := int((d % time.Minute) / time.Second)
		if sec == 0 {
			if m == 1 {
				return "1 minute"
			}
			return fmt.Sprintf("%d minutes", m)
		}
		return fmt.Sprintf("%dm%ds", m, sec)
	}
	sec := int(d / time.Second)
	if sec == 1 {
		return "1 second"
	}
	return fmt.Sprintf("%d seconds", sec)
}

func defaultStageMessage(version string, remaining time.Duration, initial bool) string {
	rem := humanizeDuration(remaining)
	if remaining <= 10*time.Second {
		return fmt.Sprintf("[Server] Restarting in %s for a game update (%s).", rem, version)
	}
	if initial {
		return fmt.Sprintf("[Server] Update scheduled — restart in %s for a game update (%s).", rem, version)
	}
	return fmt.Sprintf("[Server] Reminder — restart in %s for a game update (%s).", rem, version)
}

func formatStageMessage(spec palworldv1alpha1.PalworldServerSpec, version, image string, remaining time.Duration, initial bool) string {
	if msg := spec.Update.NotifyMessage; msg != "" {
		return strings.NewReplacer(
			"{version}", version,
			"{image}", image,
			"{remaining}", humanizeDuration(remaining),
		).Replace(msg)
	}
	return defaultStageMessage(version, remaining, initial)
}

// nextApplyInstant returns the next applySchedule fire (or now when unset).
func nextApplyInstant(spec palworldv1alpha1.PalworldServerSpec, now time.Time) (time.Time, error) {
	expr := spec.Update.ApplySchedule
	if expr == "" {
		return now, nil
	}
	loc, err := updateTimeZone(spec)
	if err != nil {
		return time.Time{}, err
	}
	sched, err := parseCronExpr(expr)
	if err != nil {
		return time.Time{}, fmt.Errorf("spec.update.applySchedule: %w", err)
	}
	local := now.In(loc)
	if cronMatchesMinute(sched, local) {
		return time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), local.Minute(), 0, 0, loc), nil
	}
	return sched.Next(local), nil
}

// computePlannedApplyTime picks T=0 for a new notify sequence.
// Without applySchedule: now + max(schedule).
// With applySchedule: next cron fire, or now+lead when the window is sooner than the full lead
// (so players still get warnings; apply may land slightly after the cron minute).
func computePlannedApplyTime(spec palworldv1alpha1.PalworldServerSpec, now time.Time, stages []notifyStage) (time.Time, error) {
	lead := maxNotifyLead(stages)
	if lead <= 0 {
		lead = time.Second
	}
	next, err := nextApplyInstant(spec, now)
	if err != nil {
		return time.Time{}, err
	}
	if spec.Update.ApplySchedule == "" {
		return now.Add(lead), nil
	}
	if next.Before(now) {
		next = now
	}
	if next.Sub(now) < lead {
		return now.Add(lead), nil
	}
	return next, nil
}

func appendUniqueStages(existing []string, keys ...string) []string {
	seen := announcedSet(existing)
	out := append([]string(nil), existing...)
	for _, k := range keys {
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	return out
}

func clearNotifyStatus(server *palworldv1alpha1.PalworldServer) {
	server.Status.PendingUpdateImage = ""
	server.Status.PlannedApplyTime = nil
	server.Status.AnnouncedNotifyStages = nil
	server.Status.LastAnnounceTime = nil
}

func isCountdownStage(s notifyStage) bool {
	return s.At <= 10*time.Second && s.At > 0
}

// runFinalCountdown sends a short 10→1 style announce sequence without hammering REST.
// Best-effort: one "10 seconds" message, then 5..1 at ~1s intervals.
func (r *PalworldServerReconciler) runFinalCountdown(
	ctx context.Context,
	baseURL, adminPassword, version string,
) error {
	log := logf.FromContext(ctx)
	msgs := []string{
		fmt.Sprintf("[Server] Restarting in 10 seconds for a game update (%s).", version),
		"[Server] Restart in 5…",
		"[Server] Restart in 4…",
		"[Server] Restart in 3…",
		"[Server] Restart in 2…",
		"[Server] Restart in 1…",
	}
	for i, msg := range msgs {
		if err := r.restClient().Announce(ctx, baseURL, adminPassword, msg); err != nil {
			return err
		}
		log.V(1).Info("countdown announce", "step", i+1, "message", msg)
		if i < len(msgs)-1 {
			notifySleep(time.Second)
		}
	}
	return nil
}

// processNotifySchedule sends at most one due stage (or a final countdown), updates
// status, and returns requeue delay. readyToApply is true when T<=0 and countdown done.
func (r *PalworldServerReconciler) processNotifySchedule(
	ctx context.Context,
	server *palworldv1alpha1.PalworldServer,
	names derivedNames,
	adminPassword, version, targetImage string,
	now time.Time,
) (requeueAfter time.Duration, readyToApply bool, err error) {
	log := logf.FromContext(ctx)
	stages := parseNotifySchedule(server.Spec)

	if server.Status.PendingUpdateImage != targetImage || server.Status.PlannedApplyTime == nil {
		planned, planErr := computePlannedApplyTime(server.Spec, now, stages)
		if planErr != nil {
			return 0, false, planErr
		}
		server.Status.PendingUpdateImage = targetImage
		t := metav1.NewTime(planned)
		server.Status.PlannedApplyTime = &t
		server.Status.AnnouncedNotifyStages = nil
		server.Status.LastAnnounceTime = nil
		server.Status.Message = fmt.Sprintf("update to %s planned at %s", version, planned.UTC().Format(time.RFC3339))
		log.Info("planned image update", "image", targetImage, "plannedApplyTime", planned)
	}

	planned := server.Status.PlannedApplyTime.Time
	remaining := planned.Sub(now)

	if !restEnabled(server.Spec) || adminPassword == "" {
		return 0, false, fmt.Errorf("notifyPlayers requires REST API and admin password")
	}
	base := restBaseURL(names.serviceName, server.Namespace, restPort(server.Spec))

	if remaining > 0 {
		due, skip, ok := nextDueNotifyStage(stages, remaining, server.Status.AnnouncedNotifyStages)
		if len(skip) > 0 {
			server.Status.AnnouncedNotifyStages = appendUniqueStages(server.Status.AnnouncedNotifyStages, skip...)
		}
		if ok {
			if isCountdownStage(due) {
				// Defer countdown until we are ready to apply at T≈0 so we do not
				// burn the final window while onlyWhenEmpty still blocks later.
				server.Status.Message = fmt.Sprintf("update to %s; final countdown at apply (%s remaining)", version, humanizeDuration(remaining))
			} else {
				initial := len(server.Status.AnnouncedNotifyStages) == 0
				msg := formatStageMessage(server.Spec, version, targetImage, remaining, initial)
				if annErr := r.restClient().Announce(ctx, base, adminPassword, msg); annErr != nil {
					log.Error(annErr, "pre-update announce failed", "stage", due.Key)
					server.Status.Message = fmt.Sprintf("update announce failed (%s): %v", due.Key, annErr)
					return updateRequeueBusy, false, nil
				}
				announced := metav1.NewTime(now)
				server.Status.LastAnnounceTime = &announced
				server.Status.AnnouncedNotifyStages = appendUniqueStages(server.Status.AnnouncedNotifyStages, due.Key)
				server.Status.Message = fmt.Sprintf("announced update to %s (%s remaining)", version, humanizeDuration(remaining))
			}
		}
		rq := requeueUntilNextNotifyStage(stages, remaining, server.Status.AnnouncedNotifyStages)
		// Also wake near planned apply.
		if untilApply := remaining; untilApply > 0 && (rq == 0 || untilApply < rq) {
			rq = untilApply
		}
		if rq <= 0 {
			rq = time.Second
		}
		return rq, false, nil
	}

	// T<=0: warnings done; caller applies when onlyWhenEmpty allows.
	// Final countdown runs immediately before the image patch (not here) so we
	// do not announce a restart that onlyWhenEmpty then defers.
	server.Status.Message = fmt.Sprintf("notify complete; ready to apply update to %s", version)
	return 0, true, nil
}

// maybeFinalCountdown runs the short 10→1 announce once when apply is imminent.
func (r *PalworldServerReconciler) maybeFinalCountdown(
	ctx context.Context,
	server *palworldv1alpha1.PalworldServer,
	names derivedNames,
	adminPassword, version string,
	now time.Time,
) error {
	stages := parseNotifySchedule(server.Spec)
	var countdown *notifyStage
	for i := range stages {
		if isCountdownStage(stages[i]) {
			countdown = &stages[i]
			break
		}
	}
	if countdown == nil {
		return nil
	}
	if _, seen := announcedSet(server.Status.AnnouncedNotifyStages)[countdown.Key]; seen {
		return nil
	}
	if !restEnabled(server.Spec) || adminPassword == "" {
		return fmt.Errorf("notifyPlayers requires REST API and admin password")
	}
	base := restBaseURL(names.serviceName, server.Namespace, restPort(server.Spec))
	if err := r.runFinalCountdown(ctx, base, adminPassword, version); err != nil {
		return err
	}
	announced := metav1.NewTime(now)
	server.Status.LastAnnounceTime = &announced
	server.Status.AnnouncedNotifyStages = appendUniqueStages(server.Status.AnnouncedNotifyStages, countdown.Key)
	return nil
}
