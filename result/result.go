package result

import "time"

// QueryStatus represents the outcome of a DNS query
type QueryStatus string

const (
    StatusSuccess  QueryStatus = "success"
    StatusNoAnswer QueryStatus = "no_answer"
    StatusNXDomain QueryStatus = "nxdomain"
    StatusServFail QueryStatus = "servfail"
    StatusRefused  QueryStatus = "refused"
    StatusTimeout  QueryStatus = "timeout"
    StatusError    QueryStatus = "error"
)

// QueryResult holds the outcome of a single DNS query
type QueryResult struct {
    Domain       string        `json:"domain"`
    QueryType    string        `json:"query_type"`
    Transport    string        `json:"transport"`
    IPVersion    string        `json:"network"`
    Status       QueryStatus   `json:"status"`
    Latency      time.Duration `json:"latency_ms"`
    ResponseCode int           `json:"response_code"`
    ResolvedIPs  []string      `json:"resolved_ips,omitempty"`
    Records      []string      `json:"records,omitempty"`
    Error        string        `json:"error,omitempty"`
    Timestamp    time.Time     `json:"timestamp"`
}