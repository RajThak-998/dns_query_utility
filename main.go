package main

import (
	"dns_query_utility/config"
	"dns_query_utility/parser"
	"dns_query_utility/query"
	"fmt"
	"os"
	"time"
)

func main() {
	fmt.Println("=== DNS Query Utility ===\n")

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <csv_file>")
		fmt.Println("Example: go run main.go queries.csv")
		os.Exit(1)
	}

	csvFile := os.Args[1]

	cfg := config.Config{
		DNSServerIPv4: "8.8.8.8",
		DNSServerIPv6: "2001:4860:4860::8888",
		DNSPort:       53,
		Timeout:       5 * time.Second,
		RetryCount:    2,
		WorkerCount:   5,
	}

	if err := cfg.Validate(); err != nil {
		fmt.Printf("Configuration error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("DNS Servers:\n")
	fmt.Printf("  IPv4: %s:%d\n", cfg.DNSServerIPv4, cfg.DNSPort)
	fmt.Printf("  IPv6: %s:%d\n", cfg.DNSServerIPv6, cfg.DNSPort)
	fmt.Printf("Timeout: %v\n\n", cfg.Timeout)

	specs, err := parser.ParseCSV(csvFile)
	if err != nil {
		fmt.Printf("Error parsing CSV: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nExecuting DNS Queries:")
	fmt.Println("======================")

	for i, spec := range specs {
		fmt.Printf("%d. Querying %s (%s over %s)...\n",
			i+1, spec.Domain, spec.IPVersion, spec.Transport)

		result := query.ExecuteQuery(spec, cfg)

		// Print status with visual indicator
		statusIcon := getStatusIcon(string(result.Status))
		fmt.Printf("   Status:        %s %s\n", statusIcon, result.Status)
		fmt.Printf("   Latency:       %v\n", result.Latency)
		fmt.Printf("   Response Code: %d\n", result.ResponseCode)

		// Display results based on status
		switch result.Status {
		case "success":
			if len(result.Records) > 0 {
				fmt.Printf("   Records:       %v\n", result.Records)
			}
			fmt.Printf("   Resolved IPs:  %v\n", result.ResolvedIPs)

		case "no_answer":
			if len(result.Records) > 0 {
				fmt.Printf("   Records:       %v\n", result.Records)
			}
			fmt.Printf("   Note:          %s\n", result.Error)

		case "nxdomain":
			fmt.Printf("   Error:         %s\n", result.Error)

		case "servfail":
			fmt.Printf("   Error:         %s\n", result.Error)

		case "refused":
			fmt.Printf("   Error:         %s\n", result.Error)

		case "timeout":
			fmt.Printf("   Error:         %s\n", result.Error)

		case "error":
			fmt.Printf("   Error:         %s\n", result.Error)
		}

		fmt.Println()
	}

	fmt.Println("Query execution complete!")
}

// getStatusIcon returns a visual indicator for each status
func getStatusIcon(status string) string {
	switch status {
	case "success":
		return "✓"
	case "no_answer":
		return "⚠"
	case "nxdomain":
		return "✗"
	case "servfail":
		return "✗"
	case "refused":
		return "✗"
	case "timeout":
		return "⏱"
	case "error":
		return "✗"
	default:
		return "?"
	}

}
