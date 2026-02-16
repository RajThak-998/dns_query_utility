# 🌐 DNS Query Utility

A high-performance, concurrent DNS query utility for bulk domain resolution with comprehensive record type support, flexible DNS server configuration, and structured output formats.

```
╔══════════════════════════════════════════════════════════════╗
║                    DNS QUERY UTILITY                         ║
║          High-Performance Bulk DNS Resolution Tool           ║
╚══════════════════════════════════════════════════════════════╝
```

## ✨ Features

- 🚀 **Concurrent Execution** - Auto-scaling worker pool (1-50 workers) for optimal performance
- 📊 **Multiple DNS Record Types** - Support for A, AAAA, MX, TXT, NS, SOA, CNAME, PTR, SRV
- 🌍 **IPv4 & IPv6** - Full support for both IP versions with independent transport control
- 🔄 **Dual DNS Servers** - Configure primary and secondary DNS servers
- 📝 **Flexible Output** - JSON (default), CSV, or all formats with rich metadata
- ⚡ **Smart Defaults** - Uses Google DNS by default (8.8.8.8:53 and 2001:4860:4860::8888:53)
- 🎯 **Custom Ports** - Support for non-standard DNS ports
- 🔁 **Retry Logic** - Configurable retry attempts with exponential backoff
- ⏱️ **Timeout Control** - Flexible timeout configuration (seconds, milliseconds, minutes)
- 📈 **Detailed Metrics** - Latency tracking, success rates, and comprehensive status codes

