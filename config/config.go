package config

import (
    "errors"
    "fmt"
    "net"
    "strconv"
    "strings"
    "time"
)

const (
    // Worker pool limits
    MinWorkers = 1   // Minimum concurrent workers
    MaxWorkers = 50  // Maximum concurrent workers (safe for most machines)
    
    // Auto-scaling: 1 worker per N queries
    WorkersPerQuery = 5 // For every 5 queries, add 1 worker
)

// Config holds global settings that apply to all DNS queries
type Config struct {
    DNSServerIPv4 string        // IPv4 DNS server address
    DNSServerIPv6 string        // IPv6 DNS server address
    DNSPort       int           // DNS port (default 53)
    Timeout       time.Duration // Query timeout
    RetryCount    int           // Number of retries for failed queries
    WorkerCount   int           // Number of concurrent workers (auto-calculated if 0)
}

// CalculateOptimalWorkers determines the best worker count based on query count
func CalculateOptimalWorkers(queryCount int) int {
    if queryCount <= 0 {
        return MinWorkers
    }
    
    // Strategy: 1 worker for every WorkersPerQuery queries
    // For small batches: use exactly queryCount workers
    // For large batches: scale up but cap at MaxWorkers
    
    var workers int
    
    if queryCount <= 10 {
        // For small batches, use exactly the number of queries
        workers = queryCount
    } else if queryCount <= 50 {
        // Medium batches: 20% of queries as workers
        workers = queryCount / WorkersPerQuery
        if workers < 5 {
            workers = 5
        }
    } else {
        // Large batches: scale logarithmically
        workers = (queryCount / WorkersPerQuery) + 5
    }
    
    // Ensure within bounds
    if workers < MinWorkers {
        workers = MinWorkers
    }
    if workers > MaxWorkers {
        workers = MaxWorkers
    }
    
    return workers
}

// Validate checks if the configuration values are sensible
func (c *Config) Validate() error {
    if c.DNSServerIPv4 == "" && c.DNSServerIPv6 == "" {
        return errors.New("at least one DNS server (IPv4 or IPv6) must be configured")
    }

    if c.DNSPort < 1 || c.DNSPort > 65535 {
        return fmt.Errorf("invalid DNS port %d: must be between 1 and 65535", c.DNSPort)
    }

    if c.Timeout <= 0 {
        return errors.New("timeout must be positive")
    }

    if c.RetryCount < 0 {
        return errors.New("retry count cannot be negative")
    }

    if c.WorkerCount < 0 {
        return errors.New("worker count cannot be negative")
    }

    return nil
}

// ParseDNSServers parses DNS server arguments from CLI
func ParseDNSServers(args ...string) (ipv4Server string, ipv4Port int, ipv6Server string, ipv6Port int, err error) {
    // Defaults
    ipv4Port = 53
    ipv6Port = 53

    if len(args) == 0 {
        return "", 53, "", 53, nil
    }

    if len(args) > 2 {
        return "", 0, "", 0, errors.New("too many DNS server arguments (max 2: ipv4 and ipv6)")
    }

    // Parse first server
    server1, port1, err := parseServerAddress(args[0])
    if err != nil {
        return "", 0, "", 0, fmt.Errorf("invalid DNS server '%s': %w", args[0], err)
    }

    isIPv6_1 := isIPv6Address(server1)

    // If only one server provided, use it for both
    if len(args) == 1 {
        ipv4Server = server1
        ipv4Port = port1
        ipv6Server = server1
        ipv6Port = port1
        return ipv4Server, ipv4Port, ipv6Server, ipv6Port, nil
    }

    // Two servers provided
    server2, port2, err := parseServerAddress(args[1])
    if err != nil {
        return "", 0, "", 0, fmt.Errorf("invalid DNS server '%s': %w", args[1], err)
    }

    isIPv6_2 := isIPv6Address(server2)

    // Assign based on IP version
    if !isIPv6_1 && isIPv6_2 {
        ipv4Server = server1
        ipv4Port = port1
        ipv6Server = server2
        ipv6Port = port2
    } else if isIPv6_1 && !isIPv6_2 {
        ipv4Server = server2
        ipv4Port = port2
        ipv6Server = server1
        ipv6Port = port1
    } else if !isIPv6_1 && !isIPv6_2 {
        ipv4Server = server1
        ipv4Port = port1
        ipv6Server = server1
        ipv6Port = port1
        fmt.Println("Warning: Both DNS servers are IPv4, using first for IPv6 queries as well")
    } else {
        ipv4Server = server1
        ipv4Port = port1
        ipv6Server = server1
        ipv6Port = port1
        fmt.Println("Warning: Both DNS servers are IPv6, using first for IPv4 queries as well")
    }

    return ipv4Server, ipv4Port, ipv6Server, ipv6Port, nil
}

func parseServerAddress(input string) (string, int, error) {
    input = strings.TrimSpace(input)

    // Handle IPv6 bracket notation: [2620:fe::fe]:5353
    if strings.HasPrefix(input, "[") {
        closeBracket := strings.Index(input, "]")
        if closeBracket == -1 {
            return "", 0, errors.New("unclosed bracket in IPv6 address")
        }

        ipv6 := input[1:closeBracket]
        if net.ParseIP(ipv6) == nil {
            return "", 0, fmt.Errorf("invalid IPv6 address: %s", ipv6)
        }

        if len(input) > closeBracket+1 {
            if input[closeBracket+1] != ':' {
                return "", 0, errors.New("expected ':' after IPv6 bracket")
            }
            portStr := input[closeBracket+2:]
            port, err := strconv.Atoi(portStr)
            if err != nil || port < 1 || port > 65535 {
                return "", 0, fmt.Errorf("invalid port: %s", portStr)
            }
            return ipv6, port, nil
        }

        return ipv6, 53, nil
    }

    // Handle IPv4:PORT or plain IP
    parts := strings.Split(input, ":")

    if len(parts) == 1 {
        ip := parts[0]
        if net.ParseIP(ip) == nil {
            return "", 0, fmt.Errorf("invalid IP address: %s", ip)
        }
        return ip, 53, nil
    }

    if len(parts) == 2 {
        ip := parts[0]
        if net.ParseIP(ip) == nil {
            return "", 0, fmt.Errorf("invalid IP address: %s", ip)
        }

        port, err := strconv.Atoi(parts[1])
        if err != nil || port < 1 || port > 65535 {
            return "", 0, fmt.Errorf("invalid port: %s", parts[1])
        }

        return ip, port, nil
    }

    // Could be plain IPv6 without brackets
    if net.ParseIP(input) != nil {
        return input, 53, nil
    }

    return "", 0, errors.New("invalid format (use IP, IP:PORT, or [IPv6]:PORT)")
}

func isIPv6Address(ip string) bool {
    parsedIP := net.ParseIP(ip)
    if parsedIP == nil {
        return false
    }
    return parsedIP.To4() == nil
}