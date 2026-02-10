package result

import (
	"dns_query_utility/query"
	"encoding/json"
	"time"
)

type Status string

const (
	StatusSuccess Status = "success"
	StatusTimeout Status = "timeout"
	StatusError Status = "error"
)

type QueryResult struct {
	Domain string `json:"domain"`
	Transport query.Transport `json:"transport"`
	IPVersion query.IPVersion `json:"ip_version"`
	ResponseCode int `json:"response_code"`
	ResolvedIPs []string `json:"resolved_ips"`
	Latency time.Duration `json:"latency_ms"`
	Status Status `json:"status"`
	Error string `json:"error,omitempty"`
}

func (r *QueryResult) ToJSON() (string, error) {
	bytes, err := json.MarshalIndent(r, "", " ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}