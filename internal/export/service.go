package export

import (
	"context"
	"errors"
	"sync"
)

type Service struct {
	store   *MemoryStore
	worker  Worker
	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

func NewService(store *MemoryStore, worker Worker) *Service {
	return &Service{store: store, worker: worker, cancels: make(map[string]context.CancelFunc)}
}

func (s *Service) Create(report string) (Job, error) {
	if report == "" {
		return Job{}, errors.New("报表类型不能为空")
	}
	job := s.store.Create(report)
	s.start(job)
	return job, nil
}

func (s *Service) Get(id string) (Job, error) { return s.store.Get(id) }

func (s *Service) Cancel(id string) (Job, error) {
	job, err := s.store.Cancel(id)
	if err != nil {
		return Job{}, err
	}
	s.mu.Lock()
	if cancel := s.cancels[id]; cancel != nil {
		cancel()
	}
	s.mu.Unlock()
	return job, nil
}

func (s *Service) Retry(id string) (Job, error) {
	job, err := s.store.Retry(id)
	if err != nil {
		return Job{}, err
	}
	s.start(job)
	return job, nil
}

func (s *Service) start(job Job) {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.cancels[job.ID] = cancel
	s.mu.Unlock()

	go func(id, report string, attempt int) {
		if _, err := s.store.MarkRunning(id, attempt); err != nil {
			return
		}
		fileURL, runErr := s.worker.Export(ctx, report, id, attempt)
		_, _ = s.store.Complete(id, attempt, fileURL, runErr)
	}(job.ID, job.Report, job.Attempt)
}
