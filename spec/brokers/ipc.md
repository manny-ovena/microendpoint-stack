# IPC Broker Specification

## Overview

The IPC Broker provides RPC-based access to filesystem operations.  
Used for audit logging, local persistence, and lightweight file operations.

Responsibilities:

- safe file writes
- safe file reads
- directory management
- metrics
- logging

## RPC Methods

### WriteAuditRecord

Parameters:

- record (object)

Returns:

- success boolean

### ReadAuditRecord

Parameters:

- id (string)

Returns:

- record (object)

---

## Metrics

| Metric Name               | Description                           | Dimensions                                      |
|---------------------------|---------------------------------------|-------------------------------------------------|
| ipc_operation_latency_ms  | Latency of IPC operation               | service_name, ipc_operation, error_type         |
| ipc_bytes_written_total   | Total bytes written                    | service_name                                    |
| ipc_bytes_read_total      | Total bytes read                       | service_name                                    |

---

## Log Fields

| Field                    | Description                               |
|--------------------------|-------------------------------------------|
| ipc_operation            | Operation name                            |
| ipc_latency_ms           | IPC operation latency                     |
| ipc_error                | IPC error or null                         |
| ipc_bytes_read           | Bytes read                                |
| ipc_bytes_written        | Bytes written                             |

---
