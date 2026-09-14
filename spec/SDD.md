# Software Design Document: System Process Exporter

## 1. Introduction

The System Process Exporter is a lightweight, standalone Go application designed to collect and expose detailed per-process resource utilization for processes running on the host system via `/proc` filesystem parsing. This tool provides deep observability into process CPU usage, memory footprint, thread counts, and disk I/O directly into Prometheus/OpenMetrics.

## 2. Goals & Objectives

### 2.1 Core Objectives

* **Standalone Execution:** Compile to a single binary with zero external runtime dependencies.
* **System Process Monitoring:** Monitor all Linux system processes directly via `/proc` filesystem parsing.
* **Prometheus / OpenMetrics Compliance:** Expose metrics via the `/metrics` endpoint using standard exposition formats.
* **Test-Driven Development (TDD):** Implement collectors using TDD with isolated mock filesystems (`fstest.MapFS` / `afero.MemMapFs`).
* **Low Overhead:** Apply TTL metadata caching, efficient `/proc` reading strategies, and RSS thresholds to minimize scrape overhead.

### 2.2 Deployment & Release

* **Helm Chart:** Maintained at `deploy/helm/system-process-exporter/` for Kubernetes (k3s/k8s) DaemonSet deployment.
* **Helm Repository:** Hosted on the `gh-pages` branch serving `index.yaml` and chart archives.
* **Dockerfile:** Multi-stage build (`golang:1.26-bookworm` → `gcr.io/distroless/cc-debian12`).
* **DaemonSet:** Runs across system nodes with `hostPID: true` and hostPath mounts for `/proc`.

## 3. Technology Stack

* **Language:** Go 1.26+
* **Metrics Library:** `prometheus/client_golang`
* **System Interface:** Standard library `os` and `/proc` filesystem parsing via abstract `ProcReader` interface.

## 4. Architecture & Design

### 4.1 System Overview

```
┌────────────────────────────────────────────────────────┐
│                   /proc File System                    │
└───────────────────────────┬────────────────────────────┘
                            │
                  ┌─────────▼──────────┐
                  │   ProcFS Reader    │
                  └─────────┬──────────┘
                            │
                  ┌─────────▼──────────┐
                  │ System Process     │
                  │ Collector          │
                  └─────────┬──────────┘
                            │
                  ┌─────────▼──────────┐
                  │ Prometheus Registry│
                  └────────────────────┘

```

### 4.2 CLI Flags

| Flag | Default | Description |
| --- | --- | --- |
| `--addr` | `:9835` | Listen address |
| `--path` | `/metrics` | Metrics HTTP path |
| `--cache-ttl` | `60s` | Process metadata cache TTL |
| `--proc` | `/proc` | Path to proc filesystem |
| `--proc-min-rss-mb` | `10` | Minimum RSS threshold (MB) to expose system processes |

## 5. Metrics Schema (OpenMetrics)

| Metric Name | Type | Labels | Description |
| --- | --- | --- | --- |
| `process_cpu_seconds_total` | Counter | `pid`, `process_name`, `cmdline`, `state`, `cpu_id` | Total CPU time consumed (utime + stime) |
| `process_resident_memory_bytes` | Gauge | `pid`, `process_name`, `cmdline` | Resident memory size (VmRSS) |
| `process_virtual_memory_bytes` | Gauge | `pid`, `process_name`, `cmdline` | Virtual memory size (VmSize) |
| `process_threads_total` | Gauge | `pid`, `process_name` | Number of active threads |
| `process_io_read_bytes_total` | Counter | `pid`, `process_name` | Disk read bytes from `/proc/<pid>/io` |
| `process_io_write_bytes_total` | Counter | `pid`, `process_name` | Disk write bytes from `/proc/<pid>/io` |

## 6. Task Breakdown & User Stories

### Story 1: Project Setup & Agent Guidelines

**As a** developer,

**I want** to set up the repository foundation and project guidelines (`AGENT.md`),

**So that** coding standards, validation steps, and AI agent interactions remain consistent.

* **Tasks:**
* Create project structure (`cmd/exporter`, `pkg/procfs`, `pkg/collector`).
* Add `AGENT.md` defining TDD rules, validation scripts, and lint requirements.
* Set up base `go.mod` with Go 1.26+ and `prometheus/client_golang`.



