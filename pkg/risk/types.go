package risk

import "time"

type RiskItem struct {
	Service    string
	AlertName  string
	Severity   string
	FirstSeen  time.Time
	LastSeen   time.Time
	TTL        time.Duration
	Score   int
	Summary string
	Risk	  string
}
