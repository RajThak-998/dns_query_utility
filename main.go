package main

import (
    "dns_query_utility/config"
    "dns_query_utility/parser"
    "dns_query_utility/result"
    "dns_query_utility/worker"
    "fmt"
    "os"
    "strings"
    "time"
)

func main() {
    // Parse arguments manually to allow any order
    csvFile, dnsArg, showHelp := parseArgs(os.Args[1:])

    // Show help if requested
    if showHelp {
        printUsage()
        os.Exit(0)
    }

    // Check for CSV file
    if csvFile == "" {
        fmt.Println("Error: CSV file not specified")
        fmt.Println("\nUsage: dns_query_utility <csv_file> [options]")
        fmt.Println("Run 'dns_query_utility --help' for more information")
        os.Exit(1)
    }

    fmt.Println("=== DNS Query Utility ===")

    // Parse DNS servers from --dns flag
    var dnsServers []string
    if dnsArg != "" {
        dnsServers = strings.Fields(dnsArg)
        fmt.Printf("Parsed DNS arguments: %v\n", dnsArg)
    }

    // Parse DNS configuration
    ipv4Server, ipv4Port, ipv6Server, ipv6Port, err := config.ParseDNSServers(dnsServers...)
    if err != nil {
        fmt.Printf("\nError parsing DNS servers: %v\n", err)
        os.Exit(1)
    }

    // Apply defaults if no DNS servers provided
    if ipv4Server == "" && ipv6Server == "" {
        ipv4Server = "8.8.8.8"
        ipv4Port = 53
        ipv6Server = "2001:4860:4860::8888"
        ipv6Port = 53
        fmt.Println("Using default DNS servers (Google Public DNS)")
    } else {
        fmt.Println("Using custom DNS servers")
    }

    // Parse CSV to get query count FIRST (before creating config)
    specs, err := parser.ParseCSV(csvFile)
    if err != nil {
        fmt.Printf("\nError parsing CSV: %v\n", err)
        os.Exit(1)
    }

    // Auto-calculate optimal worker count based on query count
    workerCount := config.CalculateOptimalWorkers(len(specs))

    // Create configuration
    cfg := config.Config{
        DNSServerIPv4: ipv4Server,
        DNSServerIPv6: ipv6Server,
        DNSPort:       ipv4Port,
        Timeout:       5 * time.Second,
        RetryCount:    2,
        WorkerCount:   workerCount, // Auto-calculated!
    }

    if err := cfg.Validate(); err != nil {
        fmt.Printf("Configuration error: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("\nDNS Configuration:\n")
    fmt.Printf("  IPv4 Server:   %s:%d\n", cfg.DNSServerIPv4, ipv4Port)
    fmt.Printf("  IPv6 Server:   %s:%d\n", cfg.DNSServerIPv6, ipv6Port)
    fmt.Printf("  Timeout:       %v\n", cfg.Timeout)
    fmt.Printf("  Query Count:   %d\n", len(specs))
    fmt.Printf("  Workers:       %d (auto-scaled)\n\n", cfg.WorkerCount)

    fmt.Println("Executing DNS Queries (Concurrent):")
    fmt.Println("====================================\n")

    // Execute all queries concurrently with progress
    startTime := time.Now()
    results := worker.ExecuteWithProgress(specs, cfg)
    totalDuration := time.Since(startTime)

    fmt.Printf("\nAll queries completed in %v\n", totalDuration)
    fmt.Println("\nDetailed Results:")
    fmt.Println("=================\n")

    // Display results
    for i, res := range results {
        fmt.Printf("%d. %s (type=%s transport=%s network=%s)\n",
            i+1, res.Domain, res.QueryType, res.Transport, res.IPVersion)

        statusIcon := getStatusIcon(string(res.Status))
        fmt.Printf("   Status:        %s %s\n", statusIcon, res.Status)
        fmt.Printf("   Latency:       %v\n", res.Latency)
        fmt.Printf("   Response Code: %d\n", res.ResponseCode)

        switch res.Status {
        case "success":
            if len(res.Records) > 0 {
                fmt.Printf("   Records:       %v\n", res.Records)
            }
            if len(res.ResolvedIPs) > 0 {
                fmt.Printf("   Resolved IPs:  %v\n", res.ResolvedIPs)
            }

        case "no_answer":
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

    // Summary statistics
    printSummary(results, totalDuration, cfg.WorkerCount)
}

func printSummary(results []result.QueryResult, totalDuration time.Duration, workerCount int) {
    fmt.Println("Summary:")
    fmt.Println("========")

    successCount := 0
    noAnswerCount := 0
    errorCount := 0
    var totalLatency time.Duration

    for _, res := range results {
        totalLatency += res.Latency

        switch res.Status {
        case "success":
            successCount++
        case "no_answer":
            noAnswerCount++
        default:
            errorCount++
        }
    }

    avgLatency := totalLatency / time.Duration(len(results))

    fmt.Printf("Total Queries:    %d\n", len(results))
    fmt.Printf("Workers Used:     %d\n", workerCount)
    fmt.Printf("Successful:       %d\n", successCount)
    fmt.Printf("No Answer:        %d\n", noAnswerCount)
    fmt.Printf("Errors:           %d\n", errorCount)
    fmt.Printf("Total Time:       %v\n", totalDuration)
    fmt.Printf("Average Latency:  %v\n", avgLatency)
    fmt.Printf("Queries/Second:   %.2f\n", float64(len(results))/totalDuration.Seconds())
}

// parseArgs manually parses CLI arguments in any order
func parseArgs(args []string) (string, string, bool) {
    var csvFile string
    var dnsArg string
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

    return csvFile, dnsArg, showHelp
}

func printUsage() {
    fmt.Println("DNS Query Utility")
    fmt.Println("\nUsage:")
    fmt.Println("  dns_query_utility <csv_file> [options]")
    fmt.Println("  dns_query_utility [options] <csv_file>")
    fmt.Println("\nOptions:")
    fmt.Println("  --dns \"<server> [<ipv6_server>]\"")
    fmt.Println("      DNS server(s) to use. Examples:")
    fmt.Println("        --dns 9.9.9.9                           (use for both IPv4 and IPv6)")
    fmt.Println("        --dns \"1.1.1.1 2606:4700:4700::1111\"   (separate IPv4 and IPv6)")
    fmt.Println("        --dns 9.9.9.9:54                        (custom port)")
    fmt.Println("        --dns \"9.9.9.9:54 [2620:fe::fe]:5353\"  (different ports)")
    fmt.Println("      Default: 8.8.8.8:53 and 2001:4860:4860::8888:53")
    fmt.Println("\n  -h, --help")
    fmt.Println("      Show this help message")
    fmt.Println("\nFeatures:")
    fmt.Println("  • Auto-scaling worker pool (1-50 workers based on query count)")
    fmt.Println("  • Concurrent execution for maximum throughput")
    fmt.Println("  • Real-time progress tracking")
    fmt.Println("  • Support for all DNS record types (A, AAAA, MX, TXT, NS, etc.)")
    fmt.Println("  • Independent transport (UDP/TCP) and network (IPv4/IPv6) selection")
    fmt.Println("\nExamples:")
    fmt.Println("  dns_query_utility queries.csv")
    fmt.Println("  dns_query_utility queries.csv --dns 9.9.9.9")
    fmt.Println("  dns_query_utility queries.csv --dns \"1.1.1.1 2606:4700:4700::1111\"")
    fmt.Println("\nPopular Public DNS Servers:")
    fmt.Println("  Google:      8.8.8.8 / 2001:4860:4860::8888")
    fmt.Println("  Cloudflare:  1.1.1.1 / 2606:4700:4700::1111")
    fmt.Println("  Quad9:       9.9.9.9 / 2620:fe::fe")
    fmt.Println("  OpenDNS:     208.67.222.222 / 2620:119:35::35")
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