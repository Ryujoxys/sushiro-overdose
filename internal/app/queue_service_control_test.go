package app

import (
	"context"
	"os"
	"testing"
	"time"

	. "github.com/Ryujoxys/sushiro-overdose/internal/core"
	"github.com/Ryujoxys/sushiro-overdose/internal/platform"
)

func TestQueueServiceCooperativeExitPreservesConfigAndRecords(t *testing.T) {
	publicCollectorFixture(t)
	before, err := os.ReadFile(queueBaselinePath())
	if err != nil {
		t.Fatal(err)
	}
	records := []byte("{\"store_id\":1012,\"collected_at\":\"2026-09-20T10:00:00+08:00\"}\n")
	if err := os.WriteFile(queueBaselineRecordsPath(), records, 0o600); err != nil {
		t.Fatal(err)
	}
	old := queueBaselineCollector
	queueBaselineCollector = &QueueBaselineCollector{collect: func(ctx context.Context, _ QueueBaselineConfig) (int, error) { <-ctx.Done(); return 0, ctx.Err() }}
	t.Cleanup(func() { queueBaselineCollector = old })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- runPublicQueueService(ctx) }()
	deadline := time.Now().Add(2 * time.Second)
	for readQueueServiceControl().Token == "" && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if readQueueServiceControl().Token == "" {
		t.Fatal("service did not start")
	}
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	if err := stopPublicQueueService(stopCtx); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(queueBaselinePath())
	if string(before) != string(after) {
		t.Fatal("exit changed recording preferences")
	}
	saved, _ := os.ReadFile(queueBaselineRecordsPath())
	if string(saved) != string(records) {
		t.Fatal("exit changed personal records")
	}
	if _, err := os.Stat(queueServiceControlPath()); !os.IsNotExist(err) {
		t.Fatal("session leaked")
	}
	if _, err := os.Stat(queueServiceStopPath()); !os.IsNotExist(err) {
		t.Fatal("stop marker leaked")
	}
}

func TestQueueServiceStopRejectsUnknownOwner(t *testing.T) {
	reliabilityHome(t)
	lock, err := platform.TryFileLock(publicQueueServiceLockPath())
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	writeSamplingPID(os.Getpid())
	if err := stopPublicQueueService(context.Background()); err == nil {
		t.Fatal("unknown process was accepted")
	}
	if !platform.IsProcessAlive(os.Getpid()) {
		t.Fatal("test process stopped")
	}
	if _, err := os.Stat(queueServiceStopPath()); !os.IsNotExist(err) {
		t.Fatal("wrote unknown stop marker")
	}
}

func TestQueueServiceControlIgnoresOtherSessionToken(t *testing.T) {
	reliabilityHome(t)
	control, err := newQueueServiceControl()
	if err != nil {
		t.Fatal(err)
	}
	defer control.close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { control.watch(ctx, cancel); close(done) }()
	if err := AtomicWriteFile(queueServiceStopPath(), []byte("different-session"), 0o600); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
		t.Fatal("wrong token stopped service")
	case <-time.After(150 * time.Millisecond):
	}
	cancel()
	<-done
}
