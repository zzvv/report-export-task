package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/zzvv/report-export-service/internal/export"
)

type Handler struct{ service *export.Service }

func NewHandler(service *export.Service) http.Handler { return Handler{service: service} }

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if r.Method == http.MethodPost && path == "v1/exports" {
		var input struct {
			Report string `json:"report"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "请求体格式错误")
			return
		}
		job, err := h.service.Create(input.Report)
		writeJob(w, job, err, http.StatusCreated)
		return
	}
	if len(parts) == 3 && parts[0] == "v1" && parts[1] == "exports" {
		id := parts[2]
		switch r.Method {
		case http.MethodGet:
			job, err := h.service.Get(id)
			writeJob(w, job, err, http.StatusOK)
			return
		case http.MethodDelete:
			job, err := h.service.Cancel(id)
			writeJob(w, job, err, http.StatusOK)
			return
		}
	}
	if len(parts) == 4 && parts[0] == "v1" && parts[1] == "exports" && parts[3] == "retry" && r.Method == http.MethodPost {
		job, err := h.service.Retry(parts[2])
		writeJob(w, job, err, http.StatusAccepted)
		return
	}
	writeError(w, http.StatusNotFound, "路由不存在")
}

func writeJob(w http.ResponseWriter, job export.Job, err error, successCode int) {
	if err != nil {
		code := http.StatusBadRequest
		if errors.Is(err, export.ErrNotFound) {
			code = http.StatusNotFound
		}
		writeError(w, code, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(successCode)
	_ = json.NewEncoder(w).Encode(job)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
