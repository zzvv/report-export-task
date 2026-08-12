package export

import "time"

type Status string

const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCanceled  Status = "canceled"
)

type Job struct {
	ID        string    `json:"id"`
	Report    string    `json:"report"`
	Status    Status    `json:"status"`
	Attempt   int       `json:"attempt"`
	FileURL   string    `json:"file_url,omitempty"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (j Job) Retryable() bool {
	return j.Status == StatusCanceled || j.Status == StatusFailed
}
