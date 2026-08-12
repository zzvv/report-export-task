package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zzvv/report-export-service/internal/export"
)

func TestCreateAndReadExport(t *testing.T) {
	h := NewHandler(export.NewService(export.NewMemoryStore(), instantWorker{}))

	created := httptest.NewRecorder()
	h.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/v1/exports", bytes.NewBufferString(`{"report":"orders"}`)))
	if created.Code != http.StatusCreated {
		t.Fatalf("创建导出任务状态码 = %d", created.Code)
	}
}

type instantWorker struct{}

func (instantWorker) Export(context.Context, string, string, int) (string, error) {
	return "/downloads/orders.csv", nil
}