### Story 2: ProcFS Abstraction & Parsers (TDD)

**As a** system engineer,

**I want** unit-tested parsers for `/proc/<pid>/stat`, `/proc/<pid>/status`, and `/proc/<pid>/io`,

**So that** process metadata and metrics can be extracted safely without runtime crashes.

* **Tasks:**
* Define `ProcReader` interface to abstract host filesystem access.
* Implement unit tests using mock filesystem (`fstest.MapFS`) covering valid, corrupt, and missing proc entries.
* Write parser logic for `stat` (CPU jiffies, state, core ID), `status` (VmRSS, VmSize, threads), and `io` (read/write bytes).



### Story 3: Prometheus Collector Implementation

**As an** SRE/DevOps engineer,

**I want** a Prometheus `Collector` exposing OpenMetrics for non-GPU processes,

**So that** I can scrape CPU, RAM, and I/O metrics per process via standard Prometheus scraping.

* **Tasks:**
* Implement `prometheus.Collector` executing `/proc` iteration on `Collect()`.
* Add metadata caching with configurable TTL (`--cache-ttl`).
* Implement RSS filter (`--proc-min-rss-mb`) to reduce metric cardinality.



### Story 4: Binary CLI & Validation Pipeline

**As a** developer,

**I want** a CLI runner and local testing script,

**So that** I can build the binary, spin up a container, and verify metrics programmatically.

* **Tasks:**
* Implement `main.go` supporting CLI flags (`--addr`, `--path`, `--proc`, `--proc-min-rss-mb`).
* Write multi-stage `Dockerfile` (`golang:1.26-bookworm` → `distroless/cc-debian12`).
* Build local shell/validation step running `go test`, binary execution with `curl`, and Docker test (`-v /proc:/host/proc:ro`).



### Story 5: Helm Chart & GitHub Pages Release

**As a** Kubernetes cluster operator,

**I want** a Helm chart published to a GitHub Pages repository,

**So that** I can deploy the exporter as a DaemonSet using standard Helm repos.

* **Tasks:**
* Create Helm chart manifests at `deploy/helm/system-process-exporter/` (`DaemonSet`, `Service`, `ServiceAccount`).
* Configure DaemonSet spec with `hostPID: true` and `/proc` hostPath mount.
* Configure GitHub Actions workflow using `chart-releaser-action` targeting the `gh-pages` branch.



## 7. Development & Validation Strategy

### 7.1 TDD Workflow

1. Write failing unit tests for `/proc/<pid>/stat`, `/proc/<pid>/status`, and `/proc/<pid>/io` parsers using virtual memory filesystems.
2. Implement collectors in Go 1.26 to pass tests.
3. Benchmark and optimize allocations using buffer pools (`sync.Pool`).

### 7.2 Agent Rules (`AGENT.md`)

Maintain `AGENT.md` at the project root defining project guidelines:

* Strictly follow TDD.
* Enforce validation via binary execution and containerized testing (`-v /proc:/host/proc:ro`).
* Maintain zero lint issues with `golangci-lint` and `go vet`.

### 7.3 Local Testing Pipeline

```bash
# Unit Tests
go test -v -cover -race ./...

# Binary Validation
go build -o process-exporter ./cmd/exporter
./process-exporter --addr=":9835" &
EXPORTER_PID=$!
sleep 2
curl -s http://localhost:9835/metrics | grep "process_cpu_seconds_total"
kill $EXPORTER_PID

# Docker Validation
docker build -t process-exporter:test .
docker run --rm -d --name test-process-exporter -p 9835:9835 -v /proc:/host/proc:ro process-exporter:test --proc=/host/proc
curl -s http://localhost:9835/metrics | grep "process_resident_memory_bytes"
docker stop test-process-exporter

```

## 8. Deployment & Helm Repository (GitHub Pages)

### 8.1 GitHub Pages Repository Structure (`gh-pages` branch)

```
gh-pages/
├── index.yaml
└── system-process-exporter-0.1.0.tgz

```

### 8.2 Release Workflow (`.github/workflows/helm-release.yaml`)

Automation packages the Helm chart located at `deploy/helm/system-process-exporter/` and updates the index on the `gh-pages` branch using `chart-releaser-action`.