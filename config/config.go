package config

import (
	"errors"
	"time"
)

// Config holds global settings that apply to all DNS queries
type Config struct {
	DNSServerIPv4 string        // IPv4 DNS server (e.g., "8.8.8.8")
	DNSServerIPv6 string        // IPv6 DNS server (e.g., "2001:4860:4860::8888")
	DNSPort       int           // Port number (typically 53)
	Timeout       time.Duration // How long to wait for a response
	RetryCount    int           // Number of retries on failure
	WorkerCount   int           // Number of concurrent workers
}

// Validate checks if the configuration values are sensible
func (c *Config) Validate() error {
	// At least one DNS server must be provided
	if c.DNSServerIPv4 == "" && c.DNSServerIPv6 == "" {
		return errors.New("at least one DNS server (IPv4 or IPv6) must be configured")
	}

	// Port must be between 1 and 65535
	if c.DNSPort < 1 || c.DNSPort > 65535 {
		return errors.New("DNS port must be between 1 and 65535")
	}

	// Timeout must be positive
	if c.Timeout <= 0 {
		return errors.New("timeout must be positive")
	}

	// Retry count should be non-negative
	if c.RetryCount < 0 {
		return errors.New("retry count cannot be negative")
	}

	// Worker count must be at least 1
	if c.WorkerCount < 1 {
		return errors.New("worker count must be at least 1")
	}

	return nil
}
