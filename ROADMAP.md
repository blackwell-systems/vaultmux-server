# vaultmux-server Roadmap

This document outlines planned features and improvements for vaultmux-server. Items are prioritized based on user demand and architectural impact.

## v0.2.0 (Next Release)

### Under Consideration

**HTTP-Level RBAC for Cluster Service Pattern**
- Authenticate Kubernetes service account tokens via TokenReview API
- Namespace-based authorization policies
- Configurable secret access rules (prefix matching, explicit allow lists)
- Token caching for performance

**Status:** Gathering feedback from community on demand

**Why:** Enables secure multi-tenant cluster service deployments without relying solely on cloud IAM. Currently, sidecar pattern + IAM provides namespace isolation.

**Estimated effort:** 2-3 weeks

**Dependencies:** k8s.io/client-go, TokenReview API integration

---

## Future Considerations

### OpenAPI Spec Generation
- Auto-generate OpenAPI 3.0 spec from handlers
- Interactive API documentation via Swagger UI
- Client library generation support

**Priority:** Low - REST API is simple enough without formal spec

### Metrics and Observability
- Prometheus metrics for request counts, latency, errors
- Health check improvements (backend connectivity probes)
- Structured logging with configurable levels

**Priority:** Medium - useful for production deployments

### Secret Caching
- Optional in-memory cache with TTL
- Reduce backend API calls
- Configurable per-secret or global policy

**Priority:** Low - adds complexity, most backends are fast enough

### Additional Backend Support
- HashiCorp Vault integration
- Kubernetes Secrets (for migration scenarios)
- Custom backend plugin system

**Priority:** Medium - depends on demand from vaultmux library adoption

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

- Secret synchronization between backends (use migration scripts)
- Replace cloud-native secret managers (we wrap them, don't reimplement)
- Become a general-purpose Kubernetes operator (intentionally runtime API, not declarative)
- Support non-cloud backends in production (local backends for testing only)

---

**Last updated:** 2026-01-27
