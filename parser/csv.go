package parser

import (
    "dns_query_utility/query"
    "encoding/csv"
    "fmt"
    "os"
)

func ParseCSV(filepath string) ([]query.QuerySpec, error) {
    file, err := os.Open(filepath)
    if err != nil {
        return nil, fmt.Errorf("failed to open CSV file: %w", err)
    }
    defer file.Close()

    reader := csv.NewReader(file)

    rows, err := reader.ReadAll()
    if err != nil {
        return nil, fmt.Errorf("failed to read CSV: %w", err)
    }

    if len(rows) == 0 {
        return nil, fmt.Errorf("CSV file is empty")
    }

    specs := make([]query.QuerySpec, 0, len(rows)-1)

    for i, row := range rows {
        // Skip header row
        if i == 0 {
            continue
        }

        if len(row) != 4 {
            fmt.Printf("Warning: Skipping row %d - expected 4 columns, got %d\n", i+1, len(row))
            continue
        }

        domain := row[0]
        queryTypeStr := row[1]
        transportStr := row[2]
        ipVersionStr := row[3]

        // Parse query type (A, AAAA, MX, TXT, etc.)
        queryType, err := query.ParseQueryType(queryTypeStr)
        if err != nil {
            fmt.Printf("Warning: Skipping row %d - %v\n", i+1, err)
            continue
        }

        // Parse transport (UDP/TCP)
        transport, err := query.ParseTransport(transportStr)
        if err != nil {
            fmt.Printf("Warning: Skipping row %d - invalid transport '%s': %v\n", i+1, transportStr, err)
            continue
        }

        // Parse IP version (IPv4/IPv6)
        ipVersion, err := query.ParseIPVersion(ipVersionStr)
        if err != nil {
            fmt.Printf("Warning: Skipping row %d - invalid ip_version '%s': %v\n", i+1, ipVersionStr, err)
            continue
        }

        spec := query.QuerySpec{
            Domain:    domain,
            QueryType: queryType,
            Transport: transport,
            IPVersion: ipVersion,
        }

        if err := spec.Validate(); err != nil {
            fmt.Printf("Warning: Skipping row %d - validation failed: %v\n", i+1, err)
            continue
        }

        specs = append(specs, spec)
    }

    if len(specs) == 0 {
        return nil, fmt.Errorf("no valid query specifications found in CSV")
    }

    fmt.Printf("Successfully parsed %d valid queries from CSV\n", len(specs))
    return specs, nil
}