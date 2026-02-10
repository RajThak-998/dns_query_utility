package config

import (
	"errors"
	"time"
)

type Config struct {
	DNSServer string
	DNSPort int
	Timeout time.Duration
	RetryCount int
	WorkerCount int
}

func (c *Config) Validate() error {
	if c.DNSServer=="" {
		return errors.New("DNS server address cannot be empty")
	}

	if c.DNSPort < 1 || c.DNSPort > 65535 {
		return errors.New("DNS port must be in btw 1 and 65535")
	}

	if c.Timeout <= 0 {
		return errors.New("timeout must be positive")
	}

	if c.RetryCount < 0 {
		return errors.New("retry count cannot be negative")
	}

	if c.WorkerCount < 1 {
		return errors.New("Worker count must be at least 1")
	}

	return nil
}