package query

import (
	"errors"
	"strings"
)

// IPVersion represents the network family (socket layer)
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
		return 0, errors.New("invalid ip version: must be 'ipv4' or 'ipv6'")
	}
}

// Transport represents UDP or TCP protocol
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

// QueryType represents DNS query type (QTYPE) as a uint16 wire value
type QueryType int

const (
	QueryTypeA     QueryType = 1
	QueryTypeAAAA  QueryType = 28
	QueryTypeMX    QueryType = 15
	QueryTypeTXT   QueryType = 16
	QueryTypeNS    QueryType = 2
	QueryTypeSOA   QueryType = 6
	QueryTypeCNAME QueryType = 5
	QueryTypePTR   QueryType = 12
	QueryTypeSRV   QueryType = 33
	QueryTypeANY QueryType = 255
)

// String returns canonical name for QueryType
func (qt QueryType) String() string {
	switch qt {
	case QueryTypeA:
		return "A"
	case QueryTypeAAAA:
		return "AAAA"
	case QueryTypeNS:
		return "NS"
	case QueryTypeCNAME:
		return "CNAME"
	case QueryTypeSOA:
		return "SOA"
	case QueryTypePTR:
		return "PTR"
	case QueryTypeMX:
		return "MX"
	case QueryTypeTXT:
		return "TXT"
	case QueryTypeSRV:
		return "SRV"
	case QueryTypeANY:
		return "ANY"
	default:
		return "UNKNOWN"
	}
}

// ParseQueryType converts string to QueryType (case-insensitive)
func ParseQueryType(s string) QueryType {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "A":
		return QueryTypeA
	case "AAAA":
		return QueryTypeAAAA
	case "MX":
		return QueryTypeMX
	case "TXT":
		return QueryTypeTXT
	case "NS":
		return QueryTypeNS
	case "SOA":
		return QueryTypeSOA
	case "CNAME":
		return QueryTypeCNAME
	case "PTR":
		return QueryTypePTR
	case "SRV":
		return QueryTypeSRV
	case "ANY":
		return QueryTypeANY
	default:
		return QueryTypeA // fallback or handle error upstream
	}
}

// WireValue returns the uint16 value used in DNS packets
func (qt QueryType) WireValue() uint16 {
	return uint16(qt)
}

// QuerySpec defines a single DNS query with three independent dimensions
type QuerySpec struct {
	Domain    string    // Domain name to resolve (e.g., "google.com")
	QueryType QueryType // DNS record type: A, AAAA, MX, TXT, etc.
	Transport Transport // Protocol: UDP or TCP
	IPVersion IPVersion // Network family: IPv4 or IPv6 (socket layer)
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
