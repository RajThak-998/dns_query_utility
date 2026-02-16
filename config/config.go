package config

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// Config holds DNS query configuration
type Config struct {
	DNSServerIPv4     string
	DNSServerIPv6     string
	DNSPort           int
	Timeout           time.Duration
	RetryCount        int
	WorkerCount       int
	TransportOverride string // NEW: Optional transport override ("tcp", "udp", or "")
	QueryAllTypes     bool   // NEW: If true, query all record types
}

// Validate checks if configuration is valid
func Validate(cfg Config) error {
	if cfg.DNSServerIPv4 == "" && cfg.DNSServerIPv6 == "" {
		return errors.New("at least one DNS server (IPv4 or IPv6) must be specified")
	}

	if cfg.DNSPort < 1 || cfg.DNSPort > 65535 {
		return errors.New("DNS port must be between 1 and 65535")
	}

	if cfg.Timeout <= 0 {
		return errors.New("timeout must be positive")
	}

	if cfg.RetryCount < 0 || cfg.RetryCount > 10 {
		return errors.New("retry count must be between 0 and 10")
	}

	if cfg.WorkerCount < MinWorkers || cfg.WorkerCount > AbsoluteMaxWorkers {
		return fmt.Errorf("worker count must be between %d and %d", MinWorkers, AbsoluteMaxWorkers)
	}

	// Validate transport override if specified
	if cfg.TransportOverride != "" {
		if cfg.TransportOverride != "tcp" && cfg.TransportOverride != "udp" {
			return fmt.Errorf("transport override must be 'tcp' or 'udp', got '%s'", cfg.TransportOverride)
		}
	}

	return nil
}

const (
	MinWorkers         = 1
	MaxWorkers         = 50
	AbsoluteMaxWorkers = 200
	WorkersPerQuery    = 5
)

// CalculateOptimalWorkers determines the best worker count based on query count
func CalculateOptimalWorkers(queryCount int) int {
	if queryCount <= 0 {
		return MinWorkers
	}

	var workers int

	if queryCount <= 10 {
		workers = queryCount
	} else if queryCount <= 50 {
		workers = queryCount / WorkersPerQuery
		if workers < 5 {
			workers = 5
		}
	} else if queryCount <= 250 {
		workers = (queryCount / WorkersPerQuery) + 5
	} else {
		workers = queryCount / 3
		if workers > AbsoluteMaxWorkers {
			workers = AbsoluteMaxWorkers
		}
	}

	if workers < MinWorkers {
		workers = MinWorkers
	}
	if workers > MaxWorkers {
		workers = MaxWorkers
	}

	return workers
}

// ParseDNSServers parses DNS server arguments
func ParseDNSServers(args ...string) (ipv4Server string, ipv4Port int, ipv6Server string, ipv6Port int, err error) {
	ipv4Port = 53
	ipv6Port = 53

	if len(args) == 0 {
		return "", 53, "", 53, nil
	}

	if len(args) > 2 {
		return "", 0, "", 0, errors.New("too many DNS server arguments (max 2: ipv4 and ipv6)")
	}

	server1, port1, err := parseServerAddress(args[0])
	if err != nil {
		return "", 0, "", 0, fmt.Errorf("invalid DNS server '%s': %w", args[0], err)
	}

	isIPv6_1 := isIPv6Address(server1)

	if len(args) == 1 {
		if isIPv6_1 {
			ipv4Server = server1
			ipv4Port = port1
			ipv6Server = server1
			ipv6Port = port1
		} else {
			ipv4Server = server1
			ipv4Port = port1
			ipv6Server = server1
			ipv6Port = port1
		}
		return ipv4Server, ipv4Port, ipv6Server, ipv6Port, nil
	}

	server2, port2, err := parseServerAddress(args[1])
	if err != nil {
		return "", 0, "", 0, fmt.Errorf("invalid DNS server '%s': %w", args[1], err)
	}

	isIPv6_2 := isIPv6Address(server2)

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
