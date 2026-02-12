package main

import (
	"dns_query_utility/config"
	"dns_query_utility/output"
	"dns_query_utility/parser"
	"dns_query_utility/result"
	"dns_query_utility/worker"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	// Parse arguments
	csvFile, dnsArg, outputFile, formatArg, timeoutArg, retryArg, showHelp := parseArgs(os.Args[1:])

	if showHelp {
		printUsage()
		os.Exit(0)
	}

	if csvFile == "" {
		fmt.Println("Error: CSV file not specified")
		fmt.Println("\nUsage: dns_query_utility <csv_file> [options]")
		fmt.Println("Run 'dns_query_utility --help' for more information")
		os.Exit(1)
	}

	fmt.Println("=== DNS Query Utility ===")

	// Parse DNS servers
	var dnsServers []string
	if dnsArg != "" {
		dnsServers = strings.Fields(dnsArg)
		fmt.Printf("DNS Server(s): %v\n", dnsArg)
	}

	ipv4Server, ipv4Port, ipv6Server, ipv6Port, err := config.ParseDNSServers(dnsServers...)
	if err != nil {
		fmt.Printf("\nError parsing DNS servers: %v\n", err)
		os.Exit(1)
	}

	// Apply DNS defaults
	if ipv4Server == "" && ipv6Server == "" {
		ipv4Server = "8.8.8.8"
		ipv4Port = 53
		ipv6Server = "2001:4860:4860::8888"
		ipv6Port = 53
		fmt.Println("Using default DNS servers (Google Public DNS)")
	} else {
		fmt.Println("Using custom DNS servers")
	}

	// Parse timeout
	timeout := 5 * time.Second
	if timeoutArg != "" {
		t, err := time.ParseDuration(timeoutArg)
		if err != nil {
			fmt.Printf("Error: invalid timeout '%s' (use format like 5s, 500ms, 1m)\n", timeoutArg)
			os.Exit(1)
		}
		if t <= 0 {
			fmt.Println("Error: timeout must be positive")
			os.Exit(1)
		}
		timeout = t
	}

	// Parse retry count
	retryCount := 2
	if retryArg != "" {
		rc, err := strconv.Atoi(retryArg)
		if err != nil || rc < 0 || rc > 10 {
			fmt.Printf("Error: invalid retry count '%s' (must be 0-10)\n", retryArg)
			os.Exit(1)
		}
		retryCount = rc
	}

	// Parse CSV
	specs, err := parser.ParseCSV(csvFile)
	if err != nil {
		fmt.Printf("\nError parsing CSV: %v\n", err)
		os.Exit(1)
	}

	// Auto-calculate workers
	workerCount := config.CalculateOptimalWorkers(len(specs))

	// Create configuration
	cfg := config.Config{
		DNSServerIPv4: ipv4Server,
		DNSServerIPv6: ipv6Server,
		DNSPort:       ipv4Port,
		Timeout:       timeout,
		RetryCount:    retryCount,
		WorkerCount:   workerCount,
	}

	if err := cfg.Validate(); err != nil {
		fmt.Printf("Configuration error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nDNS Configuration:\n")
	fmt.Printf("  IPv4 Server:   %s:%d\n", cfg.DNSServerIPv4, ipv4Port)
	fmt.Printf("  IPv6 Server:   %s:%d\n", cfg.DNSServerIPv6, ipv6Port)
	fmt.Printf("  Timeout:       %v\n", cfg.Timeout)
	fmt.Printf("  Retry Count:   %d\n", cfg.RetryCount)
	fmt.Printf("  Query Count:   %d\n", len(specs))
	fmt.Printf("  Workers:       %d (auto-scaled)\n\n", cfg.WorkerCount)

	fmt.Println("Executing DNS Queries (Concurrent):")
	fmt.Println("====================================")

	// Execute queries
	startTime := time.Now()
	results := worker.ExecuteWithProgress(specs, cfg)
	totalDuration := time.Since(startTime)

	fmt.Printf("\nAll queries completed in %v\n", totalDuration)

	// Determine output format
	format := output.FormatJSON // Default: JSON
	if formatArg != "" {
		switch strings.ToLower(formatArg) {
		case "csv":
			format = output.FormatCSV
		case "json":
			format = output.FormatJSON
		case "all":
			format = output.FormatAll
		default:
			fmt.Printf("Error: unknown format '%s' (use: csv, json, all)\n", formatArg)
			os.Exit(1)
		}
	}

	// Determine output file name
	if outputFile == "" {
		outputFile = "result" // Default base name
	}

	// Build metadata
	metadata := buildMetadata(results, totalDuration, cfg, ipv4Server, ipv4Port, ipv6Server, ipv6Port)

	// Generate output file(s)
	switch format {
	case output.FormatJSON:
		jsonPath := output.ChangeExtension(outputFile, ".json")
		if err := output.WriteOutput(jsonPath, output.FormatJSON, results, metadata); err != nil {
			fmt.Printf("\nError writing JSON file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n✓ JSON output written to: %s\n", jsonPath)

	case output.FormatCSV:
		csvPath := output.ChangeExtension(outputFile, ".csv")
		if err := output.WriteOutput(csvPath, output.FormatCSV, results, metadata); err != nil {
			fmt.Printf("\nError writing CSV file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n✓ CSV output written to: %s\n", csvPath)

	case output.FormatAll:
		jsonPath := output.ChangeExtension(outputFile, ".json")
		csvPath := output.ChangeExtension(outputFile, ".csv")

		if err := output.WriteOutput(jsonPath, output.FormatJSON, results, metadata); err != nil {
			fmt.Printf("\nError writing JSON file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n✓ JSON output written to: %s\n", jsonPath)

		if err := output.WriteOutput(csvPath, output.FormatCSV, results, metadata); err != nil {
			fmt.Printf("\nError writing CSV file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ CSV output written to: %s\n", csvPath)
	}

	// Console display
	fmt.Println("\nDetailed Results:")
	fmt.Println("=================")
	displayResults(results)
	printSummary(results, totalDuration, cfg.WorkerCount)
}

func buildMetadata(results []result.QueryResult, duration time.Duration, cfg config.Config, ipv4 string, ipv4Port int, ipv6 string, ipv6Port int) output.Metadata {
	successCount := 0
	noAnswerCount := 0
	errorCount := 0
	var totalLatencyMs float64

	for _, res := range results {
		totalLatencyMs += res.LatencyMs
		switch res.Status {
		case result.StatusSuccess:
			successCount++
		case result.StatusNoAnswer:
			noAnswerCount++
		default:
			errorCount++
		}
	}

	avgLatency := float64(0)
	if len(results) > 0 {
		avgLatency = totalLatencyMs / float64(len(results))
	}

	return output.Metadata{
		Timestamp:         time.Now(),
		TotalQueries:      len(results),
		SuccessfulQueries: successCount,
		NoAnswerQueries:   noAnswerCount,
		FailedQueries:     errorCount,
		TotalDurationMs:   duration.Milliseconds(),
		AverageLatencyMs:  avgLatency,
		QueriesPerSecond:  float64(len(results)) / duration.Seconds(),
		DNSServerIPv4:     fmt.Sprintf("%s:%d", ipv4, ipv4Port),
		DNSServerIPv6:     fmt.Sprintf("%s:%d", ipv6, ipv6Port),
		WorkersUsed:       cfg.WorkerCount,
		TimeoutSeconds:    cfg.Timeout.Seconds(),
		RetryCount:        cfg.RetryCount,
	}
}

func displayResults(results []result.QueryResult) {
	for i, res := range results {
		fmt.Printf("%d. %s (type=%s transport=%s network=%s)\n",
			i+1, res.Domain, res.QueryType, res.Transport, res.IPVersion)

		statusIcon := getStatusIcon(string(res.Status))
		fmt.Printf("   Status:        %s %s\n", statusIcon, res.Status)
		fmt.Printf("   Latency:       %.2fms\n", res.LatencyMs)
		fmt.Printf("   Response Code: %d\n", res.ResponseCode)

		switch res.Status {
		case result.StatusSuccess:
			if len(res.Records) > 0 {
				fmt.Printf("   Records:       %v\n", res.Records)
			}
			if len(res.ResolvedIPs) > 0 {
				fmt.Printf("   Resolved IPs:  %v\n", res.ResolvedIPs)
			}

		case result.StatusNoAnswer:
			if len(res.Records) > 0 {
				fmt.Printf("   Records:       %v\n", res.Records)
			}
			fmt.Printf("   Note:          %s\n", res.Error)

		default:
			if res.Error != "" {
				fmt.Printf("   Error:         %s\n", res.Error)
			}
		}

		fmt.Println()
	}
}

func printSummary(results []result.QueryResult, totalDuration time.Duration, workerCount int) {
	fmt.Println("Summary:")
	fmt.Println("========")

	successCount := 0
	noAnswerCount := 0
	errorCount := 0
	var totalLatencyMs float64

	for _, res := range results {
		totalLatencyMs += res.LatencyMs
		switch res.Status {
		case result.StatusSuccess:
			successCount++
		case result.StatusNoAnswer:
			noAnswerCount++
		default:
			errorCount++
		}
	}

	avgLatency := float64(0)
	if len(results) > 0 {
		avgLatency = totalLatencyMs / float64(len(results))
	}

	fmt.Printf("Total Queries:    %d\n", len(results))
	fmt.Printf("Workers Used:     %d\n", workerCount)
	fmt.Printf("Successful:       %d\n", successCount)
	fmt.Printf("No Answer:        %d\n", noAnswerCount)
	fmt.Printf("Errors:           %d\n", errorCount)
	fmt.Printf("Total Time:       %v\n", totalDuration)
	fmt.Printf("Average Latency:  %.2fms\n", avgLatency)
	if totalDuration.Seconds() > 0 {
		fmt.Printf("Queries/Second:   %.2f\n", float64(len(results))/totalDuration.Seconds())
	}
}

func parseArgs(args []string) (string, string, string, string, string, string, bool) {
	var csvFile string
	var dnsArg string
	var outputFile string
	var formatArg string
	var timeoutArg string
	var retryArg string
	showHelp := false

	i := 0
	for i < len(args) {
		arg := args[i]

		switch {
		case arg == "--help" || arg == "-h":
			showHelp = true
			i++

		case arg == "--dns":
			if i+1 < len(args) {
				i++
				dnsArg = args[i]
			} else {
				fmt.Println("Error: --dns requires a value")
				os.Exit(1)
			}
			i++

		case strings.HasPrefix(arg, "--dns="):
			dnsArg = strings.TrimPrefix(arg, "--dns=")
			i++

		case arg == "--output" || arg == "-o":
			if i+1 < len(args) {
				i++
				outputFile = args[i]
			} else {
				fmt.Println("Error: --output requires a value")
				os.Exit(1)
			}
			i++

		case strings.HasPrefix(arg, "--output="):
			outputFile = strings.TrimPrefix(arg, "--output=")
			i++

		case arg == "--format" || arg == "-f":
			if i+1 < len(args) {
				i++
				formatArg = args[i]
			} else {
				fmt.Println("Error: --format requires a value")
				os.Exit(1)
			}
			i++

		case strings.HasPrefix(arg, "--format="):
			formatArg = strings.TrimPrefix(arg, "--format=")
			i++

		case arg == "--timeout" || arg == "-t":
			if i+1 < len(args) {
				i++
				timeoutArg = args[i]
			} else {
				fmt.Println("Error: --timeout requires a value")
				os.Exit(1)
			}
			i++

		case strings.HasPrefix(arg, "--timeout="):
			timeoutArg = strings.TrimPrefix(arg, "--timeout=")
			i++

		case arg == "--retry" || arg == "-r":
			if i+1 < len(args) {
				i++
				retryArg = args[i]
			} else {
				fmt.Println("Error: --retry requires a value")
				os.Exit(1)
			}
			i++

		case strings.HasPrefix(arg, "--retry="):
			retryArg = strings.TrimPrefix(arg, "--retry=")
			i++

		case strings.HasPrefix(arg, "-"):
			fmt.Printf("Error: unknown flag '%s'\n", arg)
			fmt.Println("Run 'dns_query_utility --help' for usage")
			os.Exit(1)

		default:
			if csvFile == "" {
				csvFile = arg
			} else {
				fmt.Printf("Error: unexpected argument '%s'\n", arg)
				os.Exit(1)
			}
			i++
		}
	}

	return csvFile, dnsArg, outputFile, formatArg, timeoutArg, retryArg, showHelp
}

func printUsage() {
	fmt.Println(`
╔══════════════════════════════════════════════════════════════╗
║                    DNS Query Utility                         ║
║              Concurrent DNS Querying Tool                    ║
╚══════════════════════════════════════════════════════════════╝

USAGE:
  dns_query_utility <csv_file> [options]

DESCRIPTION:
  Performs bulk DNS queries concurrently from a CSV input file.
  By default, generates a JSON output file with detailed results.

INPUT CSV FORMAT:
  The CSV file should have these columns (with header row):

    domain,query_type,transport,network

  Columns:
    domain      - Domain name to query (e.g., google.com)
    query_type  - DNS record type: A, AAAA, MX, TXT, NS, SOA, CNAME, PTR, SRV
    transport   - Protocol: udp or tcp
    network     - IP version: ipv4 or ipv6

  Example CSV:
    domain,query_type,transport,network
    google.com,A,udp,ipv4
    cloudflare.com,AAAA,tcp,ipv6
    example.com,MX,udp,ipv4
    github.com,TXT,tcp,ipv4

DNS OPTIONS:
  --dns <server>
      DNS server(s) to use for queries.
      Accepts IPv4, IPv6, or both with optional port.

      Examples:
        --dns 9.9.9.9                           Single server
        --dns "1.1.1.1 2606:4700:4700::1111"    IPv4 + IPv6
        --dns 9.9.9.9:5353                       Custom port
        --dns "9.9.9.9:54 [2620:fe::fe]:5353"   Different ports

      Default: 8.8.8.8:53 (IPv4) and 2001:4860:4860::8888:53 (IPv6)

  -t, --timeout <duration>
      Maximum time to wait for each DNS query response.
      Format: Go duration string (e.g., 5s, 500ms, 1m)

      Default: 5s
      Recommended:
        Fast network:     2s - 3s
        Normal network:   5s (default)
        Slow/satellite:   15s - 30s

  -r, --retry <count>
      Number of times to retry a failed query before giving up.
      Range: 0-10

      Default: 2
      Use 0 for fail-fast behavior.
      Use 3-5 for unreliable networks.

OUTPUT OPTIONS:
  -o, --output <filename>
      Base name for output file(s).
      Extension is automatically added based on format.
      If not specified, defaults to "result"

      Examples:
        --output myresults       → myresults.json
        --output report.json     → report.json
        --output /tmp/dns_test   → /tmp/dns_test.json

  -f, --format <type>
      Output file format. Options:
        json   - JSON with metadata and results (default)
        csv    - Comma-separated values
        all    - Generate both JSON and CSV files

      Default: json

      Examples:
        --format json    → result.json
        --format csv     → result.csv
        --format all     → result.json + result.csv

OTHER:
  -h, --help
      Show this help message and exit.

FEATURES:
  • Auto-scaling worker pool (1-50 concurrent workers)
  • Real-time progress tracking
  • Automatic retry on failure
  • Support for all major DNS record types
  • Independent transport (UDP/TCP) and network (IPv4/IPv6) selection
  • Rich JSON output with metadata and statistics
  • CSV output for spreadsheet analysis

OUTPUT FILE DETAILS:

  JSON Output (default):
    Contains metadata section with:
      - Timestamp, total queries, success/failure counts
      - Total duration, average latency, queries/second
      - DNS server configuration used
      - Worker count and timeout settings
    Contains results array with:
      - Domain, query type, transport, network
      - Status, latency, response code
      - Resolved IPs, DNS records, errors

  CSV Output:
    Columns: domain, query_type, transport, network, status,
             latency_ms, response_code, resolved_ips, records,
             error, timestamp

STATUS VALUES:
  success    - Query returned valid records
  no_answer  - Query succeeded but no matching records found
  nxdomain   - Domain does not exist
  servfail   - DNS server failure
  refused    - DNS server refused the query
  timeout    - Query timed out
  error      - Other error occurred

EXAMPLES:

  Basic usage (generates result.json):
    $ dns_query_utility queries.csv

  Custom DNS server:
    $ dns_query_utility queries.csv --dns 1.1.1.1

  Generate CSV output:
    $ dns_query_utility queries.csv --format csv

  Generate both JSON and CSV:
    $ dns_query_utility queries.csv --format all

  Custom output name:
    $ dns_query_utility queries.csv --output dns_report

  Full options:
    $ dns_query_utility queries.csv \
        --dns "1.1.1.1 2606:4700:4700::1111" \
        --timeout 10s \
        --retry 3 \
        --output report \
        --format all

  Quick test with short flags:
    $ dns_query_utility test.csv -t 3s -r 0 -o test_result -f json

POPULAR DNS SERVERS:
  Google:      8.8.8.8         / 2001:4860:4860::8888
  Cloudflare:  1.1.1.1         / 2606:4700:4700::1111
  Quad9:       9.9.9.9         / 2620:fe::fe
  OpenDNS:     208.67.222.222  / 2620:119:35::35`)
}

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
