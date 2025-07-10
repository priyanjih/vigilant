package risk

import (
	"fmt"
	"sync"
	"time"

	"vigilant/pkg/prometheus"
)

type RiskTracker struct {
	Items map[string]*RiskItem
	Mutex sync.Mutex
	TTL   time.Duration
}

func NewRiskTracker(ttl time.Duration) *RiskTracker {
	return &RiskTracker{
		Items: make(map[string]*RiskItem),
		TTL:   ttl,
	}
}

func (rt *RiskTracker) UpdateFromAlerts(alerts []prometheus.Alert) {
	rt.Mutex.Lock()
	defer rt.Mutex.Unlock()

	now := time.Now()

	for _, a := range alerts {
		key := a.Instance // or a.Name+a.Instance if multiple alerts per service

		if item, exists := rt.Items[key]; exists {
			item.LastSeen = now
			item.TTL = rt.TTL
		} else {
			rt.Items[key] = &RiskItem{
				Service:   a.Instance,
				AlertName: a.Name,
				Severity:  a.Severity,
				FirstSeen: now,
				LastSeen:  now,
				TTL:       rt.TTL,
			}
		}
	}
}

func (rt *RiskTracker) CleanupExpired() {
	rt.Mutex.Lock()
	defer rt.Mutex.Unlock()

	now := time.Now()
	for key, item := range rt.Items {
		if now.Sub(item.LastSeen) > item.TTL {
			fmt.Printf("[INFO] Expired: %s\n", key)
			delete(rt.Items, key)
		}
	}
}

func (rt *RiskTracker) Print() {
	rt.Mutex.Lock()
	defer rt.Mutex.Unlock()

	fmt.Println("=== Active Risk Items ===")
	for _, item := range rt.Items {
		fmt.Printf("Service: %s | Alert: %s | Severity: %s | TTL: %v\n",
			item.Service, item.AlertName, item.Severity, time.Until(item.LastSeen.Add(item.TTL)))
	}
}
