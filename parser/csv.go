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
	if err!=nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	specs := make([]query.QuerySpec, 0, len(rows)-1)

	for i, row := range rows {
		if i==0 {
			continue
		}

		if len(row) !=3 {
			fmt.Printf("Warning: Skipping row %d - expected 3 columns, got %d\n", i+1, len(row))
			continue
		}
		domain := row[0]
		ipVersionStr := row[1]
		transportStr := row[2]

		ipVersion, err := query.ParseIPVersion(ipVersionStr)
		if err!=nil {
			fmt.Printf("Warning: Skipping row %d - invalid IP version '%s': %v\n", i+1, ipVersionStr, err)
			continue
		}

		transport, err := query.ParseTransport(transportStr)
		if err != nil {
			fmt.Printf("Warning: Skipping row %d - invalid transport '%s': %v\n", i+1, transportStr, err)
			continue
		}

		spec:= query.QuerySpec {
			Domain: domain,
			IPVersion: ipVersion,
			Transport: transport,
		}

		if err:= spec.Validate(); err!=nil {
			fmt.Printf("Warning: Skipping row %d - validation failed: %v\n", i+1, err)
			continue
		}

		specs = append(specs, spec)

	}

	if len(specs) ==0 {
		return nil, fmt.Errorf("no valid query specifications found in CSV")
	}

	fmt.Printf("Successfully parsed %d valid queries from CSV\n", len(specs))
	return specs, nil
}