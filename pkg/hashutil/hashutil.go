package hashutil

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
)

// HashData creates an MD5 hash of the given data by marshaling it to JSON
func HashData(data interface{}) string {
	jsonData, _ := json.Marshal(data)
	hash := md5.Sum(jsonData)
	return fmt.Sprintf("%x", hash)
}

// SafeHashDisplay returns a truncated version of the hash for display purposes
// Returns the first 8 characters if the hash is long enough, otherwise returns the full hash
func SafeHashDisplay(hash string) string {
	if len(hash) >= 8 {
		return hash[:8]
	}
	return hash
}

// Simplified data structures for hashing consistency

type SimplifiedSymptom struct {
	Service string
	Pattern string
	Count   int
}


type SimplifiedMetric struct {
	Service   string
	CheckName string
	Value     float64
	Operator  string
	Threshold float64
}


type SimplifiedAlert struct {
	Service   string
	AlertName string
	Severity  string
}