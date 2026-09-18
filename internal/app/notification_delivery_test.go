package app

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Ryujoxys/sushiro-overdose/internal/notify"
)

var testNotificationCounts sync.Map

func notificationCountForTest(channel string) int64 {
	value, _ := testNotificationCounts.LoadOrStore(channel, &atomic.Int64{})
	return value.(*atomic.Int64).Load()
}

func recordTestNotification(channel string) {
	value, _ := testNotificationCounts.LoadOrStore(channel, &atomic.Int64{})
	value.(*atomic.Int64).Add(1)
}

type silentTestNotifier struct{ channel string }

func (n silentTestNotifier) Name() string { return n.channel }
func (n silentTestNotifier) Send(context.Context, string, string) error {
	recordTestNotification(n.channel)
	return nil
}
func (n silentTestNotifier) SendSession(ctx context.Context, _ string, title, content string) error {
	return n.Send(ctx, title, content)
}

func TestMain(m *testing.M) {
	// Never inherit a real preview's data root when running the test suite.
	_ = os.Unsetenv("SUSHIRO_DATA_HOME")
	home, err := os.MkdirTemp("", "sushiro-app-tests-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// Never load the developer's saved notification channels during tests.
	for _, name := range []string{"HOME", "USERPROFILE"} {
		if err := os.Setenv(name, home); err != nil {
			fmt.Fprintln(os.Stderr, err)
			_ = os.RemoveAll(home)
			os.Exit(1)
		}
	}
	sendDesktopNotification = func(string, string) { recordTestNotification("desktop") }
	configuredNotifiers = func() *notify.MultiNotifier {
		// Preserve configured channel names so routing and readiness still get tested.
		var channels []notify.Notifier
		for _, channel := range notify.BuildNotifierFromConfig().List() {
			channels = append(channels, silentTestNotifier{channel: channel.Name()})
		}
		return notify.NewMultiNotifier(channels...)
	}
	code := m.Run()
	_ = os.RemoveAll(home)
	os.Exit(code)
}

func TestNotificationDeliveryUsesSilentTestChannels(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	cfg := &notify.NotifyConfig{}
	cfg.Feishu.Webhook = "https://example.invalid/must-not-be-requested"
	if err := notify.SaveNotifyConfig(cfg); err != nil {
		t.Fatal(err)
	}
	for _, channel := range configuredNotifiers().List() {
		if _, ok := channel.(silentTestNotifier); !ok {
			t.Fatalf("real notifier in test: %T", channel)
		}
	}
	desktopBefore, feishuBefore := notificationCountForTest("desktop"), notificationCountForTest("feishu")
	sendQueueAlert(context.Background(), "test", "test")
	sendQueueAlertWithChannels(context.Background(), "test", "test", "", []string{"feishu"})
	sendQueueAlertWithChannels(context.Background(), "test", "test", "test-session", []string{"feishu"})
	results, ok := runNotificationTest(context.Background(), "feishu")
	if !ok || len(results) != 1 || !results[0].OK {
		t.Fatalf("notification probe = %+v, %v", results, ok)
	}
	if got := notificationCountForTest("desktop") - desktopBefore; got != 3 {
		t.Fatalf("desktop deliveries = %d, want 3", got)
	}
	if got := notificationCountForTest("feishu") - feishuBefore; got != 4 {
		t.Fatalf("channel deliveries = %d, want 4", got)
	}
}

func TestNotificationCallsStayBehindDeliveryBoundary(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") || path == "notification_delivery.go" {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			if name, ok := node.(*ast.Ident); ok && (name.Name == "DesktopNotification" || name.Name == "BuildNotifierFromConfig") {
				t.Errorf("%s bypasses the notification delivery boundary", fset.Position(name.Pos()))
			}
			return true
		})
	}
}
