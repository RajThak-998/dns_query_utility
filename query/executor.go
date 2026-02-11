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
        QueryType: spec.QueryType.String(),
        Transport: spec.Transport.String(),
        IPVersion: spec.IPVersion.String(),
        Status:    result.StatusError,
    }

    // Select appropriate DNS server based on IP version (network family)
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

    // Build DNS query packet using QueryType (independent of transport/IP version)
    packet, err := BuildDNSQuery(spec.Domain, spec.QueryType)
    if err != nil {
        res.Error = fmt.Sprintf("failed to build DNS query: %v", err)
        res.Latency = time.Since(startTime)
        return res
    }

    serverAddr := formatServerAddress(dnsServer, cfg.DNSPort)

    // Execute over chosen transport + network family
    var response []byte
    if spec.Transport == UDP {
        response, err = executeUDP(packet, serverAddr, spec.IPVersion, cfg)
    } else {
        response, err = executeTCP(packet, serverAddr, spec.IPVersion, cfg)
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
        if isIPv4(answer) {
            ips = append(ips, answer)
        } else if isIPv6(answer) {
            ips = append(ips, answer)
        } else {
            records = append(records, answer)
        }
    }

    res.ResolvedIPs = ips
    res.Records = records

    // Determine status based on RCODE and answer content
    res.Status = determineStatus(rcode, ips)

    // Set error messages for non-success status
    switch res.Status {
    case result.StatusNXDomain:
        res.Error = "domain does not exist"
    case result.StatusServFail:
        res.Error = "DNS server failure"
    case result.StatusRefused:
        res.Error = "query refused by server"
    case result.StatusNoAnswer:
        if len(records) > 0 {
            res.Error = fmt.Sprintf("no A/AAAA records found (got: %v)", records)
        } else {
            res.Error = "no records found"
        }
    }

    return res
}

// determineStatus maps RCODE and answer content to semantic status
func determineStatus(rcode int, ips []string) result.Status {
    switch rcode {
    case 0:
        if len(ips) > 0 {
            return result.StatusSuccess
        }
        return result.StatusNoAnswer
    case 3:
        return result.StatusNXDomain
    case 2:
        return result.StatusServFail
    case 5:
        return result.StatusRefused
    default:
        return result.StatusError
    }
}

// isIPv4 checks if a string looks like an IPv4 address (digits and dots only)
func isIPv4(s string) bool {
    if !strings.Contains(s, ".") {
        return false
    }
    for _, c := range s {
        if c != '.' && (c < '0' || c > '9') {
            return false
        }
    }
    return true
}

// isIPv6 checks if a string looks like a raw IPv6 address (hex groups with colons, no label prefix)
func isIPv6(s string) bool {
    if !strings.Contains(s, ":") {
        return false
    }
    // If it contains a record type prefix like "CNAME:", "MX:", etc., it's not an IP
    for _, c := range s {
        if (c >= 'g' && c <= 'z') || (c >= 'G' && c <= 'Z') {
            return false
        }
    }
    return true
}

func formatServerAddress(server string, port int) string {
    if strings.Contains(server, ":") {
        return fmt.Sprintf("[%s]:%d", server, port)
    }
    return fmt.Sprintf("%s:%d", server, port)
}

// networkString returns the Go net library network string for transport + IP version
func networkString(transport Transport, ipVersion IPVersion) string {
    base := "udp"
    if transport == TCP {
        base = "tcp"
    }

    if ipVersion == IPv6 {
        return base + "6"
    }
    return base + "4"
}

func executeUDP(packet []byte, serverAddr string, ipVersion IPVersion, cfg config.Config) ([]byte, error) {
    network := networkString(UDP, ipVersion)

    conn, err := net.DialTimeout(network, serverAddr, cfg.Timeout)
    if err != nil {
        return nil, fmt.Errorf("failed to dial %s: %w", network, err)
    }
    defer conn.Close()

    conn.SetDeadline(time.Now().Add(cfg.Timeout))

    _, err = conn.Write(packet)
    if err != nil {
        return nil, fmt.Errorf("failed to send packet: %w", err)
    }

    responseBuffer := make([]byte, 512)
    n, err := conn.Read(responseBuffer)
    if err != nil {
        return nil, fmt.Errorf("failed to read response: %w", err)
    }

    return responseBuffer[:n], nil
}

func executeTCP(packet []byte, serverAddr string, ipVersion IPVersion, cfg config.Config) ([]byte, error) {
    network := networkString(TCP, ipVersion)

    conn, err := net.DialTimeout(network, serverAddr, cfg.Timeout)
    if err != nil {
        return nil, fmt.Errorf("failed to dial %s: %w", network, err)
    }
    defer conn.Close()

    conn.SetDeadline(time.Now().Add(cfg.Timeout))

    // TCP DNS requires 2-byte length prefix
    messageLength := uint16(len(packet))
    lengthPrefix := make([]byte, 2)
    lengthPrefix[0] = byte(messageLength >> 8)
    lengthPrefix[1] = byte(messageLength & 0xFF)

    _, err = conn.Write(lengthPrefix)
    if err != nil {
        return nil, fmt.Errorf("failed to send length prefix: %w", err)
    }

    _, err = conn.Write(packet)
    if err != nil {
        return nil, fmt.Errorf("failed to send packet: %w", err)
    }

    responseLengthBuf := make([]byte, 2)
    _, err = conn.Read(responseLengthBuf)
    if err != nil {
        return nil, fmt.Errorf("failed to read response length: %w", err)
    }

    responseLength := int(responseLengthBuf[0])<<8 | int(responseLengthBuf[1])

    responseBuffer := make([]byte, responseLength)
    totalRead := 0
    for totalRead < responseLength {
        n, err := conn.Read(responseBuffer[totalRead:])
        if err != nil {
            return nil, fmt.Errorf("failed to read response: %w", err)
        }
        totalRead += n
    }

    return responseBuffer, nil
}