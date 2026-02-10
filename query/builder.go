package query

import (
	"encoding/binary"
	"errors"
	"math/rand"
	"strings"
)

const (
	TypeA = 1
	TypeAAAA = 28
)

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
	if err!=nil {
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
	totalSize +=1
	encoded := make([]byte, totalSize)
	offset :=0

	for _, label := range labels {
		encoded[offset] = byte(len(label))
		offset++

		copy(encoded[offset:], []byte(label))
		offset += len(label)
	}

	encoded[offset] = 0
	return encoded, nil
}

func ParseDNSResponse(response []byte) (rcode int, answers []string, err error) {
	if len(response) < 12 {
		return 0, nil, errors.New("response too short")
	}

	flags := binary.BigEndian.Uint16(response[2:4])
	rcode = int(flags & 0x000F)

	answerCount := binary.BigEndian.Uint16(response[6:8])

	answers = make([]string, 0)

	if rcode ==0 && answerCount > 0 {
		answers = append(answers, "parsing_not_implemented")
	}

	return rcode, answers, nil
}
