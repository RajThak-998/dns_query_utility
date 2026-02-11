package result

import (
    "encoding/json"
    "time"
)

type Status string

const (
    StatusSuccess  Status = "success"
    StatusNoAnswer Status = "no_answer"
    StatusNXDomain Status = "nxdomain"
    StatusServFail Status = "servfail"
    StatusRefused  Status = "refused"
    StatusTimeout  Status = "timeout"
    StatusError    Status = "error"
)

type QueryResult struct {
    Domain       string        `json:"domain"`
    QueryType    string        `json:"query_type"`
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