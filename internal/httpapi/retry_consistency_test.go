package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/zzvv/report-export-service/internal/export"
)

func Test取消后重试不能被旧导出结果覆盖(t *testing.T) {
	worker := newControlledWorker()
	h := NewHandler(export.NewService(export.NewMemoryStore(), worker))

	job := requestJob(t, h, http.MethodPost, "/v1/exports", `{"report":"daily-sales"}`)
	<-worker.firstStarted
	requestJob(t, h, http.MethodDelete, "/v1/exports/"+job.ID, "")
	requestJob(t, h, http.MethodPost, "/v1/exports/"+job.ID+"/retry", "")
	<-worker.secondStarted

	close(worker.allowFirst)
	close(worker.allowSecond)

	var got export.Job
	for range 10_000 {
		got = requestJob(t, h, http.MethodGet, "/v1/exports/"+job.ID, "")
		if got.Status == export.StatusSucceeded {
			break
		}
	}
	if got.Attempt != 2 || got.FileURL != "/downloads/retry.csv" {
		t.Fatalf("旧执行覆盖了重试结果: attempt=%d file_url=%q", got.Attempt, got.FileURL)
	}
}

type controlledWorker struct {
	firstStarted  chan struct{}
	secondStarted chan struct{}
	allowFirst    chan struct{}
	allowSecond   chan struct{}
	mu            sync.Mutex
	runs          int
}

func newControlledWorker() *controlledWorker {
	return &controlledWorker{make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{}), sync.Mutex{}, 0}
}

func (w *controlledWorker) Export(_ context.Context, _ string, _ string, _ int) (string, error) {
	w.mu.Lock()
	w.runs++
	run := w.runs
	w.mu.Unlock()
	if run == 1 {
		close(w.firstStarted)
		<-w.allowFirst
		return "/downloads/stale.csv", nil
	}
	close(w.secondStarted)
	<-w.allowSecond
	return "/downloads/retry.csv", nil
}

func requestJob(t *testing.T, h http.Handler, method, path, body string) export.Job {
	t.Helper()
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, httptest.NewRequest(method, path, bytes.NewBufferString(body)))
	if recorder.Code >= 300 {
		t.Fatalf("%s %s 返回 %d: %s", method, path, recorder.Code, recorder.Body.String())
	}
	var job export.Job
	if err := json.NewDecoder(recorder.Body).Decode(&job); err != nil {
		t.Fatal(err)
	}
	return job
}
