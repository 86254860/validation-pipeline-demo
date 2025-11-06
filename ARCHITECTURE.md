# GCP Validation Pipeline - Architecture

This document provides detailed technical architecture information.

## Pipeline Execution Flow

```
┌──────────────────────────────────────────────────────────────────┐
│                     GCP Validation Pipeline                      │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ Task 1: validate-apis (Go-based)                           │ │
│  │ ┌──────────────────────────────────────────────────────┐   │ │
│  │ │ Container: gcp-api-validator:latest                  │   │ │
│  │ │ Implementation: Go                                    │   │ │
│  │ │ Purpose: Check if required APIs are enabled          │   │ │
│  │ │                                                        │   │ │
│  │ │ Checks:                                               │   │ │
│  │ │  ✅ compute.googleapis.com   → ENABLED               │   │ │
│  │ │  ✅ container.googleapis.com → ENABLED               │   │ │
│  │ │  ❌ iam.googleapis.com       → NOT ENABLED           │   │ │
│  │ │                                                        │   │ │
│  │ │ Results:                                              │   │ │
│  │ │  status:  "FAILED"                                    │   │ │
│  │ │  message: "API validation failed: 1 API(s) not       │   │ │
│  │ │            enabled - iam.googleapis.com"              │   │ │
│  │ │  details: {"project": "demo-project",                │   │ │
│  │ │            "enabled_apis": "compute...,container...", │   │ │
│  │ │            "missing_apis": "iam.googleapis.com",      │   │ │
│  │ │            "missing_count": "1"}                      │   │ │
│  │ └──────────────────────────────────────────────────────┘   │ │
│  └────────────────────────────────────────────────────────────┘ │
│                              │                                  │
│                    ┌─────────┴──────────┐                      │
│                    │ runAfter           │ runAfter             │
│                    ↓                    ↓                      │
│  ┌──────────────────────────┐  ┌────────────────────────────┐ │
│  │ Task 2: validate-quotas  │  │ Task 3: validate-network   │ │
│  │     (Bash-based)         │  │     (Bash-based)           │ │
│  │  (runs in parallel) ────→│  │←──── (runs in parallel)    │ │
│  │ ┌──────────────────────────────────────────────────────┐   │ │
│  │ │ Container: alpine:3.19                               │   │ │
│  │ │ Implementation: Bash script                          │   │ │
│  │ │ Purpose: Validate CPU, Memory, IP quotas             │   │ │
│  │ │ Depends on: API validator (compute API enabled)      │   │ │
│  │ │ Runs in parallel with: Network validator             │   │ │
│  │ │                                                        │   │ │
│  │ │ Checks:                                               │   │ │
│  │ │  1. CPU Quota:                                        │   │ │
│  │ │     Limit: 100 cores, Usage: 85, Available: 15       │   │ │
│  │ │     Required: 4 cores → ✅ PASSED                    │   │ │
│  │ │                                                        │   │ │
│  │ │  2. Memory Quota:                                     │   │ │
│  │ │     Limit: 256 GB, Usage: 200, Available: 56         │   │ │
│  │ │     Required: 16 GB → ✅ PASSED                      │   │ │
│  │ │                                                        │   │ │
│  │ │  3. IP Quota:                                         │   │ │
│  │ │     Limit: 50 IPs, Usage: 45, Available: 5           │   │ │
│  │ │     Required: 10 IPs → ✅ PASSED                     │   │ │
│  │ │                                                        │   │ │
│  │ │ Results:                                              │   │ │
│  │ │  status:  "SUCCESS"                                   │   │ │
│  │ │  message: "All quota checks passed..."               │   │ │
│  │ │  details: {"cpu_available": "15",                    │   │ │
│  │ │            "memory_available": "56",                  │   │ │
│  │ │            "ip_available": "5",                       │   │ │
│  │ │            "cpu_status": "PASSED", ...}               │   │ │
│  │ └──────────────────────────────────────────────────────┘   │ │
│  └────────────────────────────────────────────────────────────┘ │
│                    │                    │                      │
│           ┌────────┴────────┐  ┌────────┴────────┐            │
│           │  Both run in    │  │  Both run in    │            │
│           │  parallel after │  │  parallel after │            │
│           │  API validator  │  │  API validator  │            │
│           └─────────────────┘  └─────────────────┘            │
│                    │                    │                      │
│                    └──────────┬─────────┘                      │
│                               │ finally (always runs)          │
│                               ↓                                │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ Finally Task: aggregate-results                            │ │
│  │ ┌──────────────────────────────────────────────────────┐   │ │
│  │ │ Container: alpine:3.19                               │   │ │
│  │ │ Implementation: Bash script                          │   │ │
│  │ │ Purpose: Collect and aggregate all results           │   │ │
│  │ │                                                        │   │ │
│  │ │ Inputs:                                               │   │ │
│  │ │  - API task status + results                         │   │ │
│  │ │  - Quota task status + results                       │   │ │
│  │ │  - Network task status + results                     │   │ │
│  │ │                                                        │   │ │
│  │ │ Processing:                                           │   │ │
│  │ │  - Count passed/failed/skipped validations           │   │ │
│  │ │  - Determine overall status                          │   │ │
│  │ │  - Build comprehensive summary                       │   │ │
│  │ │                                                        │   │ │
│  │ │ Pipeline Results:                                     │   │ │
│  │ │  overall-status: "FAILED"                            │   │ │
│  │ │  validation-summary: {                               │   │ │
│  │ │    "overall_status": "FAILED",                       │   │ │
│  │ │    "total_validations": 3,                           │   │ │
│  │ │    "passed": 2,                                      │   │ │
│  │ │    "failed": 1,                                      │   │ │
│  │ │    "skipped": 0,                                     │   │ │
│  │ │    "failed_validators": "API",                       │   │ │
│  │ │    "validations": {...}                              │   │ │
│  │ │  }                                                    │   │ │
│  │ └──────────────────────────────────────────────────────┘   │ │
│  └────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

## Data Flow

```
┌─────────────────┐
│  PipelineRun    │
│  Parameters:    │
│  - project-id   │
│  - region       │
│  - network-name │
│  - subnet-name  │
└────────┬────────┘
         │
         ├──────────────────────────────────────────┐
         │                                          │
         ↓                                          ↓
