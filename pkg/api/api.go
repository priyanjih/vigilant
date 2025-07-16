package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type APIRiskItem struct {
	Service   string   `json:"service"`
	Alert     string   `json:"alert"`
	Severity  string   `json:"severity"`
	Score     int      `json:"score"`
	Symptoms  []string `json:"symptoms"`
	Metrics   []string `json:"metrics"`
	Summary   string   `json:"summary"`
}

var (
	currentAPIRisks []APIRiskItem
	riskMu          sync.RWMutex
)

func StartServer() {
	http.HandleFunc("/api/risks", func(w http.ResponseWriter, r *http.Request) {
		riskMu.RLock()
		defer riskMu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentAPIRisks)
	})

	// Optional frontend handler
	http.Handle("/", http.FileServer(http.Dir("./dashboard/dist")))

	fmt.Println("🚀 API running at: http://localhost:8090")
	http.ListenAndServe(":8090", nil)
}

func UpdateRisks(newRisks []APIRiskItem) {
	riskMu.Lock()
	defer riskMu.Unlock()
	currentAPIRisks = newRisks
}
