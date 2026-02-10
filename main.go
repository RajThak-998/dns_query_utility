package main 

import (
	"fmt"
	"dns_query_utility/parser"
	"os"
)

func main() {
	fmt.Println("DNS Query Utility - Phase 2")

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <csv_file>")
        fmt.Println("Example: go run main.go queries.csv")
        os.Exit(1)
	}

	csvFile := os.Args[1]

	specs, err:= parser.ParseCSV(csvFile)
	if err!=nil {
		fmt.Printf("Error parsing CSV: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nParsed Query Specifications:")
    fmt.Println("----------------------------")
    for i, spec := range specs {
        fmt.Printf("%d. Domain: %s | IP: %s | Transport: %s\n",
            i+1, spec.Domain, spec.IPVersion, spec.Transport)
    }

    fmt.Printf("\nTotal valid queries: %d\n", len(specs))
	
}