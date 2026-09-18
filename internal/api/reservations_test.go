package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

func TestGetReservationsTreatsNotFoundAsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wechat/api_auth/2.0/ticketing/getReservations" {
			t.Fatalf("path = %q, want getReservations", r.URL.Path)
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"Not Found"}`))
	}))
	defer server.Close()

	client := NewClient(Settings{
		BaseURL:         server.URL,
		ReservationAuth: "reservation-auth",
		XAppCode:        "app-code",
		XAppClient:      "miniapp",
		UserAgent:       "ua",
		Referer:         "ref",
		WechatID:        "wechat",
		PhoneNumber:     "phone",
	})

	_, err := client.GetReservations(context.Background())
	if !errors.Is(err, ErrReservationsEndpointUnavailable) {
		t.Fatalf("GetReservations() error = %v, want ErrReservationsEndpointUnavailable", err)
	}
}

func TestGetReservationsMarksRecordsAsReservation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wechat/api_auth/2.0/ticketing/getReservations" {
			t.Fatalf("path = %q, want getReservations", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"ticketId":123,"number":"A001","status":"RESERVED"}]`))
	}))
	defer server.Close()

	client := NewClient(Settings{
		BaseURL:         server.URL,
		ReservationAuth: "reservation-auth",
		XAppCode:        "app-code",
		XAppClient:      "miniapp",
		UserAgent:       "ua",
		Referer:         "ref",
		WechatID:        "wechat",
		PhoneNumber:     "phone",
	})

	records, err := client.GetReservations(context.Background())
	if err != nil {
		t.Fatalf("GetReservations() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("records len = %d, want 1", len(records))
	}
	if records[0].Kind != "reservation" {
		t.Fatalf("record kind = %q, want reservation", records[0].Kind)
	}
}

func TestGetReservationsAcceptsEmptyWrappedData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wechat/api_auth/2.0/ticketing/getReservations" {
			t.Fatalf("path = %q, want getReservations", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	client := NewClient(Settings{
		BaseURL:         server.URL,
		ReservationAuth: "reservation-auth",
		XAppCode:        "app-code",
		XAppClient:      "miniapp",
		UserAgent:       "ua",
		Referer:         "ref",
		WechatID:        "wechat",
		PhoneNumber:     "phone",
	})

	records, err := client.GetReservations(context.Background())
	if err != nil {
		t.Fatalf("GetReservations() error = %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("records len = %d, want 0", len(records))
	}
}

func TestGetReservationsRejectsUnknownOrNullResponses(t *testing.T) {
	for _, body := range []string{`{}`, `{"error":"expired"}`, `null`, `{"data":null}`, `{"data":{}}`, `"unexpected"`, ``} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(body))
			}))
			defer server.Close()
			client := NewClient(Settings{BaseURL: server.URL})
			if records, err := client.GetReservations(context.Background()); err == nil {
				t.Fatalf("unknown response treated as a reservation list: %+v", records)
			}
		})
	}
}

func TestGetNetTicketStatusMarksNetTicketKind(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wechat/api_auth/2.0/ticket/status" {
			t.Fatalf("path = %q, want ticket/status", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"netTicket":{"ticketId":456,"number":"1843","status":"WAITING","wait":1182}}`))
	}))
	defer server.Close()

	client := NewClient(Settings{
		BaseURL:         server.URL,
		ReservationAuth: "reservation-auth",
		XAppCode:        "app-code",
		XAppClient:      "miniapp",
		UserAgent:       "ua",
		Referer:         "ref",
		WechatID:        "wechat",
	})

	record, err := client.GetNetTicketStatus(context.Background())
	if err != nil {
		t.Fatalf("GetNetTicketStatus() error = %v", err)
	}
	if record.Kind != "net_ticket" {
		t.Fatalf("record kind = %q, want net_ticket", record.Kind)
	}
}

func TestParseReservationTicketMarksReservationKind(t *testing.T) {
	record, err := parseReservationRecord([]byte(`{"reservationTicket":{"ticketId":789,"number":"R001","status":"RESERVED"}}`), "net ticket status")
	if err != nil {
		t.Fatalf("parseReservationRecord() error = %v", err)
	}
	if record.Kind != "reservation" {
		t.Fatalf("record kind = %q, want reservation", record.Kind)
	}
}
