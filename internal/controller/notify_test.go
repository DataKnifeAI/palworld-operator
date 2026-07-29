package controller

import (
	"strings"
	"testing"
	"time"

	palworldv1alpha1 "github.com/DataKnifeAI/palworld-operator/api/v1alpha1"
)

func TestParseNotifyScheduleDefaultAndLeadTime(t *testing.T) {
	stages := parseNotifySchedule(palworldv1alpha1.PalworldServerSpec{})
	if len(stages) != len(defaultNotifyScheduleKeys) {
		t.Fatalf("default stages=%d want %d", len(stages), len(defaultNotifyScheduleKeys))
	}
	if stages[0].At != time.Hour {
		t.Fatalf("max lead want 1h got %s", stages[0].At)
	}

	spec := palworldv1alpha1.PalworldServerSpec{
		Update: palworldv1alpha1.UpdateConfig{NotifyLeadTime: "2m"},
	}
	stages = parseNotifySchedule(spec)
	if len(stages) != 1 || stages[0].At != 2*time.Minute {
		t.Fatalf("leadTime single stage: %+v", stages)
	}

	spec.Update.NotifySchedule = []string{"30m", "5m", "10s"}
	stages = parseNotifySchedule(spec)
	if len(stages) != 3 || stages[0].At != 30*time.Minute {
		t.Fatalf("explicit schedule: %+v", stages)
	}
}

func TestNextDueNotifyStageCatchUp(t *testing.T) {
	stages := parseNotifySchedule(palworldv1alpha1.PalworldServerSpec{
		Update: palworldv1alpha1.UpdateConfig{
			NotifySchedule: []string{"60m", "30m", "15m", "5m", "1m", "30s", "10s"},
		},
	})

	// Mid-schedule join: 4m remaining → announce 5m, skip 60/30/15.
	due, skip, ok := nextDueNotifyStage(stages, 4*time.Minute, nil)
	if !ok || due.At != 5*time.Minute {
		t.Fatalf("due=%+v ok=%v", due, ok)
	}
	if len(skip) != 3 {
		t.Fatalf("skip=%v want 3 larger stages", skip)
	}

	announced := appendUniqueStages(skip, due.Key)
	due, skip, ok = nextDueNotifyStage(stages, 4*time.Minute, announced)
	if ok {
		t.Fatalf("expected no due stage yet, got %+v skip=%v", due, skip)
	}

	due, _, ok = nextDueNotifyStage(stages, time.Minute, announced)
	if !ok || due.At != time.Minute {
		t.Fatalf("1m due: %+v ok=%v", due, ok)
	}
}

func TestNextDueNotifyStageExactBoundary(t *testing.T) {
	stages := parseNotifySchedule(palworldv1alpha1.PalworldServerSpec{
		Update: palworldv1alpha1.UpdateConfig{NotifySchedule: []string{"60m", "30m"}},
	})
	due, skip, ok := nextDueNotifyStage(stages, 60*time.Minute, nil)
	if !ok || due.At != time.Hour || len(skip) != 0 {
		t.Fatalf("at T-60m: due=%+v skip=%v ok=%v", due, skip, ok)
	}
	due, _, ok = nextDueNotifyStage(stages, 60*time.Minute+time.Second, nil)
	if ok {
		t.Fatalf("before first stage should not be due, got %+v", due)
	}
}

func TestRequeueUntilNextNotifyStage(t *testing.T) {
	stages := parseNotifySchedule(palworldv1alpha1.PalworldServerSpec{
		Update: palworldv1alpha1.UpdateConfig{NotifySchedule: []string{"60m", "30m", "10s"}},
	})
	// After 60m announced, 50m remaining → next is 30m in 20m.
	rq := requeueUntilNextNotifyStage(stages, 50*time.Minute, []string{(time.Hour).String()})
	if rq != 20*time.Minute {
		t.Fatalf("requeue=%s want 20m", rq)
	}
	if got := requeueUntilNextNotifyStage(stages, 0, nil); got != 0 {
		t.Fatalf("at T=0 requeue=%s", got)
	}
}

func TestComputePlannedApplyTime(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	stages := parseNotifySchedule(palworldv1alpha1.PalworldServerSpec{})
	spec := palworldv1alpha1.PalworldServerSpec{}
	planned, err := computePlannedApplyTime(spec, now, stages)
	if err != nil {
		t.Fatal(err)
	}
	if !planned.Equal(now.Add(time.Hour)) {
		t.Fatalf("no cron: planned=%s want %s", planned, now.Add(time.Hour))
	}

	spec.Update.ApplySchedule = "0 4 * * *"
	spec.Update.TimeZone = defaultUpdateTimeZone
	// Next 04:00 is tomorrow → use that (more than 1h away).
	planned, err = computePlannedApplyTime(spec, now, stages)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 7, 19, 4, 0, 0, 0, time.UTC)
	if !planned.Equal(want) {
		t.Fatalf("cron far: planned=%s want %s", planned, want)
	}

	// Inside a window sooner than lead → now+lead.
	atWindow := time.Date(2026, 7, 18, 4, 0, 30, 0, time.UTC)
	planned, err = computePlannedApplyTime(spec, atWindow, stages)
	if err != nil {
		t.Fatal(err)
	}
	if !planned.Equal(atWindow.Add(time.Hour)) {
		t.Fatalf("cron soon: planned=%s want %s", planned, atWindow.Add(time.Hour))
	}
}

func TestHumanizeAndStageMessage(t *testing.T) {
	if got := humanizeDuration(time.Hour); got != "1 hour" {
		t.Fatalf("humanize hour: %q", got)
	}
	if got := humanizeDuration(30 * time.Minute); got != "30 minutes" {
		t.Fatalf("humanize 30m: %q", got)
	}
	msg := defaultStageMessage(testPalVersion101, 30*time.Minute, false)
	for _, part := range []string{"Reminder", "30 minutes", testPalVersion101} {
		if !strings.Contains(msg, part) {
			t.Fatalf("message=%q missing %q", msg, part)
		}
	}
	spec := palworldv1alpha1.PalworldServerSpec{
		Update: palworldv1alpha1.UpdateConfig{NotifyMessage: "Update {version} in {remaining}"},
	}
	got := formatStageMessage(spec, testPalVersion101, "img", 5*time.Minute, true)
	if got != "Update "+testPalVersion101+" in 5 minutes" {
		t.Fatalf("template=%q", got)
	}
}
