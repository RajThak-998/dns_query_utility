package main

import (
    "dns_query_utility/parser"
    "dns_query_utility/query"
    "fmt"
    "os"
)

func main() {
    fmt.Println("=== Phase 3: DNS Packet Construction ===\n")

    // Check if CSV file path is provided
    if len(os.Args) < 2 {
        fmt.Println("Usage: go run main.go <csv_file>")
        fmt.Println("Example: go run main.go queries.csv")
        os.Exit(1)
    }

    csvFile := os.Args[1]

    // Parse CSV
    specs, err := parser.ParseCSV(csvFile)
    if err != nil {
        fmt.Printf("Error parsing CSV: %v\n", err)
        os.Exit(1)
    }

    fmt.Println("Building DNS query packets:")
    fmt.Println("---------------------------")

    // Build DNS packet for each query spec
    for i, spec := range specs {
        packet, err := query.BuildDNSQuery(spec.Domain, spec.IPVersion)
        if err != nil {
            fmt.Printf("%d. %s - Error: %v\n", i+1, spec.Domain, err)
            continue
        }

        // Display packet info
        fmt.Printf("%d. %s (%s) - Packet size: %d bytes\n",
            i+1, spec.Domain, spec.IPVersion, len(packet))

        // Show first 20 bytes in hex format
        fmt.Print("   Hex dump: ")
        for j := 0; j < 20 && j < len(packet); j++ {
            fmt.Printf("%02x ", packet[j])
        }
        fmt.Println("...")
    }

    fmt.Printf("\nSuccessfully built %d DNS query packets\n", len(specs))
}