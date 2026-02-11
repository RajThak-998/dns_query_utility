package output

import (
    "dns_query_utility/result"
    "encoding/csv"
    "fmt"
    "os"
    "strconv"
    "strings"
)

// CSVWriter writes results to CSV format
type CSVWriter struct {
    filepath string
}

// NewCSVWriter creates a new CSV writer
func NewCSVWriter(filepath string) *CSVWriter {
    return &CSVWriter{filepath: filepath}
}

// Write outputs results to CSV file
func (w *CSVWriter) Write(results []result.QueryResult, metadata Metadata) error {
    file, err := os.Create(w.filepath)
    if err != nil {
        return fmt.Errorf("failed to create CSV file: %w", err)
    }
    defer file.Close()

    writer := csv.NewWriter(file)
    defer writer.Flush()

    // Write header
    header := []string{
        "domain",
        "query_type",
        "transport",
        "network",
        "status",
        "latency_ms",
        "response_code",
        "resolved_ips",
        "records",
        "error",
        "timestamp",
    }
    if err := writer.Write(header); err != nil {
        return fmt.Errorf("failed to write CSV header: %w", err)
    }

    // Write data rows
    for _, res := range results {
        row := []string{
            res.Domain,
            res.QueryType,
            res.Transport,
            res.IPVersion,
            string(res.Status),
            strconv.FormatInt(res.Latency.Milliseconds(), 10),
            strconv.Itoa(res.ResponseCode),
            joinIPs(res.ResolvedIPs),
            joinRecords(res.Records),
            res.Error,
            res.Timestamp.Format("2006-01-02 15:04:05.000"),
        }
        if err := writer.Write(row); err != nil {
            return fmt.Errorf("failed to write CSV row: %w", err)
        }
    }

    return nil
}

// joinIPs converts IP slice to comma-separated string
func joinIPs(ips []string) string {
    return strings.Join(ips, ";")
}

// joinRecords converts records slice to semicolon-separated string
func joinRecords(records []string) string {
    return strings.Join(records, ";")
}