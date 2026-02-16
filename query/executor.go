package query

import (
	"dns_query_utility/config"
	"dns_query_utility/result"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

func ExecuteQuery(spec QuerySpec, cfg config.Config) result.QueryResult {
	startTime := time.Now()

	res := result.QueryResult{
		Domain:          spec.Domain,
		QueryType:       spec.QueryType.String(),
		Transport:       spec.Transport.String(),
		IPVersion:       spec.IPVersion.String(),
		Status:          result.StatusError,
		Timestamp:       startTime,
		AuthoritativeNS: []string{}, // Initialize as empty array, never nil
	}

	// Determine DNS server and network
	var server string
	var network string

	if spec.IPVersion.String() == "ipv4" {
		server = net.JoinHostPort(cfg.DNSServerIPv4, fmt.Sprintf("%d", cfg.DNSPort))
		if spec.Transport.String() == "udp" {
			network = "udp"
		} else {
			network = "tcp"
		}
	} else {
		server = net.JoinHostPort(cfg.DNSServerIPv6, fmt.Sprintf("%d", cfg.DNSPort))
		if spec.Transport.String() == "udp" {
			network = "udp6"
		} else {
			network = "tcp6"
		}
	}

	// Create DNS message
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(spec.Domain), uint16(spec.QueryType))
	msg.RecursionDesired = true

	// Create DNS client
	client := &dns.Client{
		Net:     network,
		Timeout: cfg.Timeout,
	}

	// Execute query with retries
	var response *dns.Msg
	var err error

	for attempt := 0; attempt <= cfg.RetryCount; attempt++ {
		response, _, err = client.Exchange(msg, server)
		if err == nil {
			break
		}
		if attempt == cfg.RetryCount {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Convert nanoseconds to milliseconds (float64)
	res.LatencyMs = float64(time.Since(startTime).Nanoseconds()) / 1e6

	if err != nil {
		res.Error = err.Error()
		if strings.Contains(err.Error(), "timeout") {
			res.Status = result.StatusTimeout
		} else {
			res.Status = result.StatusError
		}
		return res
	}

	res.ResponseCode = response.Rcode

	// Extract authoritative nameservers from Authority AND Additional sections
	res.AuthoritativeNS = extractAuthoritativeNS(response.Ns, response.Extra)

	switch response.Rcode {
	case dns.RcodeSuccess:
		if len(response.Answer) == 0 {
			res.Status = result.StatusNoAnswer
			allRecords := append(response.Ns, response.Extra...)
			if len(allRecords) > 0 {
				records := extractRecords(allRecords)
				res.Records = records
				res.Error = fmt.Sprintf("no A/AAAA records found (got: %v)", records)
			} else {
				res.Error = "no records found"
			}
		} else {
			// parseAnswers handles all record types (A, AAAA, MX, TXT, etc.)
			// This works for both specific queries (A, MX) and ANY queries
			ips, records := parseAnswers(response.Answer)
			res.ResolvedIPs = ips
			res.Records = records

			if len(ips) > 0 || len(records) > 0 {
				res.Status = result.StatusSuccess
			} else {
				res.Status = result.StatusNoAnswer
				res.Error = "response contained no useful records"
			}
		}

	case dns.RcodeNameError:
		res.Status = result.StatusNXDomain
		res.Error = "domain does not exist"

	case dns.RcodeServerFailure:
		res.Status = result.StatusServFail
		res.Error = "server failure"

	case dns.RcodeRefused:
		res.Status = result.StatusRefused
		res.Error = "query refused"

	default:
		res.Status = result.StatusError
		res.Error = fmt.Sprintf("unexpected response code: %d", response.Rcode)
	}

	// If no authoritative NS found in response, do a separate NS lookup
	if len(res.AuthoritativeNS) == 0 {
		res.AuthoritativeNS = lookupAuthoritativeNS(spec.Domain, cfg)
	}

	// Guarantee it's never nil (always return at least empty array)
	if res.AuthoritativeNS == nil {
		res.AuthoritativeNS = []string{}
	}

	return res
}

// getBaseDomain extracts the registrable domain for NS lookup
func getBaseDomain(domain string) string {
	domain = strings.TrimSuffix(domain, ".")
	parts := strings.Split(domain, ".")

	// Handle special cases
	if len(parts) <= 2 {
		return domain
	}

	// For subdomains, return the base domain (last 2 parts)
	// e.g., "www.example.com" -> "example.com"
	return strings.Join(parts[len(parts)-2:], ".")
}

// lookupAuthoritativeNS performs a separate NS query to find authoritative nameservers
func lookupAuthoritativeNS(domain string, cfg config.Config) []string {
	baseDomain := getBaseDomain(domain)

	// Create NS query
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(baseDomain), dns.TypeNS)
	msg.RecursionDesired = true

	// Use IPv4 UDP for NS lookup (most reliable)
	server := net.JoinHostPort(cfg.DNSServerIPv4, fmt.Sprintf("%d", cfg.DNSPort))
	client := &dns.Client{
		Net:     "udp",
		Timeout: cfg.Timeout,
	}

	// Execute NS query
	resp, _, err := client.Exchange(msg, server)
	if err != nil || resp == nil {
		return []string{}
	}

	// Extract NS records from answer section
	var nsRecords []string
	for _, ans := range resp.Answer {
		if ns, ok := ans.(*dns.NS); ok {
			nsRecords = append(nsRecords, ns.Ns)
		}
	}

	// If no answer, check authority section
	if len(nsRecords) == 0 {
		for _, auth := range resp.Ns {
			if ns, ok := auth.(*dns.NS); ok {
				nsRecords = append(nsRecords, ns.Ns)
			}
		}
	}

	// If still no NS records, check additional section
	if len(nsRecords) == 0 {
		for _, extra := range resp.Extra {
			if ns, ok := extra.(*dns.NS); ok {
				nsRecords = append(nsRecords, ns.Ns)
			}
		}
	}

	return nsRecords
}

// extractAuthoritativeNS extracts NS records from Authority and Additional sections
func extractAuthoritativeNS(authority []dns.RR, additional []dns.RR) []string {
	nsMap := make(map[string]bool) // Use map to avoid duplicates

	// Extract from Authority section (SOA, NS)
	for _, rr := range authority {
		switch record := rr.(type) {
		case *dns.NS:
			nsMap[record.Ns] = true
		case *dns.SOA:
			// SOA record indicates the primary nameserver
			nsMap[fmt.Sprintf("SOA:%s", record.Ns)] = true
		}
	}

	// Extract from Additional section (NS records often appear here)
	for _, rr := range additional {
		if ns, ok := rr.(*dns.NS); ok {
			nsMap[ns.Ns] = true
		}
	}

	// Convert map to slice
	var nsRecords []string
	for ns := range nsMap {
		nsRecords = append(nsRecords, ns)
	}

	return nsRecords
}

func parseAnswers(answers []dns.RR) ([]string, []string) {
	var ips []string
	var records []string

	for _, answer := range answers {
		switch rr := answer.(type) {
		case *dns.A:
			ips = append(ips, rr.A.String())
		case *dns.AAAA:
			ips = append(ips, rr.AAAA.String())
		case *dns.CNAME:
			records = append(records, fmt.Sprintf("CNAME:%s", rr.Target))
		case *dns.MX:
			records = append(records, fmt.Sprintf("MX:%d %s", rr.Preference, rr.Mx))
		case *dns.NS:
			records = append(records, fmt.Sprintf("NS:%s", rr.Ns))
		case *dns.TXT:
			records = append(records, fmt.Sprintf("TXT:%s", strings.Join(rr.Txt, " ")))
		case *dns.SOA:
			records = append(records, fmt.Sprintf("SOA:%s %s", rr.Ns, rr.Mbox))
		case *dns.PTR:
			records = append(records, fmt.Sprintf("PTR:%s", rr.Ptr))
		case *dns.SRV:
			records = append(records, fmt.Sprintf("SRV:%d %d %d %s",
				rr.Priority, rr.Weight, rr.Port, rr.Target))
		default:
			records = append(records, fmt.Sprintf("%s:%s", dns.TypeToString[rr.Header().Rrtype], rr.String()))
		}
	}

	return ips, records
}

func extractRecords(rrs []dns.RR) []string {
	var records []string
	for _, rr := range rrs {
		switch r := rr.(type) {
		case *dns.MX:
			records = append(records, fmt.Sprintf("MX:%d %s", r.Preference, r.Mx))
		case *dns.NS:
			records = append(records, fmt.Sprintf("NS:%s", r.Ns))
		case *dns.TXT:
			records = append(records, fmt.Sprintf("TXT:%s", strings.Join(r.Txt, " ")))
		case *dns.SOA:
			records = append(records, fmt.Sprintf("SOA:%s", r.Ns))
		default:
			records = append(records, dns.TypeToString[rr.Header().Rrtype])
		}
	}
	return records
}
