package core

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestUpdateStatePreservesConcurrentTicketSlots(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	for round := 0; round < 30; round++ {
		if err := ClearState(path); err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		start := make(chan struct{})
		for _, kind := range []string{"reservation", "net_ticket"} {
			wg.Add(1)
			go func(kind string) {
				defer wg.Done()
				<-start
				if err := UpdateState(path, func(s *State) {
					rec := &ReservationRecord{Kind: kind, Number: kind}
					if kind == "reservation" {
						s.ActiveReservation = rec
					} else {
						s.ActiveNetTicket = rec
					}
				}); err != nil {
					t.Error(err)
				}
			}(kind)
		}
		close(start)
		wg.Wait()
		state, err := LoadState(path)
		if err != nil || state.ActiveReservation == nil || state.ActiveNetTicket == nil {
			t.Fatalf("lost slot: %+v, %v", state, err)
		}
	}
}

func TestUpdateStateRejectsCorruptState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := UpdateState(path, func(s *State) { s.ActiveNetTicket = &ReservationRecord{Number: "1"} }); err == nil {
		t.Fatal("corrupt state must not be silently replaced")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "broken" {
		t.Fatal("corrupt state was overwritten")
	}
}

func TestCapturedSettingsAlwaysHaveStoreTimezone(t *testing.T) {
	t.Setenv("ZONEINFO", filepath.Join(t.TempDir(), "missing"))
	s := NewCapturedTokens().ToSettingsWithPrefs(UserPreferences{})
	if s.Location != SushiroTimezone {
		t.Fatal("settings must use the embedded store timezone")
	}
	_, offset := time.Date(2026, 9, 16, 12, 0, 0, 0, s.Location).Zone()
	if offset != 8*60*60 {
		t.Fatalf("store timezone offset = %d", offset)
	}
}