## 📋 Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [CSV Input Format](#csv-input-format)
- [Command Reference](#command-reference)
- [DNS Server Configuration](#dns-server-configuration)
- [Output Formats](#output-formats)
- [Usage Examples](#usage-examples)
- [Status Codes](#status-codes)
- [How It Works](#how-it-works)

## 🔧 Installation

### Prerequisites

- Go 1.22 or later

### Build from Source

```bash
# Clone the repository
cd dns_query_utility

# Download dependencies
go mod download

# Build the binary
go build -o dns_query_utility

# Run the utility
./dns_query_utility queries.csv
```

### Cross-Platform Builds

```bash
# Linux (AMD64)
GOOS=linux GOARCH=amd64 go build -o dns_query_utility_linux

# macOS (AMD64)
GOOS=darwin GOARCH=amd64 go build -o dns_query_utility_macos

# macOS (ARM64 - Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o dns_query_utility_macos_arm

# Windows (AMD64)
GOOS=windows GOARCH=amd64 go build -o dns_query_utility.exe
```

## 🚀 Quick Start

1. **Create a CSV file** with your DNS queries:

```csv
domain,query_type,transport,network
google.com,A,udp,ipv4
github.com,AAAA,udp,ipv6
microsoft.com,MX,tcp,ipv4
```

2. **Run the utility** with default settings:

```bash
./dns_query_utility queries.csv
```

3. **Check the results** in `result.json`:

```bash
cat result.json | jq .
```

That's it! 🎉

## 📄 CSV Input Format

The utility reads DNS queries from a CSV file with 4 columns:

| Column | Description | Valid Values | Example |
|--------|-------------|--------------|---------|
| `domain` | Domain name to query | Any valid domain | `google.com` |
| `query_type` | DNS record type | `A`, `AAAA`, `MX`, `TXT`, `NS`, `SOA`, `CNAME`, `PTR`, `SRV` | `A` |
| `transport` | Network transport | `udp`, `tcp` | `udp` |
| `network` | IP version | `ipv4`, `ipv6` | `ipv4` |

### Sample queries.csv

```csv
domain,query_type,transport,network
google.com,A,udp,ipv4
google.com,AAAA,udp,ipv6
cloudflare.com,A,tcp,ipv4
github.com,MX,udp,ipv4
example.com,TXT,udp,ipv4
microsoft.com,NS,udp,ipv4
aws.amazon.com,CNAME,udp,ipv4
```

### Supported DNS Record Types

- **A** - IPv4 addresses
- **AAAA** - IPv6 addresses
- **MX** - Mail exchange servers
- **TXT** - Text records
- **NS** - Name servers
- **SOA** - Start of authority
- **CNAME** - Canonical names
- **PTR** - Pointer records
- **SRV** - Service records

## 📖 Command Reference

### Basic Usage

```bash
./dns_query_utility <csv_file> [options]
```

### Command-Line Options

| Flag | Short | Description | Default | Example |
|------|-------|-------------|---------|---------|
| `--dns` | - | DNS server(s) to use (up to 2). Accepts IPv4/IPv6, optional port. When passing multiple servers, wrap the list in quotes. | `8.8.8.8:53` (IPv4) & `2001:4860:4860::8888:53` (IPv6) | `--dns 1.1.1.1` |
| `-t`, `--timeout` | -t | Query timeout (Go duration format) | `5s` | `--timeout 10s` |
| `-r`, `--retry` | -r | Retry attempts (0-10) | `2` | `--retry 3` |
| `-o`, `--output` | -o | Base name for output file(s). Extension added based on format. | `result` | `--output dns_results` |
| `-f`, `--format` | -f | Output format: `json`, `csv`, `all` | `json` | `--format csv` |
| `-h`, `--help` | -h | Show help message | - | `--help` |

### Timeout Format

- **Seconds**: `5s`, `10s`
- **Milliseconds**: `500ms`, `1500ms`
- **Minutes**: `1m`, `2m`

## 🌐 DNS Server Configuration

### Default DNS Servers

```bash
./dns_query_utility queries.csv
# Uses Google DNS by default:
#  - IPv4:  8.8.8.8:53
#  - IPv6:  2001:4860:4860::8888:53
```

### Single Custom DNS Server

```bash
# IPv4 DNS server
./dns_query_utility queries.csv --dns 1.1.1.1

# IPv4 with custom port
./dns_query_utility queries.csv --dns 1.1.1.1:5353

# IPv6 DNS server
./dns_query_utility queries.csv --dns 2606:4700:4700::1111

# IPv6 with custom port (use brackets)
./dns_query_utility queries.csv --dns [2606:4700:4700::1111]:5353
```

### Dual / Multiple DNS Servers (wrap list in quotes)

When providing more than one DNS server (or mixing IPv4/IPv6), wrap the argument in double quotes:

```bash
# IPv4 + IPv6
./dns_query_utility queries.csv --dns "8.8.8.8 2606:4700:4700::1111"

# Two IPv4 servers
./dns_query_utility queries.csv --dns "8.8.8.8 1.1.1.1"

# With custom ports (IPv6 addresses with ports must use brackets)
./dns_query_utility queries.csv --dns "9.9.9.9:54 [2620:fe::fe]:5353"
```

### Popular DNS Servers

| Provider | IPv4 | IPv6 |
|----------|------|------|
| **Google** | `8.8.8.8`, `8.8.4.4` | `2001:4860:4860::8888`, `2001:4860:4860::8844` |
| **Cloudflare** | `1.1.1.1`, `1.0.0.1` | `2606:4700:4700::1111`, `2606:4700:4700::1001` |
| **Quad9** | `9.9.9.9`, `149.112.112.112` | `2620:fe::fe`, `2620:fe::9` |
| **OpenDNS** | `208.67.222.222`, `208.67.220.220` | `2620:119:35::35`, `2620:119:53::53` |

## 📊 Output Formats

### Supported formats
- `json` — JSON with metadata and results (default)
- `csv`  — Comma-separated values
- `all`  — Generate both JSON and CSV files

Examples:

```bash
# JSON (default)
./dns_query_utility queries.csv        # creates: result.json

# CSV
./dns_query_utility queries.csv --format csv   # creates: result.csv

# Both (all)
./dns_query_utility queries.csv --format all   # creates: result.json + result.csv
```

### Custom Output File Names

- If `--output` is not provided the base name defaults to `result`.
- The utility appends the appropriate extension based on `--format` (e.g., `.json`, `.csv`).

Examples:

```bash
# JSON with custom name
./dns_query_utility queries.csv -o my_dns_results
# creates: my_dns_results.json

# CSV with custom name
./dns_query_utility queries.csv -f csv -o dns_audit
# creates: dns_audit.csv

# Both formats with custom base name
./dns_query_utility queries.csv -f all -o dns_test
# creates: dns_test.json AND dns_test.csv
```

## 💡 Usage Examples

### Example 1: Quick DNS Check with Defaults

```bash
./dns_query_utility queries.csv
```
- Uses Google DNS (8.8.8.8 / 2001:4860:4860::8888)
- 5-second timeout (default)
- 2 retry attempts (default)
- JSON output to result.json

### Example 2: Use Cloudflare DNS with Longer Timeout

```bash
./dns_query_utility queries.csv --dns 1.1.1.1 --timeout 5s
```

### Example 3: Dual DNS Servers with Custom Ports

```bash
./dns_query_utility queries.csv --dns 8.8.8.8:53 1.1.1.1:5353
```

### Example 4: High-Reliability Configuration

```bash
./dns_query_utility queries.csv --dns 1.1.1.1 --timeout 5s --retry 5
```
- Cloudflare DNS
- 5-second timeout
- 5 retry attempts
- Good for unreliable networks

### Example 5: Fast Queries with No Retries

```bash
./dns_query_utility queries.csv --timeout 500ms --retry 0
```
- 500ms timeout
- No retries
- Good for low-latency networks

### Example 6: IPv6 DNS Server

```bash
./dns_query_utility queries.csv --dns 2606:4700:4700::1111
```

### Example 7: Private DNS Server

```bash
./dns_query_utility queries.csv --dns 192.168.1.1:5353
```

### Example 8: CSV Output for Spreadsheet Analysis

```bash
./dns_query_utility queries.csv --format csv --output dns_audit.csv
```

### Example 9: Both Formats for Comprehensive Analysis

```bash
./dns_query_utility queries.csv --format all --output dns_test
```
- Creates dns_test.json (for automated processing)
- Creates dns_test.csv (for manual review)

### Example 10: Production Monitoring Setup

```bash
./dns_query_utility production_domains.csv \
  --dns 10.0.0.1 8.8.8.8 \
  --timeout 3s \
  --retry 3 \
  --format both \
  --output prod_dns_check
```
- Primary DNS: Internal (10.0.0.1)
- Fallback DNS: Google (8.8.8.8)
- 3-second timeout
- 3 retries
- Both JSON and CSV output

## 📊 Status Codes

The utility provides semantic status codes for each query:

| Status | Description | Meaning |
|--------|-------------|---------|
| `success` | Query successful with answers | Domain resolved successfully |
| `no_answer` | Query successful but no records | Domain exists but no records found |
| `nxdomain` | Non-existent domain | Domain does not exist |
| `servfail` | Server failure | DNS server encountered an error |
| `refused` | Query refused | DNS server refused the query |
| `timeout` | Query timeout | No response within timeout period |
| `error` | Execution error | Network or connection error |

### Status Code Distribution (Example)

```json
{
  "metadata": {
    "status_distribution": {
      "success": 95,
      "no_answer": 2,
      "nxdomain": 1,
      "timeout": 1,
      "error": 1
    }
  }
}
```

## 🛠️ How It Works

### Architecture Overview

```
┌─────────────┐
│  CSV Input  │
│ queries.csv │
└──────┬──────┘
       │
       ▼
┌─────────────────┐
│  Query Parser   │
│ Validates & Loads│
└──────┬──────────┘
       │
       ▼
┌─────────────────────────┐
│   Worker Pool Manager   │
│  Auto-scales: 1-50      │
│  Based on query count   │
└──────┬──────────────────┘
       │
       ▼
┌────────────────────────────┐
│  Concurrent DNS Queries    │
│  • UDP/TCP Transport       │
│  • IPv4/IPv6 Support       │
│  • Timeout & Retry Logic   │
│  • Latency Measurement     │
└──────┬─────────────────────┘
       │
       ▼
┌─────────────────┐
│  Result Collect │
│  & Aggregation  │
└──────┬──────────┘
       │
       ▼
┌─────────────────┐
│  Output Writer  │
│  JSON/CSV/Both  │
└─────────────────┘
```

### Worker Pool Auto-Scaling

The utility automatically determines optimal worker count:

```
Workers = min(max(query_count / 5, 1), 50)
```

**Examples:**
- 10 queries → 2 workers
- 100 queries → 20 workers
- 500+ queries → 50 workers (max)

### Query Execution Flow

1. **Parse CSV** - Validate domains and parameters
2. **Initialize Workers** - Spawn optimal number of goroutines
3. **Distribute Queries** - Feed queries to worker pool via buffered channel
4. **Execute Queries** - Each worker:
   - Constructs DNS packet
   - Sends query via UDP/TCP
   - Parses response
   - Measures latency
   - Applies retry logic on failures
5. **Collect Results** - Aggregate all results with statistics
6. **Write Output** - Generate JSON/CSV with metadata

### Concurrency Model

- **Worker Pool Pattern** - Fixed number of worker goroutines
- **Buffered Channels** - Size = workers × 2 for optimal throughput
- **Non-blocking** - Slow queries don't block fast ones
- **Progress Tracking** - Real-time progress bar during execution

### DNS Query Process

Each query follows this sequence:

1. **Domain Encoding** - Convert domain to DNS wire format
2. **Packet Construction** - Build DNS query packet with appropriate record type
3. **Transport Selection** - Choose UDP or TCP based on specification
4. **IP Version Selection** - Use IPv4 or IPv6 socket
5. **Query Execution** - Send packet to DNS server
6. **Response Parsing** - Extract IPs and other records
7. **Status Determination** - Classify result (success/failure/error)
8. **Latency Measurement** - Record query execution time

## 🔍 Troubleshooting

### Common Issues

**Issue: Permission Denied**
```bash
chmod +x dns_query_utility
./dns_query_utility queries.csv
```

**Issue: Timeout Errors**
```bash
# Increase timeout
./dns_query_utility queries.csv --timeout 10s --retry 5
```

**Issue: IPv6 Connection Failures**
```bash
# Use IPv4 DNS server
./dns_query_utility queries.csv --dns 8.8.8.8
```

**Issue: High Memory Usage**
```bash
# Worker pool auto-scales; for very large inputs (10k+ queries),
# the utility may use more memory. This is expected behavior.
```

## 📝 Advanced Tips

### 1. Monitoring DNS Server Performance

Create queries for the same domain with different DNS servers:

```csv
domain,query_type,transport,network
google.com,A,udp,ipv4
```

Run multiple times with different servers and compare latencies:

```bash
./dns_query_utility test.csv --dns 8.8.8.8 -o google_dns.json
./dns_query_utility test.csv --dns 1.1.1.1 -o cloudflare_dns.json
./dns_query_utility test.csv --dns 9.9.9.9 -o quad9_dns.json
```

### 2. Testing DNS Failover

```bash
./dns_query_utility queries.csv --dns 192.168.1.1 8.8.8.8 --timeout 2s
```
If primary (192.168.1.1) fails, queries automatically use secondary (8.8.8.8).

### 3. Comparing UDP vs TCP Performance

Create two CSV files with identical domains but different transports:

```csv
# udp_queries.csv
domain,query_type,transport,network
example.com,A,udp,ipv4

# tcp_queries.csv
domain,query_type,transport,network
example.com,A,tcp,ipv4
```

Run both and compare latencies.

### 4. Automated Scheduling

```bash
# Run every hour (cron example)
0 * * * * /path/to/dns_query_utility /path/to/queries.csv -o /path/to/results/$(date +\%Y\%m\%d_\%H\%M).json
```

## 🎯 Performance Metrics

**Benchmark Results** (tested on Ubuntu 22.04, Intel i7, 16GB RAM):

| Query Count | Workers | Execution Time | Avg Latency |
|-------------|---------|----------------|-------------|
| 10 | 2 | 0.5s | 45ms |
| 100 | 20 | 2.3s | 42ms |
| 500 | 50 | 8.1s | 48ms |
| 1000 | 50 | 15.7s | 46ms |

## 📜 License

This project is provided as-is for DNS querying and analysis purposes.

## 🤝 Contributing

Contributions are welcome! Feel free to submit issues or pull requests.

## 📧 Support

For questions or issues, please open a GitHub issue.

---

**Built with ❤️ using Go | Powered by [miekg/dns](https://github.com/miekg/dns)**
