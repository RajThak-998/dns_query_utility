package main

import (
	"dns_query_utility/config"
	"dns_query_utility/output"
	"dns_query_utility/parser"
	"dns_query_utility/query"
	"dns_query_utility/result"
	"dns_query_utility/worker"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	// Parse arguments with new flags
	csvFile, dnsArg, outputFile, formatArg, timeoutArg, retryArg, workersArg, transportOverride, queryAll, showHelp := parseArgs(os.Args[1:])

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

	fmt.Printf("Successfully parsed %d queries from CSV\n", len(specs))

	// Check for ANY + --query-all conflict
	checkForANYWithQueryAll(specs, queryAll)

	// Apply overrides BEFORE calculating workers
	originalCount := len(specs)

	// 1. Apply transport override
	if transportOverride != "" {
		specs = applyTransportOverride(specs, transportOverride)
		fmt.Printf("✓ Transport override: All queries will use %s\n", strings.ToUpper(transportOverride))
	}

	// 2. Expand to all query types if requested
	if queryAll {
		specs = expandToAllTypes(specs)
		fmt.Printf("✓ Query-all mode: Expanded %d domains to %d queries (all record types)\n", originalCount, len(specs))
		fmt.Printf("✓ Output will be consolidated (one record per domain)\n")
	}

	// Auto-calculate or parse workers
	var workerCount int
	if workersArg != "" {
		wc, err := strconv.Atoi(workersArg)
		if err != nil || wc < config.MinWorkers || wc > config.AbsoluteMaxWorkers {
			fmt.Printf("Error: invalid worker count '%s' (must be %d-%d)\n", workersArg, config.MinWorkers, config.AbsoluteMaxWorkers)
			os.Exit(1)
		}
		workerCount = wc
		fmt.Printf("✓ Manual worker override: Using %d workers\n", workerCount)
	} else {
		workerCount = config.CalculateOptimalWorkers(len(specs))
	}

	// Create configuration
	cfg := config.Config{
		DNSServerIPv4:     ipv4Server,
		DNSServerIPv6:     ipv6Server,
		DNSPort:           ipv4Port,
		Timeout:           timeout,
		RetryCount:        retryCount,
		WorkerCount:       workerCount,
		TransportOverride: transportOverride,
		QueryAllTypes:     queryAll,
	}

	if err := config.Validate(cfg); err != nil {
		fmt.Printf("Configuration error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nDNS Configuration:\n")
	fmt.Printf("  IPv4 Server:   %s:%d\n", cfg.DNSServerIPv4, ipv4Port)
	fmt.Printf("  IPv6 Server:   %s:%d\n", cfg.DNSServerIPv6, ipv6Port)
	fmt.Printf("  Timeout:       %v\n", cfg.Timeout)
	fmt.Printf("  Retry Count:   %d\n", cfg.RetryCount)
	fmt.Printf("  Query Count:   %d\n", len(specs))
	fmt.Printf("  Workers:       %d", cfg.WorkerCount)
	if workersArg != "" {
		fmt.Printf(" (manual override)")
	} else {
		fmt.Printf(" (auto-scaled)")
	}
	fmt.Println("")

	fmt.Println("Executing DNS Queries (Concurrent):")
	fmt.Println("====================================")

	// Execute queries
	startTime := time.Now()
	results := worker.ExecuteWithProgress(specs, cfg)
	totalDuration := time.Since(startTime)

	fmt.Printf("\nAll queries completed in %v\n", totalDuration)

	// Determine output format
	format := output.FormatJSON
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
		outputFile = "result"
	}

	// Build metadata
	metadata := buildMetadata(results, totalDuration, cfg, ipv4Server, ipv4Port, ipv6Server, ipv6Port)

	// Generate output file(s) - consolidate if using --query-all
	consolidate := queryAll

	switch format {
	case output.FormatJSON:
		jsonPath := output.ChangeExtension(outputFile, ".json")
		if err := output.WriteOutput(jsonPath, output.FormatJSON, results, metadata, consolidate); err != nil {
			fmt.Printf("\nError writing JSON file: %v\n", err)
			os.Exit(1)
		}
		if consolidate {
			fmt.Printf("\n✓ Consolidated JSON output written to: %s\n", jsonPath)
		} else {
			fmt.Printf("\n✓ JSON output written to: %s\n", jsonPath)
		}

	case output.FormatCSV:
		csvPath := output.ChangeExtension(outputFile, ".csv")
		if err := output.WriteOutput(csvPath, output.FormatCSV, results, metadata, false); err != nil {
			fmt.Printf("\nError writing CSV file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n✓ CSV output written to: %s\n", csvPath)

	case output.FormatAll:
		jsonPath := output.ChangeExtension(outputFile, ".json")
		csvPath := output.ChangeExtension(outputFile, ".csv")

		if err := output.WriteOutput(jsonPath, output.FormatJSON, results, metadata, consolidate); err != nil {
			fmt.Printf("\nError writing JSON file: %v\n", err)
			os.Exit(1)
		}
		if consolidate {
			fmt.Printf("\n✓ Consolidated JSON output written to: %s\n", jsonPath)
		} else {
			fmt.Printf("\n✓ JSON output written to: %s\n", jsonPath)
		}

		if err := output.WriteOutput(csvPath, output.FormatCSV, results, metadata, false); err != nil {
			fmt.Printf("\nError writing CSV file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ CSV output written to: %s\n", csvPath)
	}

	// Console display
	fmt.Println("\nDetailed Results:")
	fmt.Println("=================")

	if consolidate {
		displayConsolidatedResults(result.ConsolidateResults(results))
	} else {
		displayResults(results)
	}

	printSummary(results, totalDuration, cfg.WorkerCount)
}

// applyTransportOverride overrides transport protocol for all queries
func applyTransportOverride(specs []query.QuerySpec, transport string) []query.QuerySpec {
	var overrideTransport query.Transport
	if transport == "tcp" {
		overrideTransport = query.TCP
	} else {
		overrideTransport = query.UDP
	}

	for i := range specs {
		specs[i].Transport = overrideTransport
	}

	return specs
}

// expandToAllTypes creates queries for all record types for each unique domain
func expandToAllTypes(specs []query.QuerySpec) []query.QuerySpec {
	// Group by domain to avoid duplicates
	domainMap := make(map[string]query.QuerySpec)

	for _, spec := range specs {
		// Use first occurrence of each domain
		if _, exists := domainMap[spec.Domain]; !exists {
			domainMap[spec.Domain] = spec
		}
	}

	// Expand each domain to all query types
	var expanded []query.QuerySpec
	for _, spec := range domainMap {
		// ExpandToAllTypes expects: domain, transport, ipVersion (3 args)
		allTypeSpecs := query.ExpandToAllTypes(spec.Domain, spec.Transport, spec.IPVersion)
		expanded = append(expanded, allTypeSpecs...)
	}

	return expanded
}

// checkForANYWithQueryAll validates that ANY queries aren't combined with --query-all
func checkForANYWithQueryAll(specs []query.QuerySpec, queryAll bool) {
	if !queryAll {
		return
	}

	hasANY := false
	for _, spec := range specs {
		if spec.QueryType == query.QueryTypeANY {
			hasANY = true
			break
		}
	}

	if hasANY {
		fmt.Println("\n⚠️  WARNING: Your CSV contains 'ANY' query type.")
		fmt.Println("   Using --query-all with ANY is redundant.")
		fmt.Println("   Recommendation: Either use --query-all OR use ANY queries, not both.")
		fmt.Println("   The ANY queries will be expanded to individual types.")
	}
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

		// NEW: Display authoritative nameservers
		if len(res.AuthoritativeNS) > 0 {
			fmt.Printf("   Authority NS:  %v\n", res.AuthoritativeNS)
		}

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

// Update the displayConsolidatedResults function

func displayConsolidatedResults(consolidated []result.ConsolidatedResult) {
	for i, cr := range consolidated {
		fmt.Printf("%d. %s\n", i+1, cr.Domain)
		fmt.Printf("   Summary: %d queries, %d successful, %d no-answer, %d failed (avg: %.2fms)\n",
			cr.Summary.TotalQueries, cr.Summary.Successful, cr.Summary.NoAnswer,
			cr.Summary.Failed, cr.Summary.AverageLatencyMs)

		fmt.Println("   Query Types:")
		for qType, typeRes := range cr.QueryTypes {
			statusIcon := getStatusIcon(string(typeRes.Status))
			fmt.Printf("     [%s] %-6s: %s %s (%.2fms)",
				statusIcon, qType, typeRes.Transport, typeRes.IPVersion, typeRes.LatencyMs)

			if typeRes.Status == result.StatusSuccess {
				if len(typeRes.ResolvedIPs) > 0 {
					fmt.Printf(" → %v", typeRes.ResolvedIPs)
				} else if len(typeRes.Records) > 0 {
					fmt.Printf(" → %v", typeRes.Records)
				}
			} else if typeRes.Error != "" {
				fmt.Printf(" [%s]", typeRes.Error)
			}

			// NEW: Display authoritative NS if present
			if len(typeRes.AuthoritativeNS) > 0 {
				fmt.Printf(" | Auth: %v", typeRes.AuthoritativeNS)
			}

			fmt.Println()
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

func parseArgs(args []string) (string, string, string, string, string, string, string, string, bool, bool) {
	var csvFile string
	var dnsArg string
	var outputFile string
	var formatArg string
	var timeoutArg string
	var retryArg string
	var workersArg string
	var transportOverride string
	queryAll := false
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

		case arg == "--workers" || arg == "-w":
			if i+1 < len(args) {
				i++
				workersArg = args[i]
			} else {
				fmt.Println("Error: --workers requires a value")
				os.Exit(1)
			}
			i++

		case strings.HasPrefix(arg, "--workers="):
			workersArg = strings.TrimPrefix(arg, "--workers=")
			i++

		case arg == "--transport":
			if i+1 < len(args) {
				i++
				transportOverride = strings.ToLower(args[i])
				if transportOverride != "tcp" && transportOverride != "udp" {
					fmt.Printf("Error: --transport must be 'tcp' or 'udp', got '%s'\n", transportOverride)
					os.Exit(1)
				}
			} else {
				fmt.Println("Error: --transport requires a value (tcp or udp)")
				os.Exit(1)
			}
			i++

		case strings.HasPrefix(arg, "--transport="):
			transportOverride = strings.ToLower(strings.TrimPrefix(arg, "--transport="))
			if transportOverride != "tcp" && transportOverride != "udp" {
				fmt.Printf("Error: --transport must be 'tcp' or 'udp', got '%s'\n", transportOverride)
				os.Exit(1)
			}
			i++

		case arg == "--query-all":
			queryAll = true
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

	return csvFile, dnsArg, outputFile, formatArg, timeoutArg, retryArg, workersArg, transportOverride, queryAll, showHelp
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
    query_type  - DNS record type: A, AAAA, MX, TXT, NS, SOA, CNAME, PTR, SRV, ANY
    transport   - Protocol: udp or tcp
    network     - IP version: ipv4 or ipv6

  Example CSV:
    domain,query_type,transport,network
    google.com,A,udp,ipv4
    cloudflare.com,ANY,tcp,ipv4
    example.com,MX,udp,ipv4

DNS OPTIONS:
  --dns <server>
      DNS server(s) to use for queries.
      Default: 8.8.8.8:53 (IPv4) and 2001:4860:4860::8888:53 (IPv6)

  -t, --timeout <duration>
      Maximum time to wait for each DNS query response.
      Default: 5s

  -r, --retry <count>
      Number of times to retry a failed query.
      Range: 0-10, Default: 2

PERFORMANCE OPTIONS:
  -w, --workers <count>
      Number of concurrent workers (manual override).
      Range: 1-200
      If not specified, auto-scales based on query count.

      Examples:
        --workers 10     Use exactly 10 workers
        --workers 100    Use 100 workers for large batches

OVERRIDE OPTIONS:
  --transport <tcp|udp>
      Override transport protocol for ALL queries.
      Ignores 'transport' column in CSV.

      Examples:
        --transport tcp   Force all queries to use TCP
        --transport udp   Force all queries to use UDP

  --query-all
      Query ALL record types for each domain.
      Expands each domain to: A, AAAA, MX, TXT, NS, SOA, CNAME, PTR, SRV
      Ignores 'query_type' column in CSV.
      
      NOTE: Does NOT include ANY queries (redundant with individual types)
      
      Example: If CSV has 10 domains, this generates 90 queries (10×9 types)

OUTPUT OPTIONS:
  -o, --output <filename>
      Base name for output file(s).
      Default: "result"

  -f, --format <type>
      Output file format: json, csv, all
      Default: json

OTHER:
  -h, --help
      Show this help message.

EXAMPLES:

  Basic usage:
    $ dns_query_utility queries.csv

  Manual worker count:
    $ dns_query_utility queries.csv --workers 20

  Force TCP for all queries:
    $ dns_query_utility queries.csv --transport tcp

  Query all record types:
    $ dns_query_utility queries.csv --query-all

  Combined overrides:
    $ dns_query_utility queries.csv \
        --dns 1.1.1.1 \
        --workers 50 \
        --transport tcp \
        --query-all \
        --output comprehensive_scan \
        --format all

  High-performance scan:
    $ dns_query_utility large_list.csv \
        --dns 1.1.1.1 \
        --workers 100 \
        --timeout 2s \
        --retry 0 \
        --transport udp

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
