package query

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"strings"
)

// DNS Record Types (RFC 1035 + extensions)
const (
	TypeA     uint16 = 1   // IPv4 address
	TypeNS    uint16 = 2   // Name server
	TypeCNAME uint16 = 5   // Canonical name (alias)
	TypeSOA   uint16 = 6   // Start of authority
	TypePTR   uint16 = 12  // Pointer record
	TypeMX    uint16 = 15  // Mail exchange
	TypeTXT   uint16 = 16  // Text record
	TypeAAAA  uint16 = 28  // IPv6 address
	TypeSRV   uint16 = 33  // Service locator
	TypeCAA   uint16 = 257 // Certification authority authorization
)

// recordTypeName returns human-readable name for DNS record type
func recordTypeName(rtype uint16) string {
	switch rtype {
	case TypeA:
		return "A"
	case TypeNS:
		return "NS"
	case TypeCNAME:
		return "CNAME"
	case TypeSOA:
		return "SOA"
	case TypePTR:
		return "PTR"
	case TypeMX:
		return "MX"
	case TypeTXT:
		return "TXT"
	case TypeAAAA:
		return "AAAA"
	case TypeSRV:
		return "SRV"
	case TypeCAA:
		return "CAA"
	default:
		return fmt.Sprintf("TYPE%d", rtype)
	}
}

const (
	ClassIN = 1
)

func BuildDNSQuery(domain string, ipVersion IPVersion) ([]byte, error) {
	if domain == "" {
		return nil, errors.New("domain cannot be empty")
	}

	var queryType uint16
	if ipVersion == IPv4 {
		queryType = TypeA
	} else {
		queryType = TypeAAAA
	}

	txID := uint16(rand.Intn(65536))
	header := buildDNSHeader(txID, 1)

	question, err := buildDNSQuestion(domain, queryType, ClassIN)
	if err != nil {
		return nil, err
	}

	packet := append(header, question...)
	return packet, nil
}

func buildDNSHeader(txID uint16, questionCount uint16) []byte {
	header := make([]byte, 12)

	binary.BigEndian.PutUint16(header[0:2], txID)
	binary.BigEndian.PutUint16(header[2:4], 0x0100)
	binary.BigEndian.PutUint16(header[4:6], questionCount)
	binary.BigEndian.PutUint16(header[6:8], 0)
	binary.BigEndian.PutUint16(header[8:10], 0)
	binary.BigEndian.PutUint16(header[10:12], 0)

	return header
}

func buildDNSQuestion(domain string, queryType uint16, class uint16) ([]byte, error) {
	encodedDomain, err := encodeDomainName(domain)
	if err != nil {
		return nil, err
	}

	question := make([]byte, len(encodedDomain)+4)
	copy(question, encodedDomain)
	binary.BigEndian.PutUint16(question[len(encodedDomain):], queryType)
	binary.BigEndian.PutUint16(question[len(encodedDomain)+2:], class)

	return question, nil
}

func encodeDomainName(domain string) ([]byte, error) {
	domain = strings.TrimSuffix(domain, ".")
	labels := strings.Split(domain, ".")

	for _, label := range labels {
		if len(label) == 0 {
			return nil, errors.New("domain has empty label")
		}
		if len(label) > 63 {
			return nil, errors.New("domain label exceeds 63 characters")
		}
	}

	totalSize := 0
	for _, label := range labels {
		totalSize += 1 + len(label)
	}
	totalSize += 1

	encoded := make([]byte, totalSize)
	offset := 0

	for _, label := range labels {
		encoded[offset] = byte(len(label))
		offset++
		copy(encoded[offset:], []byte(label))
		offset += len(label)
	}

	encoded[offset] = 0
	return encoded, nil
}

// ParseDNSResponse extracts RCODE and all record data from DNS response
// Parses answer, authority, and additional sections
func ParseDNSResponse(response []byte) (rcode int, answers []string, err error) {
	if len(response) < 12 {
		return 0, nil, errors.New("response too short")
	}

	flags := binary.BigEndian.Uint16(response[2:4])
	rcode = int(flags & 0x000F)

	questionCount := binary.BigEndian.Uint16(response[4:6])
	answerCount := binary.BigEndian.Uint16(response[6:8])
	authorityCount := binary.BigEndian.Uint16(response[8:10])
	additionalCount := binary.BigEndian.Uint16(response[10:12])

	// Total records across all sections
	totalRecords := int(answerCount) + int(authorityCount) + int(additionalCount)

	if rcode != 0 {
		return rcode, []string{}, nil
	}

	offset := 12

	// Skip question section
	for i := 0; i < int(questionCount); i++ {
		newOffset, err := skipDomainName(response, offset)
		if err != nil {
			return rcode, nil, err
		}
		offset = newOffset
		offset += 4 // Skip query type + class
	}

	// Parse all sections: answer + authority + additional
	ips := make([]string, 0)
	cnames := make([]string, 0)
	otherRecords := make([]string, 0)

	for i := 0; i < totalRecords; i++ {
		if offset >= len(response) {
			break
		}

		// Read record name
		newOffset, err := skipDomainName(response, offset)
		if err != nil {
			break
		}
		offset = newOffset

		if offset+10 > len(response) {
			break
		}

		// Extract record fields
		recordType := binary.BigEndian.Uint16(response[offset : offset+2])
		offset += 2 // type
		offset += 2 // class
		offset += 4 // TTL

		rdLength := binary.BigEndian.Uint16(response[offset : offset+2])
		offset += 2

		if offset+int(rdLength) > len(response) {
			break
		}

		// Parse record based on type
		record := parseRecord(response, offset, recordType, rdLength)
		if record != "" {
			switch recordType {
			case TypeA, TypeAAAA:
				ips = append(ips, record)
			case TypeCNAME:
				cnames = append(cnames, record)
			default:
				otherRecords = append(otherRecords, record)
			}
		}

		// Move past rdata
		offset += int(rdLength)
	}

	// Build final answer list
	answers = make([]string, 0)

	// Add CNAME chain first
	if len(cnames) > 0 {
		answers = append(answers, cnames...)
	}

	// Add IP addresses
	if len(ips) > 0 {
		answers = append(answers, ips...)
	}

	// Add other records if no IPs found
	if len(ips) == 0 && len(otherRecords) > 0 {
		answers = append(answers, otherRecords...)
	}

	return rcode, answers, nil
}

