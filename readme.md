# DNS Query Utility

## Overview

A DNS query execution and analysis utility designed to run predefined DNS queries against a specific DNS server and capture detailed, structured results for each query. The intent is to make DNS behavior explicit, repeatable, and observable under different query conditions.

## Input Model

### CSV-Based Query Definition

The utility takes a CSV file as its primary input, where each row represents one DNS query scenario.

**CSV Fields:**
- Domain name to be resolved
- IP version (`ipv4` or `ipv6`)
- Transport protocol (`udp` or `tcp`)

This structure allows the CSV to act as a test matrix, where multiple DNS conditions can be expressed declaratively and executed automatically.

### Global Configuration

Global parameters are provided as configuration to the tool and apply uniformly to all queries:
- DNS server address
- DNS server port
- Timeout and retry values

## Core Workflow

### 1. Parsing and Validation

When the utility starts, it:
1. Reads the CSV file
2. Parses each row into an internal query specification
3. Normalizes and validates all fields

**Validation ensures:**
- Domains are well-formed
- IP version values are valid
- Transport values are valid
- Only validated queries are scheduled for execution

### 2. Query Scheduling and Concurrency

The utility schedules queries using a concurrent worker model, enabling:
- Parallel execution of multiple DNS queries
- Improved overall execution time
- Prevention of slow or unresponsive queries from blocking others
- Efficient processing of large CSV inputs

Each query is treated as an independent unit of work.

### 3. DNS Request Construction

For every query:
- A DNS request packet is constructed programmatically
- Query type is selected based on IP version:
  - **A records** for IPv4
  - **AAAA records** for IPv6
- Standard DNS header fields and flags are set

### 4. Transport and IP Selection

Based on the query specification, the utility:
- Chooses either UDP or TCP as the transport
- Selects the appropriate IP family (IPv4 or IPv6)

**Supported combinations:**
- UDP over IPv4
- UDP over IPv6
- TCP over IPv4
- TCP over IPv6

### 5. Query Execution

The utility sends the constructed DNS request to the configured DNS server and port.

**During execution:**
- Timeouts are enforced to prevent indefinite blocking
- Retries are applied where appropriate
- Execution time is measured to capture latency
- For TCP queries, connection establishment and message framing are handled as part of the request lifecycle

### 6. Response Handling

When a response is received, the utility:
- Parses the DNS response packet
- Extracts the response code (e.g., success or specific DNS errors)
- Collects all answer records returned by the server
- Associates the response with the original query parameters

## Output

### Result Structure

Each query produces a structured result containing:
- Domain name
- Transport protocol used
- IP version used
- DNS response code
- Resolved IP addresses (if any)
- Query latency
- Execution status

### Logging and Analysis

Results are written in a structured format suitable for:
- Human inspection
- Automated analysis
- Aggregation and reporting

**This enables:**
- Comparison of DNS behavior across different transports
- Analysis of IPv4 vs IPv6 responses
- Detection of latency issues or failures

## Platform Support

The utility runs entirely in user space and relies only on standard networking primitives, ensuring consistent behavior across:
- Linux
- macOS
- Windows

The tool can be distributed as a single compiled binary with no external runtime dependencies.

## Summary

The project provides a way to define DNS query scenarios declaratively, execute them in a controlled and repeatable manner, and produce detailed results that describe how a DNS server responds under different conditions.


# Implementation Blueprint

The implementation is divided into phases to keep the codebase clean, testable, and scalable.

---

## Phase 1: Foundation & Data Models

**Goal:** Define core data structures before writing any logic.

### `config/config.go`

* Define `Config` struct with:

  * DNS server address
  * DNS server port
  * Timeout
  * Retry count
  * Worker count
* Add validation methods to ensure configuration values are sensible.

### `query/spec.go`

* Define `QuerySpec` struct:

  * Domain name
  * IP version enum (`IPv4`, `IPv6`)
  * Transport enum (`UDP`, `TCP`)
* Define enums for IP version and transport.
* Add validation methods for query specifications.

