package output

import (
    "dns_query_utility/result"
    "encoding/json"
    "fmt"
    "os"
    
)

// ConsolidatedJSONWriter writes consolidated results to JSON format
type ConsolidatedJSONWriter struct {
    filepath string
}

// NewConsolidatedJSONWriter creates a new consolidated JSON writer
func NewConsolidatedJSONWriter(filepath string) *ConsolidatedJSONWriter {
    return &ConsolidatedJSONWriter{filepath: filepath}
}

// ConsolidatedJSONOutput represents the consolidated JSON output structure
type ConsolidatedJSONOutput struct {
    Metadata Metadata                      `json:"metadata"`
    Results  []result.ConsolidatedResult   `json:"results"`
}

// WriteConsolidated outputs consolidated results to JSON file
func (w *ConsolidatedJSONWriter) WriteConsolidated(results []result.ConsolidatedResult, metadata Metadata) error {
    output := ConsolidatedJSONOutput{
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