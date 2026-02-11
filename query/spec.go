package query

import (
    "errors"
    "fmt"
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
type QueryType uint16

// Supported DNS query types
const (
    QTypeA     QueryType = 1
    QTypeNS    QueryType = 2
    QTypeCNAME QueryType = 5
    QTypeSOA   QueryType = 6
    QTypePTR   QueryType = 12
    QTypeMX    QueryType = 15
    QTypeTXT   QueryType = 16
    QTypeAAAA  QueryType = 28
    QTypeSRV   QueryType = 33
    QTypeCAA   QueryType = 257
)

func (qt QueryType) String() string {
    switch qt {
    case QTypeA:
        return "A"
    case QTypeNS:
        return "NS"
    case QTypeCNAME:
        return "CNAME"
    case QTypeSOA:
        return "SOA"
    case QTypePTR:
        return "PTR"
    case QTypeMX:
        return "MX"
    case QTypeTXT:
        return "TXT"
    case QTypeAAAA:
        return "AAAA"
    case QTypeSRV:
        return "SRV"
    case QTypeCAA:
        return "CAA"
    default:
        return fmt.Sprintf("TYPE%d", qt)
    }
}

// WireValue returns the uint16 value used in DNS packets
func (qt QueryType) WireValue() uint16 {
    return uint16(qt)
}

func ParseQueryType(s string) (QueryType, error) {
    switch strings.ToUpper(s) {
    case "A":
        return QTypeA, nil
    case "NS":
        return QTypeNS, nil
    case "CNAME":
        return QTypeCNAME, nil
    case "SOA":
        return QTypeSOA, nil
    case "PTR":
        return QTypePTR, nil
    case "MX":
        return QTypeMX, nil
    case "TXT":
        return QTypeTXT, nil
    case "AAAA":
        return QTypeAAAA, nil
    case "SRV":
        return QTypeSRV, nil
    case "CAA":
        return QTypeCAA, nil
    default:
        return 0, fmt.Errorf("invalid query type '%s': must be one of A, AAAA, NS, CNAME, SOA, PTR, MX, TXT, SRV, CAA", s)
    }
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