### `result/result.go`

* Define `QueryResult` struct containing:

  * Domain
  * Transport
  * IP version
  * DNS response code
  * Resolved IP addresses
  * Query latency
  * Status / error information
* Add JSON marshaling tags for structured output.

**Why start here?**
Clear data contracts between components ensure that each layer knows exactly what it receives and what it produces.

---

## Phase 2: CSV Parsing

**Goal:** Read and validate CSV input into query specifications.

### `parser/csv.go`

* Function: `ParseCSV(filepath string) ([]query.QuerySpec, error)`
* Read CSV file from disk.
* For each row:

  * Parse domain name
  * Parse IP version
  * Parse transport protocol
* Validate each field (domain format, enum values).
* Skip invalid rows while logging warnings.
* Return a slice of valid `QuerySpec` objects.

**Why now?**
Input parsing is independent of DNS logic and can be tested immediately using sample CSV files.

---

## Phase 3: DNS Packet Construction

**Goal:** Build raw DNS query packets programmatically.

### `query/builder.go`

* Function: `BuildDNSQuery(domain string, queryType uint16) ([]byte, error)`
* Construct DNS header:

  * Transaction ID
  * Flags
  * Question count
* Encode domain name using DNS label format (length-prefixed labels).
* Add query type:

  * `A` for IPv4
  * `AAAA` for IPv6
* Add query class (`IN`).
* Return raw DNS packet bytes ready to be sent over the network.

**Why now?**
This phase is pure computation with no network I/O and can be unit tested easily. DNS packet formats are standardized (RFC 1035).

---

## Phase 4: Query Execution

**Goal:** Send DNS packets and handle responses with explicit transport and IP control.

### `query/executor.go`

* Function: `ExecuteQuery(spec QuerySpec, config Config) QueryResult`
* Select socket type based on transport and IP version:

  * UDP over IPv4: `net.DialUDP("udp4", ...)`
  * UDP over IPv6: `net.DialUDP("udp6", ...)`
  * TCP over IPv4: `net.DialTCP("tcp4", ...)`
  * TCP over IPv6: `net.DialTCP("tcp6", ...)`
* Apply socket timeouts.
* Build DNS packet using `builder.BuildDNSQuery`.
* Send DNS request.
* Receive DNS response:

  * Handle TCP length prefix if applicable.
* Parse response:

  * Extract response code (RCODE)
  * Extract answer records
* Measure latency between send and receive.
* Return populated `QueryResult`.

**Why now?**
With data models and packet construction in place, this phase focuses solely on network I/O and response parsing.

---

## Phase 5: Worker Pool

**Goal:** Execute DNS queries concurrently while controlling resource usage.

### `worker/pool.go`

* Function: `ProcessQueries(specs []QuerySpec, config Config, workerCount int) []QueryResult`
* Create buffered channels for:

  * Input query specifications
  * Output query results
* Spawn a fixed number of worker goroutines.
* Each worker:

  * Reads a `QuerySpec` from the input channel
  * Calls `executor.ExecuteQuery`
  * Sends the resulting `QueryResult` to the output channel
* Main goroutine:

  * Feeds all query specs into the input channel
  * Collects all results from the output channel
* Return the full list of results.

**Why now?**
Concurrency is easier to add once serial execution works. This phase parallelizes execution without changing DNS logic.

---

## Phase 6: Main Entry Point

**Goal:** Wire all components together with a CLI interface.

### `main.go`

* Parse command-line flags:

  * CSV file path
  * DNS server address
  * DNS server port
  * Timeout
  * Retry count
  * Worker count
* Load and validate configuration.
* Call `parser.ParseCSV` to obtain query specifications.
* Call `worker.ProcessQueries` to execute all queries.
* Output each `QueryResult` as JSON to stdout or write to a file.
* Handle errors gracefully and return a non-zero exit code on failure.

---

## Summary

This phased approach ensures the utility is built incrementally, with clear separation of concerns. Each phase can be developed, tested, and validated independently while contributing directly to the final working system.
