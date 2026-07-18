package controller

import (
	"testing"
	"time"

	palworldv1alpha1 "github.com/DataKnifeAI/palworld-operator/api/v1alpha1"
)

func TestCronMatchesMinute(t *testing.T) {
	sched, err := parseCronExpr("0 4 * * 1-5")
	if err != nil {
		t.Fatal(err)
	}
	// Monday 2026-07-13 04:00 UTC
	mon := time.Date(2026, 7, 13, 4, 0, 15, 0, time.UTC)
	if !cronMatchesMinute(sched, mon) {
		t.Fatal("expected Monday 04:00 to match")
	}
	tueWrong := time.Date(2026, 7, 14, 5, 0, 0, 0, time.UTC)
	if cronMatchesMinute(sched, tueWrong) {
		t.Fatal("expected Tuesday 05:00 not to match")
	}
}

func TestInApplyWindow(t *testing.T) {
	spec := palworldv1alpha1.PalworldServerSpec{}
	ok, err := inApplyWindow(spec, time.Now())
	if err != nil || !ok {
		t.Fatalf("unset applySchedule should always allow: ok=%v err=%v", ok, err)
	}

	spec.Update.ApplySchedule = "0 4 * * *"
	spec.Update.TimeZone = defaultUpdateTimeZone
	at := time.Date(2026, 7, 18, 4, 0, 0, 0, time.UTC)
	ok, err = inApplyWindow(spec, at)
	if err != nil || !ok {
		t.Fatalf("expected inside window: ok=%v err=%v", ok, err)
	}
	outside := time.Date(2026, 7, 18, 5, 0, 0, 0, time.UTC)
	ok, err = inApplyWindow(spec, outside)
	if err != nil || ok {
		t.Fatalf("expected outside window: ok=%v err=%v", ok, err)
	}
}

func TestShouldCheckRegistryIntervalAndCron(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	spec := palworldv1alpha1.PalworldServerSpec{
		Update: palworldv1alpha1.UpdateConfig{CheckInterval: "6h"},
	}
	due, err := shouldCheckRegistry(spec, nil, now)
	if err != nil || !due {
		t.Fatalf("nil lastCheck should be due: %v %v", due, err)
	}
	recent := now.Add(-time.Hour)
	due, err = shouldCheckRegistry(spec, &recent, now)
	if err != nil || due {
		t.Fatalf("recent check should not be due: %v %v", due, err)
	}
	old := now.Add(-7 * time.Hour)
	due, err = shouldCheckRegistry(spec, &old, now)
	if err != nil || !due {
		t.Fatalf("old check should be due: %v %v", due, err)
	}

	spec.Update.CheckSchedule = "0 12 * * *"
	spec.Update.TimeZone = defaultUpdateTimeZone
	due, err = shouldCheckRegistry(spec, nil, now)
	if err != nil || !due {
		t.Fatalf("cron match should be due: %v %v", due, err)
	}
	sameMinute := now
	due, err = shouldCheckRegistry(spec, &sameMinute, now)
	if err != nil || due {
		t.Fatalf("already checked this minute: %v %v", due, err)
	}
}

func TestFormatNotifyMessage(t *testing.T) {
	spec := palworldv1alpha1.PalworldServerSpec{}
	msg := formatNotifyMessage(spec, testPalVersion101, "ghcr.io/pocketpairjp/palserver:"+testPalVersion101)
	if msg == "" || msg == "{version}" {
		t.Fatalf("unexpected default message %q", msg)
	}
	spec.Update.NotifyMessage = "Updating to {version}"
	if got := formatNotifyMessage(spec, testPalVersion101, "x"); got != "Updating to "+testPalVersion101 {
		t.Fatalf("got %q", got)
	}
}

func TestBuildGameUserSettingsINI(t *testing.T) {
	body := buildGameUserSettingsINI("ABCD-1234")
	want := "[/Script/Pal.PalGameLocalSettings]\nDedicatedServerName=ABCD-1234\n"
	if body != want {
		t.Fatalf("got %q want %q", body, want)
	}
}
