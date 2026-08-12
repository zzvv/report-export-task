package export

import (
	"context"
	"fmt"
)

type Worker interface {
	Export(ctx context.Context, report, jobID string, attempt int) (string, error)
}

type CSVWorker struct{}

func NewCSVWorker() CSVWorker { return CSVWorker{} }

func (CSVWorker) Export(ctx context.Context, report, jobID string, attempt int) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		return fmt.Sprintf("/downloads/%s-attempt-%d.csv", jobID, attempt), nil
	}
}
