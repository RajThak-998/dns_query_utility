package query

import (
	"dns_query_utility/config"
	"dns_query_utility/result"
	"fmt"
	"net"
	"strings"
	"time"
)

func ExecuteQuery(spec QuerySpec, cfg config.Config) result.QueryResult {
	startTime := time.Now()

	res := result.QueryResult{
		Domain:    spec.Domain,
		Transport: spec.Transport.String(),
		IPVersion: spec.IPVersion.String(),
		Status:    result.StatusError,
	}

	var dnsServer string
	if spec.IPVersion == IPv4 {
		dnsServer = cfg.DNSServerIPv4
		if dnsServer == "" {
			res.Error = "IPv4 DNS server not configured"
			res.Latency = time.Since(startTime)
			return res
		}
	} else {
		dnsServer = cfg.DNSServerIPv6
		if dnsServer == "" {
			res.Error = "IPv6 DNS server not configured"
			res.Latency = time.Since(startTime)
			return res
		}
	}

	packet, err := BuildDNSQuery(spec.Domain, spec.IPVersion)
	if err != nil {
		res.Error = fmt.Sprintf("failed to build DNS query: %v", err)
		res.Latency = time.Since(startTime)
		return res
	}

	serverAddr := formatServerAddress(dnsServer, cfg.DNSPort)

	var response []byte
	if spec.Transport == UDP {
		response, err = executeUDP(packet, serverAddr, cfg)
	} else {
		response, err = executeTCP(packet, serverAddr, cfg)
	}

	res.Latency = time.Since(startTime)

	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			res.Status = result.StatusTimeout
			res.Error = "query timed out"
		} else {
			res.Status = result.StatusError
			res.Error = fmt.Sprintf("query failed: %v", err)
		}
		return res
	}

	rcode, answers, err := ParseDNSResponse(response)
	if err != nil {
		res.Status = result.StatusError
		res.Error = fmt.Sprintf("failed to parse response: %v", err)
		return res
	}

	res.ResponseCode = rcode

	// Separate IPs from other records
	ips := make([]string, 0)
	records := make([]string, 0)
	for _, answer := range answers {
		if strings.Contains(answer, ":") && !strings.Contains(answer, "CNAME") &&
			!strings.Contains(answer, "MX") && !strings.Contains(answer, "TXT") &&
			!strings.Contains(answer, "NS") && !strings.Contains(answer, "SOA") &&
			!strings.Contains(answer, "SRV") && !strings.Contains(answer, "CAA") &&
			!strings.Contains(answer, "PTR") {
			// IPv6 address (contains colons but not a record label)
			ips = append(ips, answer)
		} else if !strings.ContainsAny(answer, "ABCDEFGHIJKLMNOPQRSTUVWXYZ:") ||
			// Pure IPv4 (only dots and digits)
			isIPv4(answer) {
			ips = append(ips, answer)
		} else {
			records = append(records, answer)
		}
	}

	res.ResolvedIPs = ips
	res.Records = records
	res.Status = result.StatusSuccess

	return res
}

// isIPv4 checks if a string looks like an IPv4 address
func isIPv4(s string) bool {
	for _, c := range s {
		if c != '.' && (c < '0' || c > '9') {
			return false
		}
	}
	return strings.Contains(s, ".")
}

func formatServerAddress(server string, port int) string {
	if strings.Contains(server, ":") {
		return fmt.Sprintf("[%s]:%d", server, port)
	}
	return fmt.Sprintf("%s:%d", server, port)
}

func executeUDP(packet []byte, serverAddr string, cfg config.Config) ([]byte, error) {
	conn, err := net.DialTimeout("udp", serverAddr, cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to dial UDP: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(cfg.Timeout))

	_, err = conn.Write(packet)
	if err != nil {
		return nil, fmt.Errorf("failed to send UDP packet: %w", err)
	}

	responseBuffer := make([]byte, 512)
	n, err := conn.Read(responseBuffer)
	if err != nil {
		return nil, fmt.Errorf("failed to read UDP response: %w", err)
	}

	return responseBuffer[:n], nil
}

func executeTCP(packet []byte, serverAddr string, cfg config.Config) ([]byte, error) {
	conn, err := net.DialTimeout("tcp", serverAddr, cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to dial TCP: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(cfg.Timeout))

	messageLength := uint16(len(packet))
	lengthPrefix := make([]byte, 2)
	lengthPrefix[0] = byte(messageLength >> 8)
	lengthPrefix[1] = byte(messageLength & 0xFF)

	_, err = conn.Write(lengthPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to send TCP length prefix: %w", err)
	}

	_, err = conn.Write(packet)
	if err != nil {
		return nil, fmt.Errorf("failed to send TCP packet: %w", err)
	}

	responseLengthBuf := make([]byte, 2)
	_, err = conn.Read(responseLengthBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to read TCP response length: %w", err)
	}

	responseLength := int(responseLengthBuf[0])<<8 | int(responseLengthBuf[1])

	responseBuffer := make([]byte, responseLength)
	totalRead := 0
	for totalRead < responseLength {
		n, err := conn.Read(responseBuffer[totalRead:])
		if err != nil {
			return nil, fmt.Errorf("failed to read TCP response: %w", err)
		}
		totalRead += n
	}

	return responseBuffer, nil
}
