package query

import (
	"errors"
	"strings"
)

type IPVersion int

const (
	IPv4 IPVersion = iota
	IPv6
)

func (ip IPVersion) String() string {
	switch ip {
	case IPv4:
		return "ipv4"
	case IPv6:
		return "ipv6"
	default:
		return "unknown"
	}
}

func ParseIPVersion(s string) (IPVersion, error) {
	switch strings.ToLower(s) {
	case "ipv4":
		return IPv4, nil
	case "ipv6":
		return IPv6, nil
	default:
		return 0, errors.New("invalid IP version: must be 'ipv4' or 'ipv6'")
	}
}

type Transport int

const (
	UDP Transport = iota
	TCP
)

func (t Transport) String() string {
	switch t {
	case UDP:
		return "udp"
	case TCP:
		return "tcp"
	default:
		return "unknown"
	}
}

func ParseTransport(s string) (Transport, error) {
	switch strings.ToLower(s) {
	case "udp":
		return UDP, nil
	case "tcp":
		return TCP, nil
	default:
		return 0, errors.New("invalid transport: must be 'udp' or 'tcp'")
	}
}

type QuerySpec struct {
	Domain    string
	IPVersion IPVersion
	Transport Transport
}

func (q *QuerySpec) Validate() error {
	if q.Domain == "" {
		return errors.New("domain cannot be empty")
	}

	if !strings.Contains(q.Domain, ".") {
		return errors.New("domain must contain at least one dot")
	}

	if strings.Contains(q.Domain, " ") {
		return errors.New("domain cannot contain spaces")
	}

	return nil
}
