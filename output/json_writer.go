package output

import (
    "dns_query_utility/result"
    "encoding/json"
    "fmt"
    "os"
    "time"
)

// JSONWriter writes results to JSON format
type JSONWriter struct {
    filepath string
}

// NewJSONWriter creates a new JSON writer
func NewJSONWriter(filepath string) *JSONWriter {
    return &JSONWriter{filepath: filepath}
}

// JSONOutput represents the complete JSON output structure
type JSONOutput struct {
    Metadata Metadata              `json:"metadata"`
    Results  []result.QueryResult  `json:"results"`
}

// Write outputs results to JSON file
func (w *JSONWriter) Write(results []result.QueryResult, metadata Metadata) error {
    // Ensure timestamps are set for results that don't have them
    for i := range results {
        if results[i].Timestamp.IsZero() {
            results[i].Timestamp = time.Now()
        }
    }

    output := JSONOutput{
        Metadata: metadata,
        Results:  results,
    }

    // Create file
    file, err := os.Create(w.filepath)
    if err != nil {
        return fmt.Errorf("failed to create JSON file: %w", err)
    }
    defer file.Close()

    // Write JSON with indentation for readability
    encoder := json.NewEncoder(file)
    encoder.SetIndent("", "  ")
    
    if err := encoder.Encode(output); err != nil {
        return fmt.Errorf("failed to write JSON: %w", err)
    }

    return nil
}