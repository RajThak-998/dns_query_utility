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
	Domain          string      `json:"domain"`
	QueryType       string      `json:"query_type"`
	Transport       string      `json:"transport"`
	IPVersion       string      `json:"network"`
	Status          QueryStatus `json:"status"`
	LatencyMs       float64     `json:"latency_ms"`
	ResponseCode    int         `json:"response_code"`
	ResolvedIPs     []string    `json:"resolved_ips,omitempty"`
	Records         []string    `json:"records,omitempty"`
	AuthoritativeNS []string    `json:"authoritative_ns"` // NEW: NS records from Authority section
	Error           string      `json:"error,omitempty"`
	Timestamp       time.Time   `json:"timestamp"`
}

// TypeResult holds the result for a specific query type
type TypeResult struct {
	Status          QueryStatus `json:"status"`
	LatencyMs       float64     `json:"latency_ms"`
	ResponseCode    int         `json:"response_code"`
	ResolvedIPs     []string    `json:"ips,omitempty"`
	Records         []string    `json:"records,omitempty"`
	AuthoritativeNS []string    `json:"authoritative_ns,omitempty"` // NEW: NS records from Authority section
	Error           string      `json:"error,omitempty"`
	Transport       string      `json:"transport"`
	IPVersion       string      `json:"network"`
	Timestamp       time.Time   `json:"timestamp"`
}

// ConsolidatedResult holds all query types for a single domain
type ConsolidatedResult struct {
	Domain     string                `json:"domain"`
	QueryTypes map[string]TypeResult `json:"query_types"`
	Summary    ConsolidatedSummary   `json:"summary"`
}

// ConsolidatedSummary provides aggregate statistics for a domain
type ConsolidatedSummary struct {
	TotalQueries     int     `json:"total_queries"`
	Successful       int     `json:"successful"`
	NoAnswer         int     `json:"no_answer"`
	Failed           int     `json:"failed"`
	AverageLatencyMs float64 `json:"average_latency_ms"`
}
