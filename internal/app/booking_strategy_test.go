package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"
import . "github.com/Ryujoxys/sushiro-overdose/internal/proxy"
import . "github.com/Ryujoxys/sushiro-overdose/internal/api"

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testLocation(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	return loc
}

func TestUserPreferencesShouldTargetHonorsDaySpecificRanges(t *testing.T) {
	loc := testLocation(t)
	prefs := UserPreferences{
		WeekdaySlots:  []TimeRange{{Start: "1930", End: "2030"}},
		SaturdaySlots: []TimeRange{{Start: "103000", End: "130000"}},
		SundaySlots:   []TimeRange{{Start: "1800", End: "1900"}},
	}

	tests := []struct {
		name string
		slot Slot
		want bool
	}{
		{
			name: "weekday slot inside configured range",
			slot: Slot{Date: "20260515", Start: "193000", End: "200000"},
			want: true,
		},
		{
			name: "weekday slot ending exactly at range end",
			slot: Slot{Date: "20260515", Start: "200000", End: "203000"},
			want: true,
		},
		{
			name: "weekday slot start equals range end is outside",
			slot: Slot{Date: "20260515", Start: "203000", End: "210000"},
			want: false,
		},
		{
			name: "weekday slot crossing range end is outside",
			slot: Slot{Date: "20260515", Start: "200000", End: "203100"},
			want: false,
		},
		{
			name: "four digit slot time is normalized",
			slot: Slot{Date: "20260515", Start: "1930", End: "2000"},
			want: true,
		},
		{
			name: "empty slot end uses slot start",
			slot: Slot{Date: "20260515", Start: "193000"},
			want: true,
		},
		{
			name: "saturday uses saturday ranges",
			slot: Slot{Date: "20260516", Start: "103000", End: "120000"},
			want: true,
		},
		{
			name: "saturday does not use weekday ranges",
			slot: Slot{Date: "20260516", Start: "193000", End: "200000"},
			want: false,
		},
		{
			name: "sunday uses sunday ranges",
			slot: Slot{Date: "20260517", Start: "183000", End: "190000"},
			want: true,
		},
		{
			name: "invalid date is rejected",
			slot: Slot{Date: "2026-05-15", Start: "193000", End: "200000"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := prefs.ShouldTarget(tt.slot, loc); got != tt.want {
				t.Fatalf("ShouldTarget() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserPreferencesShouldTargetRejectsEmptyRanges(t *testing.T) {
	loc := testLocation(t)
	prefs := UserPreferences{WeekdaySlots: nil}

	if prefs.ShouldTarget(Slot{Date: "20260515", Start: "193000", End: "200000"}, loc) {
		t.Fatal("ShouldTarget() = true, want false for empty ranges")
	}
}

func TestNormalizePreferencesFillsStrategyDefaults(t *testing.T) {
	prefs := NormalizePreferences(UserPreferences{
		TableType:       "",
		SelectedStores:  []string{"001", "", "001", "002"},
		StorePriority:   []string{"003", "002", "002"},
		DayPriorityMode: "bad",
		DayPriority:     []string{"sunday", "sunday", "bad"},
		SlotStrategy:    "bad",
		TargetTime:      "2360",
	})

	if prefs.Adult != 2 || prefs.TableType != "T" {
		t.Fatalf("basic defaults = adult %d table %q, want 2/T", prefs.Adult, prefs.TableType)
	}
	if prefs.DayPriorityMode != DayPriorityDate {
		t.Fatalf("DayPriorityMode = %q, want %q", prefs.DayPriorityMode, DayPriorityDate)
	}
	if prefs.SlotStrategy != SlotStrategyEarliest || prefs.TargetTime != DefaultTargetTime {
		t.Fatalf("slot strategy defaults = %q/%q, want %q/%q", prefs.SlotStrategy, prefs.TargetTime, SlotStrategyEarliest, DefaultTargetTime)
	}
	if got, want := strings.Join(prefs.SelectedStores, ","), "001,002"; got != want {
		t.Fatalf("SelectedStores = %q, want %q", got, want)
	}
	if got, want := strings.Join(prefs.StorePriority, ","), "002,001"; got != want {
		t.Fatalf("StorePriority = %q, want %q", got, want)
	}
	if got, want := strings.Join(prefs.DayPriority, ","), "sunday,saturday,weekday"; got != want {
		t.Fatalf("DayPriority = %q, want %q", got, want)
	}
}

func TestNormalizePreferencesDropsPartialPhoneOverride(t *testing.T) {
	prefs := NormalizePreferences(UserPreferences{PhoneNumber: "9280"})
	if prefs.PhoneNumber != "" {
		t.Fatalf("PhoneNumber = %q, want empty partial override", prefs.PhoneNumber)
	}

	prefs = NormalizePreferences(UserPreferences{PhoneNumber: "138 0013 8000"})
	if prefs.PhoneNumber != "13800138000" {
		t.Fatalf("PhoneNumber = %q, want normalized full number", prefs.PhoneNumber)
	}
}

func TestCapturedTokensIgnorePartialPreferencePhone(t *testing.T) {
	tokens := &CapturedTokens{PhoneNumber: "13800138000"}

	settings := tokens.ToSettingsWithPrefs(UserPreferences{PhoneNumber: "9280"})
	if settings.PhoneNumber != "13800138000" {
		t.Fatalf("PhoneNumber = %q, want captured number", settings.PhoneNumber)
	}

	settings = tokens.ToSettingsWithPrefs(UserPreferences{PhoneNumber: "13900139000"})
	if settings.PhoneNumber != "13900139000" {
		t.Fatalf("PhoneNumber = %q, want full preference override", settings.PhoneNumber)
	}
}

func TestPreferTargetSlotHonorsPriorityAndTimeStrategy(t *testing.T) {
	loc := testLocation(t)

	tests := []struct {
		name       string
		prefs      UserPreferences
		candidate  TargetSlot
		current    TargetSlot
		storeOrder []string
		want       bool
	}{
		{
			name: "date mode keeps earlier date before weekend preference",
			prefs: UserPreferences{
				DayPriorityMode: DayPriorityDate,
				SlotStrategy:    SlotStrategyEarliest,
			},
			candidate: TargetSlot{StoreID: "001", Date: "20260516", Start: "103000"},
			current:   TargetSlot{StoreID: "001", Date: "20260515", Start: "193000"},
			want:      false,
		},
		{
			name: "weekend first beats earlier weekday",
			prefs: UserPreferences{
				DayPriorityMode: DayPriorityWeekendFirst,
				SlotStrategy:    SlotStrategyEarliest,
			},
			candidate: TargetSlot{StoreID: "001", Date: "20260516", Start: "103000"},
			current:   TargetSlot{StoreID: "001", Date: "20260515", Start: "193000"},
			want:      true,
		},
		{
			name: "weekday first beats earlier weekend",
			prefs: UserPreferences{
				DayPriorityMode: DayPriorityWeekdayFirst,
				SlotStrategy:    SlotStrategyEarliest,
			},
			candidate: TargetSlot{StoreID: "001", Date: "20260518", Start: "193000"},
			current:   TargetSlot{StoreID: "001", Date: "20260516", Start: "103000"},
			want:      true,
		},
		{
			name: "latest strategy prefers later time on same day",
			prefs: UserPreferences{
				DayPriorityMode: DayPriorityDate,
				SlotStrategy:    SlotStrategyLatest,
			},
			candidate: TargetSlot{StoreID: "001", Date: "20260515", Start: "203000"},
			current:   TargetSlot{StoreID: "001", Date: "20260515", Start: "193000"},
			want:      true,
		},
		{
			name: "closest strategy prefers nearest target time",
			prefs: UserPreferences{
				DayPriorityMode: DayPriorityDate,
				SlotStrategy:    SlotStrategyClosest,
				TargetTime:      "1930",
			},
			candidate: TargetSlot{StoreID: "001", Date: "20260515", Start: "194500"},
			current:   TargetSlot{StoreID: "001", Date: "20260515", Start: "180000"},
			want:      true,
		},
		{
			name: "closest strategy breaks equal distance by earlier time",
			prefs: UserPreferences{
				DayPriorityMode: DayPriorityDate,
				SlotStrategy:    SlotStrategyClosest,
				TargetTime:      "1930",
			},
			candidate: TargetSlot{StoreID: "001", Date: "20260515", Start: "190000"},
			current:   TargetSlot{StoreID: "001", Date: "20260515", Start: "200000"},
			want:      true,
		},
		{
			name: "store priority breaks same date and time",
			prefs: UserPreferences{
				DayPriorityMode: DayPriorityDate,
				SlotStrategy:    SlotStrategyEarliest,
				SelectedStores:  []string{"001", "002"},
				StorePriority:   []string{"002", "001"},
			},
			candidate:  TargetSlot{StoreID: "002", Date: "20260515", Start: "193000"},
			current:    TargetSlot{StoreID: "001", Date: "20260515", Start: "193000"},
			storeOrder: []string{"001", "002"},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.prefs.PreferTargetSlot(tt.candidate, tt.current, loc, tt.storeOrder); got != tt.want {
				t.Fatalf("PreferTargetSlot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSlotConfigShouldTargetLegacyRanges(t *testing.T) {
	loc := testLocation(t)
	cfg := SlotConfig{
		Weekday:  PrefBefore2000,
		Saturday: Pref1030to1300,
		Sunday:   PrefNone,
	}

	tests := []struct {
		name string
		slot Slot
		want bool
	}{
		{
			name: "weekday before 20:00 inclusive end",
			slot: Slot{Date: "20260515", Start: "195900", End: "200000"},
			want: true,
		},
		{
			name: "weekday start at 20:00 is outside",
			slot: Slot{Date: "20260515", Start: "200000", End: "201500"},
			want: false,
		},
		{
			name: "weekday crossing 20:00 is outside",
			slot: Slot{Date: "20260515", Start: "195900", End: "200100"},
			want: false,
		},
		{
			name: "saturday lunch range",
			slot: Slot{Date: "20260516", Start: "103000", End: "123000"},
			want: true,
		},
		{
			name: "sunday none",
			slot: Slot{Date: "20260517", Start: "103000", End: "123000"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cfg.ShouldTarget(tt.slot, loc); got != tt.want {
				t.Fatalf("ShouldTarget() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSettingsValidate(t *testing.T) {
	valid := validSettingsForTest()

	tests := []struct {
		name      string
		settings  Settings
		wantParts []string
	}{
		{
			name:     "valid settings",
			settings: valid,
		},
		{
			name: "missing required values and placeholders",
			settings: func() Settings {
				s := valid
				s.PhoneNumber = ""
				s.QueryAuthorization = "REPLACE_QUERY_AUTH"
				return s
			}(),
			wantParts: []string{"missing required config values", "phone_number", "query_authorization"},
		},
		{
			name: "missing store ids",
			settings: func() Settings {
				s := valid
				s.StoreIDs = nil
				return s
			}(),
			wantParts: []string{"at least one store ID"},
		},
		{
			name: "negative people count",
			settings: func() Settings {
				s := valid
				s.Child = -1
				return s
			}(),
			wantParts: []string{"zero or greater"},
		},
		{
			name: "zero total people",
			settings: func() Settings {
				s := valid
				s.Adult = 0
				s.Child = 0
				return s
			}(),
			wantParts: []string{"adult + child must be greater than zero"},
		},
		{
			name: "non-positive poll interval",
			settings: func() Settings {
				s := valid
				s.PollInterval = 0
				return s
			}(),
			wantParts: []string{"poll_interval_seconds must be greater than zero"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.settings.Validate()
			if len(tt.wantParts) == 0 {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}
				return
			}
			assertErrorContainsAll(t, err, tt.wantParts...)
		})
	}
}

func TestIsNoReservationText(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{text: "E044", want: true},
		{text: "NO_MORE_RESERVATIONS", want: true},
		{text: "no reservation available", want: true},
		{text: "名额已满", want: true},
		{text: "当前已满", want: true},
		{text: "token expired", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			if got := IsNoReservationText(tt.text); got != tt.want {
				t.Fatalf("IsNoReservationText(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestIsActiveReservationText(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{text: "E052", want: true},
		{text: "You can only make one reservation at a time.", want: true},
		{text: "すでにご予約をいただいております。", want: true},
		{text: "已经有预约", want: true},
		{text: "名额已满", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			if got := IsActiveReservationText(tt.text); got != tt.want {
				t.Fatalf("IsActiveReservationText(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestReservationBusinessError(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want error
	}{
		{
			name: "plain text no reservation code",
			body: []byte("E044"),
			want: ErrNoReservationAvailable,
		},
		{
			name: "json code no reservation",
			body: []byte(`{"code":"E044","message":"full"}`),
			want: ErrNoReservationAvailable,
		},
		{
			name: "json snake error code no reservation",
			body: []byte(`{"error_code":"NO_MORE_RESERVATIONS"}`),
			want: ErrNoReservationAvailable,
		},
		{
			name: "json chinese message no reservation",
			body: []byte(`{"message":"名额已满"}`),
			want: ErrNoReservationAvailable,
		},
		{
			name: "json active reservation exists",
			body: []byte(`{"code":"E052","message":"You can only make one reservation at a time."}`),
			want: ErrActiveReservationExists,
		},
		{
			name: "unknown business response",
			body: []byte(`{"code":"OK","message":"queued"}`),
			want: nil,
		},
		{
			name: "invalid json without known text",
			body: []byte("service unavailable"),
			want: nil,
		},
		{
			name: "empty body",
			body: nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ReservationBusinessError(tt.body)
			if !errors.Is(err, tt.want) {
				t.Fatalf("ReservationBusinessError() = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCreateReservationMapsActiveReservationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wechat/api_auth/2.0/ticketing/createReservation" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"code":"E052","message":"You can only make one reservation at a time."}`))
	}))
	defer server.Close()

	settings := validSettingsForTest()
	settings.BaseURL = server.URL
	client := NewClient(settings)
	client.SetHTTPClient(server.Client())

	_, err := client.CreateReservation(context.Background(), "001", "20260515", "193000")
	if !errors.Is(err, ErrActiveReservationExists) {
		t.Fatalf("CreateReservation() error = %v, want %v", err, ErrActiveReservationExists)
	}
}

func TestCreateReservationMapsBusinessError(t *testing.T) {
	var gotPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wechat/api_auth/2.0/ticketing/createReservation" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"E044","message":"名额已满"}`))
	}))
	defer server.Close()

	settings := validSettingsForTest()
	settings.BaseURL = server.URL
	client := NewClient(settings)
	client.SetHTTPClient(server.Client())

	_, err := client.CreateReservation(context.Background(), "001", "20260515", "193000")
	if !errors.Is(err, ErrNoReservationAvailable) {
		t.Fatalf("CreateReservation() error = %v, want %v", err, ErrNoReservationAvailable)
	}
	if gotPayload["storeId"] != "001" || gotPayload["date"] != "20260515" || gotPayload["time"] != "193000" {
		t.Fatalf("reservation payload = %#v", gotPayload)
	}
}

func TestGetNetTicketStatusParsesCurrentTicket(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wechat/api_auth/2.0/ticket/status" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("wechatId"); got != "wechat-id" {
			t.Fatalf("wechatId = %q, want wechat-id", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"netTicket":{"STORE_INFO":{"id":3006,"name":"太阳宫凯德店","address":"太阳宫中路"},"TICKET_DETAIL":{"ticketId":9876,"number":"1167","storeId":"3006","wait":33}}}`))
	}))
	defer server.Close()

	settings := validSettingsForTest()
	settings.BaseURL = server.URL
	settings.WechatID = "wechat-id"
	client := NewClient(settings)
	client.SetHTTPClient(server.Client())

	ticket, err := client.GetNetTicketStatus(context.Background())
	if err != nil {
		t.Fatalf("GetNetTicketStatus() error = %v", err)
	}
	if ticket.Number != "1167" || ticket.TicketID != 9876 || ticket.StoreID != "3006" || ticket.Wait != 33 || ticket.StoreName != "太阳宫凯德店" {
		t.Fatalf("ticket = %#v", ticket)
	}
}

func TestCapturedTokensValidation(t *testing.T) {
	queryOnly := &CapturedTokens{
		XAppCode:  "app",
		QueryAuth: "query-auth",
		UserAgent: "agent",
		Referer:   "referer",
		StoreIDs:  []string{"001"},
	}
	if err := queryOnly.ValidateForQuery(); err != nil {
		t.Fatalf("ValidateForQuery() error = %v, want nil", err)
	}
	assertErrorContainsAll(t, queryOnly.ValidateForReservation(), "预约凭证", "微信ID", "手机号")

	complete := &CapturedTokens{
		XAppCode:        "app",
		QueryAuth:       "query-auth",
		ReservationAuth: "reservation-auth",
		UserAgent:       "agent",
		Referer:         "referer",
		WechatID:        "wechat",
		PhoneNumber:     "13800138000",
		StoreIDs:        []string{"001"},
	}
	if !complete.IsComplete() {
		t.Fatal("IsComplete() = false, want true")
	}
	if err := complete.ValidateForReservation(); err != nil {
		t.Fatalf("ValidateForReservation() error = %v, want nil", err)
	}

	missingQuery := &CapturedTokens{
		XAppCode:  "  ",
		UserAgent: "agent",
	}
	assertErrorContainsAll(t, missingQuery.ValidateForQuery(), "X-App-Code", "查询凭证", "Referer", "门店")
}

func validSettingsForTest() Settings {
	return Settings{
		StoreIDs:           []string{"001"},
		Adult:              2,
		Child:              0,
		TableType:          "T",
		PhoneNumber:        "13800138000",
		WechatID:           "wechat",
		XAppCode:           "app",
		QueryAuthorization: "query-auth",
		ReservationAuth:    "reservation-auth",
		XAppClient:         "miniapp",
		UserAgent:          "agent",
		Referer:            "referer",
		Timezone:           "Asia/Shanghai",
		Location:           time.FixedZone("CST", 8*60*60),
		PollInterval:       time.Second,
		AvailableStatuses:  map[string]struct{}{"AVAILABLE": {}},
		BaseURL:            "https://example.test",
	}
}

func assertErrorContainsAll(t *testing.T, err error, parts ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want containing %v", parts)
	}
	msg := err.Error()
	for _, part := range parts {
		if !strings.Contains(msg, part) {
			t.Fatalf("error = %q, want containing %q", msg, part)
		}
	}
}