// parseRecord extracts human-readable data from a DNS record
func parseRecord(data []byte, offset int, recordType uint16, rdLength uint16) string {
	switch recordType {
	case TypeA:
		if rdLength == 4 {
			return fmt.Sprintf("%d.%d.%d.%d",
				data[offset],
				data[offset+1],
				data[offset+2],
				data[offset+3])
		}

	case TypeAAAA:
		if rdLength == 16 {
			return fmt.Sprintf("%x:%x:%x:%x:%x:%x:%x:%x",
				binary.BigEndian.Uint16(data[offset:offset+2]),
				binary.BigEndian.Uint16(data[offset+2:offset+4]),
				binary.BigEndian.Uint16(data[offset+4:offset+6]),
				binary.BigEndian.Uint16(data[offset+6:offset+8]),
				binary.BigEndian.Uint16(data[offset+8:offset+10]),
				binary.BigEndian.Uint16(data[offset+10:offset+12]),
				binary.BigEndian.Uint16(data[offset+12:offset+14]),
				binary.BigEndian.Uint16(data[offset+14:offset+16]))
		}

	case TypeCNAME, TypePTR, TypeNS:
		name, err := readDomainName(data, offset)
		if err == nil {
			return fmt.Sprintf("%s:%s", recordTypeName(recordType), name)
		}

	case TypeMX:
		if rdLength >= 4 {
			priority := binary.BigEndian.Uint16(data[offset : offset+2])
			exchange, err := readDomainName(data, offset+2)
			if err == nil {
				return fmt.Sprintf("MX:%d %s", priority, exchange)
			}
		}

	case TypeTXT:
		// TXT records contain one or more strings
		// Each string is prefixed with its length
		txtOffset := offset
		endOffset := offset + int(rdLength)
		var parts []string
		for txtOffset < endOffset {
			strLen := int(data[txtOffset])
			txtOffset++
			if txtOffset+strLen > endOffset {
				break
			}
			parts = append(parts, string(data[txtOffset:txtOffset+strLen]))
			txtOffset += strLen
		}
		if len(parts) > 0 {
			return fmt.Sprintf("TXT:%s", strings.Join(parts, " "))
		}

	case TypeSOA:
		// SOA: primary NS + admin email + 5 uint32 fields
		mname, err := readDomainName(data, offset)
		if err == nil {
			return fmt.Sprintf("SOA:%s", mname)
		}

	case TypeSRV:
		if rdLength >= 8 {
			priority := binary.BigEndian.Uint16(data[offset : offset+2])
			weight := binary.BigEndian.Uint16(data[offset+2 : offset+4])
			port := binary.BigEndian.Uint16(data[offset+4 : offset+6])
			target, err := readDomainName(data, offset+6)
			if err == nil {
				return fmt.Sprintf("SRV:%d %d %d %s", priority, weight, port, target)
			}
		}

	case TypeCAA:
		if rdLength >= 4 {
			flags := data[offset]
			tagLen := int(data[offset+1])
			if offset+2+tagLen <= offset+int(rdLength) {
				tag := string(data[offset+2 : offset+2+tagLen])
				value := string(data[offset+2+tagLen : offset+int(rdLength)])
				return fmt.Sprintf("CAA:%d %s %s", flags, tag, value)
			}
		}
	}

	return ""
}

// readDomainName reads and reconstructs a domain name from DNS response
// Handles compression pointers (0xC0xx)
func readDomainName(data []byte, offset int) (string, error) {
	var parts []string
	visited := make(map[int]bool)
	maxJumps := 10 // Safety limit to prevent infinite loops

	jumps := 0
	for {
		if offset >= len(data) {
			return "", errors.New("domain name extends beyond packet")
		}

		if visited[offset] {
			return "", errors.New("circular reference in domain name")
		}
		visited[offset] = true

		length := int(data[offset])

		// Compression pointer
		if length >= 192 {
			if offset+1 >= len(data) {
				return "", errors.New("truncated compression pointer")
			}
			pointer := int(data[offset])&0x3F<<8 | int(data[offset+1])

			jumps++
			if jumps > maxJumps {
				return "", errors.New("too many compression pointer jumps")
			}

			offset = pointer
			continue
		}

		// End of domain name
		if length == 0 {
			break
		}

		// Read label
		offset++
		if offset+length > len(data) {
			return "", errors.New("label extends beyond packet")
		}
		parts = append(parts, string(data[offset:offset+length]))
		offset += length
	}

	if len(parts) == 0 {
		return ".", nil // Root domain
	}

	return strings.Join(parts, "."), nil
}

func skipDomainName(data []byte, offset int) (int, error) {
	for {
		if offset >= len(data) {
			return 0, errors.New("domain name extends beyond packet")
		}

		length := int(data[offset])

		if length >= 192 {
			return offset + 2, nil
		}

		if length == 0 {
			return offset + 1, nil
		}

		offset += 1 + length
	}
}
