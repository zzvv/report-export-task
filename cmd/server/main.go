package main

import (
	"log"
	"net/http"

	"github.com/zzvv/report-export-service/internal/export"
	"github.com/zzvv/report-export-service/internal/httpapi"
)

func main() {
	store := export.NewMemoryStore()
	service := export.NewService(store, export.NewCSVWorker())
	handler := httpapi.NewHandler(service)

	log.Println("报表导出服务监听地址: :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
