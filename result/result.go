package result

import (
	"encoding/json"
	"time"
)

type Status string

const (
	StatusSuccess  Status = "success"   // RCODE=0 and A/AAAA records found
	StatusNoAnswer Status = "no_answer" // RCODE=0 but no A/AAAA records (e.g., only CNAME/SOA)
	StatusNXDomain Status = "nxdomain"  // RCODE=3 (domain doesn't exist)
	StatusServFail Status = "servfail"  // RCODE=2 (server failure)
	StatusRefused  Status = "refused"   // RCODE=5 (query refused)
	StatusTimeout  Status = "timeout"   // Network timeout
	StatusError    Status = "error"     // Other errors (network, parsing, etc.)
)

type QueryResult struct {
	Domain       string        `json:"domain"`
	Transport    string        `json:"transport"`
	IPVersion    string        `json:"ip_version"`
	ResponseCode int           `json:"response_code"`
	ResolvedIPs  []string      `json:"resolved_ips"`
	Records      []string      `json:"records,omitempty"`
	Latency      time.Duration `json:"latency_ms"`
	Status       Status        `json:"status"`
	Error        string        `json:"error,omitempty"`
}

func (r *QueryResult) ToJSON() (string, error) {
	bytes, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
