package output

import (
	"dns_query_utility/result"
	"time"
)

// Format represents output file format
type Format string

const (
	FormatCSV  Format = "csv"
	FormatJSON Format = "json"
	FormatAll  Format = "all" // Generate both
)

// Metadata contains summary information about the query run
type Metadata struct {
	Timestamp         time.Time `json:"timestamp"`
	TotalQueries      int       `json:"total_queries"`
	SuccessfulQueries int       `json:"successful_queries"`
	NoAnswerQueries   int       `json:"no_answer_queries"`
	FailedQueries     int       `json:"failed_queries"`
	TotalDurationMs   int64     `json:"total_duration_ms"`
	AverageLatencyMs  float64   `json:"average_latency_ms"`
	QueriesPerSecond  float64   `json:"queries_per_second"`
	DNSServerIPv4     string    `json:"dns_server_ipv4"`
	DNSServerIPv6     string    `json:"dns_server_ipv6"`
	WorkersUsed       int       `json:"workers_used"`
	TimeoutSeconds    float64   `json:"timeout_seconds"`
	RetryCount        int       `json:"retry_count"`
}

// Writer interface for output formats
type Writer interface {
	Write(results []result.QueryResult, metadata Metadata) error
}

// WriteOutput writes results to file(s) based on format
func WriteOutput(filepath string, format Format, results []result.QueryResult, metadata Metadata) error {
	switch format {
	case FormatCSV:
		w := NewCSVWriter(filepath)
		return w.Write(results, metadata)

	case FormatJSON:
		w := NewJSONWriter(filepath)
		return w.Write(results, metadata)

	case FormatAll:
		// Generate both CSV and JSON
		csvPath := ChangeExtension(filepath, ".csv")
		jsonPath := ChangeExtension(filepath, ".json")

		csvWriter := NewCSVWriter(csvPath)
		if err := csvWriter.Write(results, metadata); err != nil {
			return err
		}

		jsonWriter := NewJSONWriter(jsonPath)
		return jsonWriter.Write(results, metadata)

	default:
		return nil
	}
}

// ChangeExtension replaces or adds file extension (exported now)
func ChangeExtension(filepath string, newExt string) string {
	// Remove existing extension if any
	for i := len(filepath) - 1; i >= 0; i-- {
		if filepath[i] == '.' {
			return filepath[:i] + newExt
		}
		if filepath[i] == '/' || filepath[i] == '\\' {
			break
		}
	}
	// No extension found, append
	return filepath + newExt
}
