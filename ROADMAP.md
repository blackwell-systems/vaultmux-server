# vaultmux-server Roadmap

This document outlines planned features and improvements for vaultmux-server. Items are prioritized based on user demand, architectural impact, and alignment with the core principle: **Kubernetes as compute, not storage.**

## v0.2.0 (Next Release)

Focus: Production readiness - observability, compliance, and operational quality.

### Planned

**Native Sidecar Support (Kubernetes 1.28+)**
- Update deployment examples to use init containers with `restartPolicy: Always`
- Prevents sidecars from holding Job/CronJob workloads open after main container exits
- Backward compatible examples for K8s < 1.28
- Helm chart conditional support based on K8s version

**Why:** Solves real bug where batch workloads hang indefinitely waiting for sidecar termination. K8s 1.28+ native sidecars properly manage lifecycle for short-lived workloads.

**Estimated effort:** 1 week (examples, docs, Helm chart updates)

**Dependencies:** Kubernetes 1.28+ for native sidecar feature

---

**Prometheus Metrics**
- `/metrics` endpoint exposing Prometheus-compatible metrics
- Request counts by method, path, status code
- Latency histograms for API and backend calls
- Backend error rates by operation type
- Example Grafana dashboard and ServiceMonitor manifests

**Why:** Production deployments require observability. Essential for monitoring performance, detecting issues, and capacity planning.

**Estimated effort:** 1 week

**Dependencies:** Prometheus client library

---

**Structured Audit Logging**
- JSON-formatted audit logs with request context
- Log secret access operations (get, create, update, delete)
- Include namespace, pod name, service account for Kubernetes deployments
- Configurable log levels (audit, info, debug)
- Document integration with CloudWatch Logs, Cloud Logging, Elasticsearch

**Why:** Security and compliance teams need "who accessed what secret when" for audit trails. Essential for regulated environments (healthcare, finance, government).

**Estimated effort:** 1 week

**Dependencies:** Structured logging library

---

## v0.3.0 (Future Release)

Focus: Performance optimization and operational improvements.

### Planned

**Batch Secret Fetching**
- New endpoint: `POST /v1/secrets:batchGet` for fetching multiple secrets
- Parallel backend calls to reduce total latency
- Apps with 10+ secrets: 1000ms sequential → 100ms parallel
- Partial success handling (return successful secrets, report errors for failed ones)

**Why:** Applications fetching many secrets at startup pay cumulative latency penalty. Batch endpoint reduces startup time by parallelizing backend calls.

**Estimated effort:** 2 weeks

---

**Enhanced Health Checks**
- `/health` tests actual backend connectivity (not just static response)
- `/ready` endpoint for Kubernetes readiness probes
- Report credential expiration time for cloud provider credentials
- Latency metrics for backend connectivity test
- Warn when credentials expire soon

**Why:** Current health check is passive. Active checks catch misconfigurations (wrong IAM role, network issues, expired credentials) before production failures.

**Estimated effort:** 1 week

---

**Secret Rotation Signals**
- Optional: vaultmux-server polls backend for secret version changes
- Send SIGHUP to app container when configured secrets rotate
- Apps catch signal and re-fetch secrets from localhost
- Configurable watch list via `VAULTMUX_WATCH_SECRETS` environment variable

**Why:** Enables zero-downtime credential rotation. Apps cache secrets in memory for performance; signal mechanism tells them when to refresh.

**Estimated effort:** 2-3 weeks

**Dependencies:** Backend polling logic, container signal handling

---

## Under Consideration

These features may be added based on community demand. Open an issue if you have a use case.

**Additional Backend Support**
- HashiCorp Vault integration
- Kubernetes Secrets backend (for migration scenarios)
- Custom backend plugin system

**Status:** Depends on demand from vaultmux library adoption and user requests

---

**OpenAPI Spec Generation**
- Auto-generate OpenAPI 3.0 spec from handlers
- Interactive API documentation via Swagger UI

**Status:** API is simple (5 endpoints), formal spec may not justify maintenance burden

---

## Recently Completed

### v0.1.0 (Released)
- Initial release with AWS, GCP, Azure backend support
- Sidecar and cluster service deployment patterns
- Helm chart with configurable backends
- Health checks and graceful shutdown
- Production-ready logging and error handling
- Backend validation (cloud providers only)

---

## Contributing

Have a feature request or want to contribute? Open an issue describing your use case and proposed solution. Priority is determined by:
- Number of users requesting the feature
- Alignment with project goals (language-agnostic, cloud-native)
- Maintenance burden vs value delivered
- Availability of workarounds

---

## Non-Goals

**What vaultmux-server will NOT do:**

- **Secret caching** - Violates "cluster as compute" model. Secrets would live in pod memory, making the cluster part of the secret lifecycle. Apps should cache in their own memory if needed.
- **HTTP-level RBAC** - Sidecar pattern + cloud IAM provides superior namespace isolation enforced by the cloud provider. Users needing isolation should use sidecars, not add a weaker authorization layer.
- **Unix domain sockets** - Marginal security improvement (localhost TCP is already pod-isolated) with high complexity cost. Not worth maintaining two communication paths.
- **Secret synchronization between backends** - Use migration scripts or external tools.
- **Replace cloud-native secret managers** - We wrap them, don't reimplement them.
- **Become a general-purpose Kubernetes operator** - Intentionally a runtime API, not declarative CRDs.
- **Support non-cloud backends in production** - AWS/GCP/Azure only. Local backends (pass, bitwarden) supported via vaultmux library for development.

---

**Last updated:** 2026-01-31
