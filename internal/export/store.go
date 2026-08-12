package export

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrNotFound     = errors.New("导出任务不存在")
	ErrNotRetryable = errors.New("当前状态不允许重试")
)

type Store interface {
	Create(report string) Job
	Get(id string) (Job, error)
	Cancel(id string) (Job, error)
	Retry(id string) (Job, error)
	Complete(id string, attempt int, fileURL string, runErr error) (Job, error)
}

type MemoryStore struct {
	mu   sync.RWMutex
	jobs map[string]Job
	next uint64
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{jobs: make(map[string]Job)}
}

func (s *MemoryStore) Create(report string) Job {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	now := time.Now().UTC()
	job := Job{
		ID:        "export-" + formatID(s.next),
		Report:    report,
		Status:    StatusQueued,
		Attempt:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.jobs[job.ID] = job
	return job
}

func (s *MemoryStore) Get(id string) (Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	if !ok {
		return Job{}, ErrNotFound
	}
	return job, nil
}

func (s *MemoryStore) Cancel(id string) (Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return Job{}, ErrNotFound
	}
	if job.Status != StatusQueued && job.Status != StatusRunning {
		return Job{}, ErrNotRetryable
	}
	job.Status = StatusCanceled
	job.UpdatedAt = time.Now().UTC()
	s.jobs[id] = job
	return job, nil
}

func (s *MemoryStore) Retry(id string) (Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return Job{}, ErrNotFound
	}
	if !job.Retryable() {
		return Job{}, ErrNotRetryable
	}
	job.Status = StatusQueued
	job.Attempt++
	job.FileURL = ""
	job.Error = ""
	job.UpdatedAt = time.Now().UTC()
	s.jobs[id] = job
	return job, nil
}

func (s *MemoryStore) Complete(id string, attempt int, fileURL string, runErr error) (Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return Job{}, ErrNotFound
	}

	if runErr != nil {
		job.Status = StatusFailed
		job.Error = runErr.Error()
		job.FileURL = ""
	} else {
		job.Status = StatusSucceeded
		job.Error = ""
		job.FileURL = fileURL
	}
	job.UpdatedAt = time.Now().UTC()
	s.jobs[id] = job
	return job, nil
}

func (s *MemoryStore) MarkRunning(id string, attempt int) (Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return Job{}, ErrNotFound
	}
	if job.Attempt != attempt || job.Status != StatusQueued {
		return Job{}, ErrNotRetryable
	}
	job.Status = StatusRunning
	job.UpdatedAt = time.Now().UTC()
	s.jobs[id] = job
	return job, nil
}

func formatID(value uint64) string {
	const digits = "0123456789"
	buf := [6]byte{'0', '0', '0', '0', '0', '0'}
	for i := len(buf) - 1; i >= 0; i-- {
		buf[i] = digits[value%10]
		value /= 10
	}
	return string(buf[:])
}
