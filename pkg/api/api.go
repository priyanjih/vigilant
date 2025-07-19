package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type APIMetric struct {
	Name      string  `json:"name"`
	Value     float64 `json:"value"`
	Operator  string  `json:"operator"`
	Threshold float64 `json:"threshold"`
  }
  
  type APISymptom struct {
	Pattern string `json:"pattern"`
	Count   int    `json:"count"`
  }
  
  type APIRiskItem struct {
	Service   string       `json:"service"`
	Alert     string       `json:"alert"`
	Severity  string       `json:"severity"`
	Score     int          `json:"score"`
	Symptoms  []APISymptom `json:"symptoms"`
	Metrics   []APIMetric  `json:"metrics"`
	Summary   string       `json:"summary"`
	Risk     string `json:"risk"`
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
