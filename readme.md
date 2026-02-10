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