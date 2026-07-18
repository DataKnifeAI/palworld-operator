package controller

import (
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	palworldv1alpha1 "github.com/DataKnifeAI/palworld-operator/api/v1alpha1"
)

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

func updateTimeZone(spec palworldv1alpha1.PalworldServerSpec) (*time.Location, error) {
	name := spec.Update.TimeZone
	if name == "" {
		name = "UTC"
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("spec.update.timeZone %q: %w", name, err)
	}
	return loc, nil
}

func parseCronExpr(expr string) (cron.Schedule, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("empty cron expression")
	}
	sched, err := cronParser.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("cron %q: %w", expr, err)
	}
	return sched, nil
}

// cronMatchesMinute reports whether sched fires at the given minute (second truncated).
func cronMatchesMinute(sched cron.Schedule, t time.Time) bool {
	start := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, t.Location())
	next := sched.Next(start.Add(-time.Second))
	return next.Equal(start)
}

// inApplyWindow returns true when applySchedule is unset or matches now in the CR timezone.
func inApplyWindow(spec palworldv1alpha1.PalworldServerSpec, now time.Time) (bool, error) {
	expr := spec.Update.ApplySchedule
	if expr == "" {
		return true, nil
	}
	loc, err := updateTimeZone(spec)
	if err != nil {
		return false, err
	}
	sched, err := parseCronExpr(expr)
	if err != nil {
		return false, fmt.Errorf("spec.update.applySchedule: %w", err)
	}
	return cronMatchesMinute(sched, now.In(loc)), nil
}

// shouldCheckRegistry decides whether a registry poll is due.
// When checkSchedule is set, true only if the cron matches the current minute
// and we have not already checked within this minute.
// Otherwise uses checkInterval since lastCheck.
func shouldCheckRegistry(spec palworldv1alpha1.PalworldServerSpec, lastCheck *time.Time, now time.Time) (bool, error) {
	if expr := spec.Update.CheckSchedule; expr != "" {
		loc, err := updateTimeZone(spec)
		if err != nil {
			return false, err
		}
		sched, err := parseCronExpr(expr)
		if err != nil {
			return false, fmt.Errorf("spec.update.checkSchedule: %w", err)
		}
		local := now.In(loc)
		if !cronMatchesMinute(sched, local) {
			return false, nil
		}
		if lastCheck != nil {
			prev := lastCheck.In(loc)
			if prev.Year() == local.Year() && prev.YearDay() == local.YearDay() &&
				prev.Hour() == local.Hour() && prev.Minute() == local.Minute() {
				return false, nil
			}
		}
		return true, nil
	}

	if lastCheck == nil {
		return true, nil
	}
	return now.Sub(*lastCheck) >= updateCheckInterval(spec), nil
}

func notifyLeadTime(spec palworldv1alpha1.PalworldServerSpec) time.Duration {
	if spec.Update.NotifyLeadTime == "" {
		return defaultNotifyLeadTime
	}
	d, err := time.ParseDuration(spec.Update.NotifyLeadTime)
	if err != nil || d < 0 {
		return defaultNotifyLeadTime
	}
	return d
}

func defaultNotifyMessage(version, image string) string {
	return fmt.Sprintf("Server update to %s starting soon — please disconnect. Image: %s", version, image)
}

func formatNotifyMessage(spec palworldv1alpha1.PalworldServerSpec, version, image string) string {
	msg := spec.Update.NotifyMessage
	if msg == "" {
		return defaultNotifyMessage(version, image)
	}
	return strings.NewReplacer(
		"{version}", version,
		"{image}", image,
	).Replace(msg)
}
