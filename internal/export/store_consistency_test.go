package export

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// blockingWorker 允许测试精确控制每次 Export 何时返回，从而构造过期批次迟到的时序。
// started 在 Export 到达阻塞点（已通过 MarkRunning）时关闭，便于测试在放行前确认批次已就位。
type blockingWorker struct {
	mu      sync.Mutex
	runs    int
	started []chan struct{}
	wait    []chan struct{}
	urls    []string
	errs    []error
}

func newBlockingWorker() *blockingWorker {
	return &blockingWorker{}
}

func (w *blockingWorker) next(url string, runErr error) (started, wait chan struct{}) {
	started = make(chan struct{})
	wait = make(chan struct{})
	w.mu.Lock()
	w.started = append(w.started, started)
	w.wait = append(w.wait, wait)
	w.urls = append(w.urls, url)
	w.errs = append(w.errs, runErr)
	w.mu.Unlock()
	return started, wait
}

func (w *blockingWorker) Export(_ context.Context, _ string, _ string, _ int) (string, error) {
	w.mu.Lock()
	idx := w.runs
	w.runs++
	started := w.started[idx]
	wait := w.wait[idx]
	url := w.urls[idx]
	runErr := w.errs[idx]
	w.mu.Unlock()
	close(started) // 已到达阻塞点
	<-wait         // 等待测试放行
	return url, runErr
}

// Test旧批次迟到结果不能覆盖新批次验证：取消后重试，旧批次晚到的成功结果
// 必须被丢弃，不能改写新批次已产生的终态与文件地址。
func Test旧批次迟到结果不能覆盖新批次(t *testing.T) {
	store := NewMemoryStore()
	worker := newBlockingWorker()
	svc := NewService(store, worker)

	started1, wait1 := worker.next("/downloads/stale.csv", nil) // 旧批次：迟到返回成功
	started2, wait2 := worker.next("/downloads/retry.csv", nil) // 新批次：重试返回成功

	job := mustCreate(t, svc, "daily-sales")
	<-started1 // 旧批次已进入 worker 阻塞点（MarkRunning 成功，attempt=1）

	if _, err := svc.Cancel(job.ID); err != nil {
		t.Fatalf("取消失败: %v", err)
	}
	if _, err := svc.Retry(job.ID); err != nil {
		t.Fatalf("重试失败: %v", err)
	}
	<-started2 // 新批次已进入 worker 阻塞点（attempt=2）

	// 让新批次先完成，再放行旧批次——构造"旧批次迟到"的竞态。
	close(wait2)
	if !waitForStatus(store, job.ID, StatusSucceeded) {
		t.Fatalf("新批次未成功")
	}
	if got, _ := store.Get(job.ID); got.FileURL != "/downloads/retry.csv" || got.Attempt != 2 {
		t.Fatalf("新批次终态异常: attempt=%d file_url=%q", got.Attempt, got.FileURL)
	}

	close(wait1) // 旧批次迟到返回，必须被丢弃
	if !waitForStatus(store, job.ID, StatusSucceeded) {
		t.Fatalf("旧批次迟到结果改写了终态")
	}
	got, _ := store.Get(job.ID)
	if got.FileURL != "/downloads/retry.csv" || got.Attempt != 2 {
		t.Fatalf("旧批次覆盖了重试结果: attempt=%d file_url=%q", got.Attempt, got.FileURL)
	}
}

// Test旧批次迟到的失败不能覆盖新批次成功验证失败结果同样会被丢弃。
func Test旧批次迟到的失败不能覆盖新批次成功(t *testing.T) {
	store := NewMemoryStore()
	worker := newBlockingWorker()
	svc := NewService(store, worker)

	started1, wait1 := worker.next("", errors.New("旧批次超时"))
	started2, wait2 := worker.next("/downloads/retry.csv", nil)

	job := mustCreate(t, svc, "daily-sales")
	<-started1
	if _, err := svc.Cancel(job.ID); err != nil {
		t.Fatalf("取消失败: %v", err)
	}
	if _, err := svc.Retry(job.ID); err != nil {
		t.Fatalf("重试失败: %v", err)
	}
	<-started2

	close(wait2)
	if !waitForStatus(store, job.ID, StatusSucceeded) {
		t.Fatalf("新批次未成功")
	}
	close(wait1) // 旧批次迟到返回失败
	got, _ := store.Get(job.ID)
	if got.Status != StatusSucceeded || got.FileURL != "/downloads/retry.csv" {
		t.Fatalf("旧批次失败覆盖了新批次成功: status=%q file_url=%q", got.Status, got.FileURL)
	}
}

func mustCreate(t *testing.T, svc *Service, report string) Job {
	t.Helper()
	job, err := svc.Create(report)
	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	return job
}

func waitForStatus(store *MemoryStore, id string, want Status) bool {
	for range 10_000 {
		got, err := store.Get(id)
		if err != nil {
			return false
		}
		if got.Status == want {
			return true
		}
	}
	return false
}
