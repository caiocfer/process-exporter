
# Agent Guidelines & Rules: System Process Exporter

## 1. Context & Architecture Rules
- Always consult `SDD.md` for exact metrics naming, label key names, CLI flag defaults, and architecture specifications.
- Maintain strict alignment with OpenMetrics formatting specified in `SDD.md`.

## 2. Development Principles
- **Strict TDD:** Always write failing unit tests using mock filesystems (`fstest.MapFS` or `afero.MemMapFs`) BEFORE writing implementation code.
- **Zero Raw Warnings:** Code must pass `golangci-lint` (if available) and `go vet` with zero warnings.
- **ProcFS Efficiency:** Minimize heap allocations when scanning `/proc`. Use buffer reuse or `sync.Pool` where applicable.
- **No Direct Hardware/Host Bindings:** Always interact with the OS through the `ProcReader` interface abstraction to allow 100% test coverage.

## 3. Test & Validation Constraints
- Use Go 1.26+ standard library conventions.
- Never mark a story as complete in `SPEC.md` without running its exact `Validation Command`.
- Container validation must explicitly test host process visibility using `-v /proc:/host/proc:ro` and `--proc=/host/proc`.

## 4. Helm & CI/CD Rules
- Helm chart templates must be placed under `deploy/helm/system-process-exporter/`.
- DaemonSet manifests MUST include `hostPID: true` to access host processes from inside the container.
- Always bump `version` and `appVersion` in `Chart.yaml` when modifying metrics schema or Go application logic.