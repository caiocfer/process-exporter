# Execution Spec: System Process Exporter

## Rules for Autonomous Execution
1. **Source of Truth:** Refer to `SDD.md` for overall architecture, metrics schema, and design details.
2. **Sequential Progress:** Execute ONE story at a time in exact order (Story 1 -> Story 2 -> ...).
3. **TDD Strict Rule:** For Story 2 and Story 3, write unit tests in `*_test.go` BEFORE implementing production logic.
4. **Validation Gateway:** A story is ONLY considered complete when its `Validation Command` passes with zero errors.
5. **No Regressions:** Before completing any story, run `go test -v ./...` to guarantee previous stories didn't break.

---

### Story 1: Project Setup & Foundation
**Goal:** Initialize project structure, Go module, and agent guidelines.

- [ ] Create folder structure: `cmd/exporter/`, `pkg/procfs/`, `pkg/collector/`, `deploy/helm/system-process-exporter/`.
- [ ] Initialize Go module (`go.mod`) with Go 1.26+ and `github.com/prometheus/client_golang`.
- [ ] Verify `AGENT.md` and `SDD.md` are loaded in workspace context.

**Validation Command:**
```bash
go mod tidy && go vet ./...

```

---

### Story 2: ProcFS Abstraction & Parsers (TDD)

**Goal:** Abstract `/proc` reading and implement parsers using TDD.

* [ ] Create `ProcReader` interface in `pkg/procfs/reader.go`.
* [ ] Create test mock using `fstest.MapFS` in `pkg/procfs/reader_test.go` with fake `/proc/<pid>/stat`, `/status`, and `/io` files.
* [ ] Implement `ParseStat` (utime, stime, state, processor/cpu_id).
* [ ] Implement `ParseStatus` (VmRSS, VmSize, threads).
* [ ] Implement `ParseIO` (read_bytes, write_bytes).

**Validation Command:**

```bash
go test -v -race ./pkg/procfs/...

```

---

### Story 3: Prometheus Collector & Metadata Cache

**Goal:** Implement the Prometheus `Collector` interface with caching and RSS filter.

* [ ] Create `ProcessCollector` in `pkg/collector/process.go` satisfying `prometheus.Collector`.
* [ ] Implement metadata TTL cache (`--cache-ttl`) for stable process labels (`cmdline`, `process_name`).
* [ ] Implement RSS threshold filtering (`--proc-min-rss-mb`).
* [ ] Write unit tests verifying generated OpenMetrics output matching Section 5 of `SDD.md`.

**Validation Command:**

```bash
go test -v -race ./pkg/collector/...

```

---

### Story 4: CLI Application & Docker Validation

**Goal:** Expose `/metrics` HTTP endpoint and validate containerized execution.

* [ ] Implement `cmd/exporter/main.go` parsing CLI flags defined in `SDD.md` (`--addr`, `--path`, `--proc`, `--proc-min-rss-mb`).
* [ ] Create multi-stage `Dockerfile` (`golang:1.26-bookworm` -> `distroless/cc-debian12`).
* [ ] Test local binary execution and `/metrics` curl scrape.

**Validation Command:**

```bash
go build -o process-exporter ./cmd/exporter && \
docker build -t process-exporter:test . && \
docker run --rm -d --name test-exporter -p 9835:9835 -v /proc:/host/proc:ro process-exporter:test --proc=/host/proc && \
sleep 2 && \
curl -s http://localhost:9835/metrics | grep "process_resident_memory_bytes" && \
docker stop test-exporter

```

---

### Story 5: Helm Chart & GitHub Pages Automation

**Goal:** Create DaemonSet manifests and GitHub Actions release workflow.

* [ ] Create Helm chart at `deploy/helm/system-process-exporter/` (`Chart.yaml`, `values.yaml`, `daemonset.yaml` with `hostPID: true`).
* [ ] Create `.github/workflows/helm-release.yaml` using `chart-releaser-action` targeting `gh-pages` branch.

**Validation Command:**

```bash
helm lint deploy/helm/system-process-exporter/

```