┌────────────────────┐                    ┌──────────────────┐
│  API Validator     │                    │  Task Params     │
│  (validate-apis)   │                    │  Pass Through    │
└────────┬───────────┘                    └────────┬─────────┘
         │                                          │
         │ Results:                                 │
         │ - status: FAILED                         │
         │ - message: "..."                         │
         │ - details: {...}                         │
         │                                          │
         ↓                                          ↓
┌────────────────────┐                    ┌──────────────────┐
│  Quota Validator   │←───────────────────│  Receives params │
│ (validate-quotas)  │                    │  from Pipeline   │
└────────┬───────────┘                    └──────────────────┘
         │
         │ Results:
         │ - status: SUCCESS
         │ - message: "..."
         │ - details: {...}
         │
         ↓
┌────────────────────┐
│ Network Validator  │
│(validate-network)  │
└────────┬───────────┘
         │
         │ Results:
         │ - status: SUCCESS
         │ - message: "..."
         │ - details: {...}
         │
         ↓
┌────────────────────────────────────┐
│  Results Aggregator (Finally)      │
│                                    │
│  Collects:                         │
│  ┌──────────────────────────────┐  │
│  │ Task 1:                      │  │
│  │  execution: Failed           │  │
│  │  result: FAILED              │  │
│  │  message: "..."              │  │
│  │  details: {...}              │  │
│  └──────────────────────────────┘  │
│  ┌──────────────────────────────┐  │
│  │ Task 2:                      │  │
│  │  execution: Succeeded        │  │
│  │  result: SUCCESS             │  │
│  │  message: "..."              │  │
│  │  details: {...}              │  │
│  └──────────────────────────────┘  │
│  ┌──────────────────────────────┐  │
│  │ Task 3:                      │  │
│  │  execution: Succeeded        │  │
│  │  result: SUCCESS             │  │
│  │  message: "..."              │  │
│  │  details: {...}              │  │
│  └──────────────────────────────┘  │
│                                    │
│  Produces Pipeline Results:        │
│  - overall-status: FAILED          │
│  - validation-summary: {...}       │
└────────────────────────────────────┘
```

## Component Interaction

```
┌───────────────────────────────────────────────────────────────┐
│                        Kubernetes Cluster                      │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │                     Tekton Pipeline                      │ │
│  │                 (gcp-validation-pipeline)                │ │
│  └──────────────────────────────────────────────────────────┘ │
│                              │                                │
│           Creates ┌──────────┼──────────┐ Creates            │
│                   ↓          ↓          ↓                     │
│  ┌──────────────────┐ ┌────────────┐ ┌──────────────────┐   │
│  │   TaskRun-1      │ │ TaskRun-2  │ │   TaskRun-3      │   │
│  │  (validate-apis) │ │(validate-  │ │(validate-network)│   │
│  │                  │ │  quotas)   │ │                  │   │
│  └────────┬─────────┘ └─────┬──────┘ └─────────┬────────┘   │
│           │                 │                   │            │
│  Creates  ↓       Creates   ↓         Creates   ↓            │
│  ┌──────────────────┐ ┌────────────┐ ┌──────────────────┐   │
│  │   Pod-1          │ │   Pod-2    │ │   Pod-3          │   │
│  │ ┌──────────────┐ │ │┌──────────┐│ │┌──────────────┐  │   │
│  │ │ Container:   │ │ ││Container:││ ││ Container:   │  │   │
│  │ │ gcp-api-     │ │ ││alpine:   ││ ││alpine:       │  │   │
│  │ │ validator:   │ │ ││3.19      ││ ││3.19          │  │   │
│  │ │ latest       │ │ ││          ││ ││              │  │   │
│  │ │              │ │ ││ Runs:    ││ ││ Runs:        │  │   │
│  │ │ Runs:        │ │ ││ quota-   ││ ││ network-     │  │   │
│  │ │ /app/api-    │ │ ││ validator││ ││ validator.sh │  │   │
│  │ │ validator    │ │ ││ .sh      ││ ││              │  │   │
│  │ └──────────────┘ │ │└──────────┘│ │└──────────────┘  │   │
│  └──────────────────┘ └────────────┘ └──────────────────┘   │
│                                                               │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │               Finally TaskRun (aggregate-results)        │ │
│  │  ┌────────────────────────────────────────────────────┐  │ │
│  │  │  Pod-4                                             │  │ │
│  │  │  ┌──────────────────────────────────────────────┐  │  │ │
│  │  │  │  Container: alpine:3.19                      │  │  │ │
│  │  │  │  Runs: inline aggregation script            │  │  │ │
│  │  │  │  Reads all task results via parameters      │  │  │ │
│  │  │  └──────────────────────────────────────────────┘  │  │ │
│  │  └────────────────────────────────────────────────────┘  │ │
│  └──────────────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────────────┘
```

## Result Schema

```
PipelineRun
  ├── status
  │   ├── conditions
  │   │   ├── type: Succeeded
  │   │   ├── status: False (because one task failed)
  │   │   └── reason: Failed
  │   ├── taskRuns
  │   │   ├── validate-apis-<hash>
  │   │   │   ├── status: Failed
  │   │   │   └── taskResults
  │   │   │       ├── status: "FAILED"
  │   │   │       ├── message: "API validation failed..."
  │   │   │       └── details: "{...json...}"
  │   │   ├── validate-quotas-<hash>
  │   │   │   ├── status: Succeeded
  │   │   │   └── taskResults
  │   │   │       ├── status: "SUCCESS"
  │   │   │       ├── message: "All quota checks passed..."
  │   │   │       └── details: "{...json...}"
  │   │   └── validate-network-<hash>
  │   │       ├── status: Succeeded
  │   │       └── taskResults
  │   │           ├── status: "SUCCESS"
  │   │           ├── message: "All network checks passed..."
  │   │           └── details: "{...json...}"
  │   └── results
  │       ├── overall-status: "FAILED"
  │       └── validation-summary: "{
  │             "overall_status": "FAILED",
  │             "total_validations": 3,
  │             "passed": 2,
  │             "failed": 1,
  │             "failed_validators": "API",
  │             "validations": {...}
  │           }"
  └── ...
```

## Validator Result Schema

Each validator produces three results:
- **status**: "SUCCESS" or "FAILED"
- **message**: Human-readable description
- **details**: JSON object with diagnostic information

Pipeline aggregates all results into:
- **overall-status**: "SUCCESS" or "FAILED"
- **validation-reason**: One-sentence summary
- **validation-summary**: Complete JSON with all validator results

## Control Parameters

Each validator accepts a `force-failure` parameter to simulate failures:
- **api-force-failure**: Controls API validator behavior
- **quota-force-failure**: Controls Quota validator behavior
- **network-force-failure**: Controls Network validator behavior

Set to "true" for failure, "false" for success.
