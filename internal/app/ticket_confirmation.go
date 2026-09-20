package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
)

type ticketConfirmation struct {
	Account    [32]byte
	Generation uint64
	Ticket     string
	Expires    time.Time
}

var ticketConfirmations = struct {
	sync.Mutex
	items map[string]ticketConfirmation
}{items: make(map[string]ticketConfirmation)}

func ticketAccountFingerprint(settings Settings) [32]byte {
	data, _ := json.Marshal([]string{settings.BaseURL, settings.WechatID, settings.ReservationAuth, settings.QueryAuthorization, settings.XAppCode})
	return sha256.Sum256(data)
}

func ticketConfirmationIdentity(ticket ReservationRecord) string {
	if ticket.TicketID <= 0 || strings.TrimSpace(ticket.Number) == "" || !netTicketLooksSuccessful(ticket) {
		return ""
	}
	day := normalizeLocalReservationDate(ticket.QueueDate)
	if day == "" {
		day = time.Now().In(SushiroTimezone).Format("20060102")
	}
	data, _ := json.Marshal([]any{ticket.TicketID, strings.TrimSpace(ticket.Number), DefaultString(ticket.StoreID, ticket.MonitoredStoreID), day})
	return string(data)
}

// Opaque, short-lived tokens bind the displayed ticket to its credential session.
// Account identifiers and credentials never leave the backend in this token.
func issueTicketConfirmation(settings Settings, generation uint64, ticket ReservationRecord) string {
	identity := ticketConfirmationIdentity(ticket)
	if identity == "" {
		return ""
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	now := time.Now()
	ticketConfirmations.Lock()
	defer ticketConfirmations.Unlock()
	for key, item := range ticketConfirmations.items {
		if !item.Expires.After(now) {
			delete(ticketConfirmations.items, key)
		}
	}
	if len(ticketConfirmations.items) >= 256 {
		var oldestKey string
		var oldest time.Time
		for key, item := range ticketConfirmations.items {
			if oldestKey == "" || item.Expires.Before(oldest) {
				oldestKey, oldest = key, item.Expires
			}
		}
		delete(ticketConfirmations.items, oldestKey)
	}
	ticketConfirmations.items[token] = ticketConfirmation{Account: ticketAccountFingerprint(settings), Generation: generation, Ticket: identity, Expires: now.Add(15 * time.Minute)}
	return token
}

// Caller holds authLifecycle and netTicketMu through this check and cancellation.
func checkTicketConfirmation(ctx context.Context, token string, settings Settings, generation uint64, read func(context.Context) (ReservationRecord, error)) error {
	ticketConfirmations.Lock()
	item, ok := ticketConfirmations.items[token]
	ticketConfirmations.Unlock()
	if !ok || !item.Expires.After(time.Now()) {
		return fmt.Errorf("取消确认已过期，请先刷新号码再确认")
	}
	if item.Generation != generation || item.Account != ticketAccountFingerprint(settings) {
		return fmt.Errorf("账号连接已变化，请重新查询号码后再取消")
	}
	ticket, err := read(ctx)
	if err != nil {
		noteAuthResult(err)
		return fmt.Errorf("暂时无法核对当前号码，未执行取消，请刷新号码后重试：%s", friendlyNetTicketError(err))
	}
	noteAuthResult(nil)
	if actual := ticketConfirmationIdentity(ticket); actual == "" || actual != item.Ticket {
		return fmt.Errorf("当前号码与页面显示不一致，未执行取消，请刷新号码")
	}
	// A submitted cancellation is never replayed, including uncertain responses.
	ticketConfirmations.Lock()
	delete(ticketConfirmations.items, token)
	ticketConfirmations.Unlock()
	return nil
}
