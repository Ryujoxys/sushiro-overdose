package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOfficialMutationCallsStayInApprovedEntrypoints(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob source files: %v", err)
	}

	allowed := map[string][]string{
		".CreateReservation(": {"runBooking", "runBookingLoop", "runSniper", "runSniperLoop"},
		".CreateNetTicket(":   {"takeNetTicketLocked", "fireNetTicket", "runAuthVerify"},
		".CancelReservation(": {"handleCancelReservation", "cmdCancel"},
		".CancelNetTicket(":   {"handleCancelNetTicket", "runAuthVerify"},
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		source := string(data)
		for call, handlers := range allowed {
			if !strings.Contains(source, call) {
				continue
			}
			allowedCount := 0
			for _, handler := range handlers {
				body, ok := functionBody(source, handler)
				if ok {
					allowedCount += strings.Count(body, call)
				}
			}
			if strings.Count(source, call) != allowedCount {
				t.Fatalf("%s calls %s outside approved explicit cancellation entrypoints %v; update .specify/memory/constitution.md and this guard before changing cancellation boundaries", file, call, handlers)
			}
		}
	}
}

func functionBody(source, name string) (string, bool) {
	offset := 0
	for {
		start := strings.Index(source[offset:], "func ")
		if start < 0 {
			return "", false
		}
		start += offset
		open := strings.Index(source[start:], "{")
		if open < 0 {
			return "", false
		}
		open += start
		signature := source[start:open]
		if functionSignatureNameMatches(signature, name) {
			depth := 0
			for i := open; i < len(source); i++ {
				switch source[i] {
				case '{':
					depth++
				case '}':
					depth--
					if depth == 0 {
						return source[open : i+1], true
					}
				}
			}
			return "", false
		}
		offset = open + 1
	}
}

func functionSignatureNameMatches(signature, name string) bool {
	signature = strings.TrimSpace(strings.TrimPrefix(signature, "func"))
	if strings.HasPrefix(signature, name+"(") {
		return true
	}
	return strings.Contains(signature, ") "+name+"(")
}
