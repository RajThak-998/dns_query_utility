package result

// ConsolidateResults groups query results by domain
func ConsolidateResults(results []QueryResult) []ConsolidatedResult {
	// Group by domain
	domainMap := make(map[string][]QueryResult)

	for _, res := range results {
		domainMap[res.Domain] = append(domainMap[res.Domain], res)
	}

	// Convert to consolidated format
	consolidated := make([]ConsolidatedResult, 0, len(domainMap))

	for domain, domainResults := range domainMap {
		cr := ConsolidatedResult{
			Domain:     domain,
			QueryTypes: make(map[string]TypeResult),
		}

		var totalLatency float64
		successCount := 0
		noAnswerCount := 0
		failedCount := 0

		// Build type results
		for _, res := range domainResults {
			typeRes := TypeResult{
				Status:          res.Status,
				LatencyMs:       res.LatencyMs,
				ResponseCode:    res.ResponseCode,
				ResolvedIPs:     res.ResolvedIPs,
				Records:         res.Records,
				AuthoritativeNS: res.AuthoritativeNS, // NEW: Include in consolidated output
				Error:           res.Error,
				Transport:       res.Transport,
				IPVersion:       res.IPVersion,
				Timestamp:       res.Timestamp,
			}

			cr.QueryTypes[res.QueryType] = typeRes

			// Update counters
			totalLatency += res.LatencyMs
			switch res.Status {
			case StatusSuccess:
				successCount++
			case StatusNoAnswer:
				noAnswerCount++
			default:
				failedCount++
			}
		}

		// Calculate summary
		totalQueries := len(domainResults)
		avgLatency := float64(0)
		if totalQueries > 0 {
			avgLatency = totalLatency / float64(totalQueries)
		}

		cr.Summary = ConsolidatedSummary{
			TotalQueries:     totalQueries,
			Successful:       successCount,
			NoAnswer:         noAnswerCount,
			Failed:           failedCount,
			AverageLatencyMs: avgLatency,
		}

		consolidated = append(consolidated, cr)
	}

	return consolidated
}
