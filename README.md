# vaultmux-server

> Language-agnostic secrets control plane for Kubernetes and cloud-native systems

[![Blackwell Systems™](https://raw.githubusercontent.com/blackwell-systems/blackwell-docs-theme/main/badge-trademark.svg)](https://github.com/blackwell-systems)
[![Go Reference](https://pkg.go.dev/badge/github.com/blackwell-systems/vaultmux-server.svg)](https://pkg.go.dev/github.com/blackwell-systems/vaultmux-server)
[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://go.dev/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-326CE5?logo=kubernetes&logoColor=white)](https://kubernetes.io/)
[![Docker](https://img.shields.io/badge/Docker-2496ED?logo=docker&logoColor=white)](https://hub.docker.com/)
[![License](https://img.shields.io/badge/license-MIT%20OR%20Apache--2.0-blue.svg)](LICENSE-APACHE)

**Language-agnostic secret management for polyglot Kubernetes environments.** Deploy as sidecar or cluster service. Supports AWS Secrets Manager, GCP Secret Manager, and Azure Key Vault with zero client SDK dependencies.

```bash
# Any language, any backend, one HTTP endpoint
curl http://localhost:8080/v1/secrets/api-key
```

---

## Why vaultmux-server?

Kubernetes teams run polyglot stacks: Python services for ML, Node.js APIs, Go microservices, Rust workers. Each language needs secret management, but native SDKs create friction:

- Maintaining vaultmux ports in 4+ languages
- Each team duplicates integration work  
- No centralized backend switching (dev uses pass, prod uses AWS)
- SDK version drift across services

vaultmux-server wraps the battle-tested vaultmux library in an HTTP API. All languages fetch secrets with plain HTTP—no SDKs required. Deploy as sidecar (per-pod isolation) or cluster service (shared).

Works with any language that can make HTTP requests: Python, Node.js, Go, Rust, Java, C#, Ruby. Centralized configuration means changing cloud providers doesn't require touching application code. Kubernetes-native deployment patterns with health checks, graceful shutdown, and ~20MB distroless containers.

---

## Quick Start (Kubernetes)

### Sidecar Pattern (Recommended)

**Best for:** Per-app secret isolation, different backends per namespace, minimal latency.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    spec:
      containers:
      # Your application
      - name: app
        image: my-python-app:latest
        env:
        - name: VAULTMUX_ENDPOINT
          value: "http://localhost:8080"
      
      # vaultmux-server sidecar
      - name: vaultmux
        image: ghcr.io/blackwell-systems/vaultmux-server:latest
        ports:
        - containerPort: 8080
        env:
        - name: VAULTMUX_BACKEND
          value: "aws"
        - name: AWS_REGION
          value: "us-east-1"
        resources:
          limits:
            cpu: 100m
            memory: 128Mi
```

**Your app fetches secrets:**

```python
# Python
import requests
secret = requests.get("http://localhost:8080/v1/secrets/api-key").json()
print(secret["value"])
```

```javascript
// Node.js
const res = await fetch("http://localhost:8080/v1/secrets/api-key");
const secret = await res.json();
console.log(secret.value);
```

```go
// Go
resp, _ := http.Get("http://localhost:8080/v1/secrets/api-key")
var secret struct{ Value string }
json.NewDecoder(resp.Body).Decode(&secret)
```

See [examples/sidecar/](examples/sidecar/) for complete manifests.

---

### Cluster Service Pattern

**Best for:** Shared secrets, lower resource usage, centralized management.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: vaultmux
spec:
  replicas: 2
  selector:
    matchLabels:
      app: vaultmux
  template:
    spec:
      containers:
      - name: vaultmux-server
        image: ghcr.io/blackwell-systems/vaultmux-server:latest
        env:
        - name: VAULTMUX_BACKEND
          value: "gcp"
        - name: GCP_PROJECT_ID
          value: "my-project"
---
apiVersion: v1
kind: Service
metadata:
  name: vaultmux
spec:
  selector:
    app: vaultmux
  ports:
  - port: 8080
```

**All apps call the service:**

```bash
# From any pod in the cluster
curl http://vaultmux.default.svc.cluster.local:8080/v1/secrets/api-key
```

See [examples/cluster-service/](examples/cluster-service/) for complete manifests.

---

## Use Cases

### Multi-Language Teams

**Scenario:** Platform team supporting Python, Node.js, Go, and Rust services.

**Without vaultmux-server:** Implement secret fetching in 4 languages, maintain 4 SDKs, handle version updates across all services.

**With vaultmux-server:** Deploy once, all services use HTTP. Update backend in one place (ConfigMap).

```yaml
# Change backend cluster-wide
apiVersion: v1
kind: ConfigMap
metadata:
  name: vaultmux-config
data:
  backend: "aws"  # Change to "gcp" without redeploying apps
  region: "us-east-1"
```

---

### Environment-Based Backends

**Scenario:** Staging uses AWS Secrets Manager, production uses GCP Secret Manager.

**Solution:** Same app manifest, different ConfigMap per namespace.

```yaml
# staging namespace - uses AWS
VAULTMUX_BACKEND: aws
AWS_REGION: us-east-1

# production namespace - uses GCP
VAULTMUX_BACKEND: gcp
GCP_PROJECT_ID: prod-project-123
```

No code changes. Same container image across all environments.

---

### CI/CD Testing

**Scenario:** Integration tests need secrets without using production credentials.

**Solution:** Run vaultmux-server with emulator backends (LocalStack for AWS, GCP Secret Manager Emulator for GCP).

```yaml
# .github/workflows/integration-test.yml
services:
  localstack:
    image: localstack/localstack:latest
    ports:
      - 4566:4566
  
  vaultmux:
    image: ghcr.io/blackwell-systems/vaultmux-server:latest
    env:
      VAULTMUX_BACKEND: aws
      AWS_REGION: us-east-1
      AWS_ENDPOINT: http://localstack:4566
    ports:
      - 8080:8080

jobs:
  test:
    steps:
      - run: |
          # Create test secret in LocalStack
          aws --endpoint-url=http://localhost:4566 secretsmanager create-secret \
            --name test-key --secret-string test-value
          
          # Test vaultmux-server
          curl http://localhost:8080/v1/secrets/test-key
          pytest tests/integration/
```

---

## REST API

### List Secrets

```bash
GET /v1/secrets
```

**Response:**
```json
{
  "secrets": ["api-key", "db-password", "ssh-key"]
}
```

---

### Get Secret

```bash
GET /v1/secrets/{name}
```

**Response:**
```json
{
  "name": "api-key",
  "value": "sk-secret123"
}
```

---

### Create Secret

```bash
POST /v1/secrets
Content-Type: application/json

{
  "name": "api-key",
  "value": "sk-secret123"
}
```

**Response:** `201 Created`

---

### Update Secret

```bash
PUT /v1/secrets/{name}
Content-Type: application/json

{
  "value": "sk-newsecret456"
}
```

**Response:** `200 OK`

---

### Delete Secret

```bash
DELETE /v1/secrets/{name}
```

**Response:** `204 No Content`

---

### Health Check

```bash
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "backend": "aws"
}
```

---

## Supported Backends

| Backend | Environment Variables | Use Case |
|---------|----------------------|----------|
| **AWS Secrets Manager** | `AWS_REGION`, `AWS_ENDPOINT` | Production (AWS EKS) |
| **GCP Secret Manager** | `GCP_PROJECT_ID` | Production (GCP GKE) |
| **Azure Key Vault** | Azure SDK env vars | Production (Azure AKS) |

**For CI/CD testing:** Use emulators ([LocalStack](https://localstack.cloud/) for AWS, [GCP Secret Manager Emulator](https://github.com/blackwell-systems/gcp-secret-manager-emulator) for GCP) with endpoint overrides (`AWS_ENDPOINT`, etc).

**For developer workstations:** Use the [vaultmux library](https://github.com/blackwell-systems/vaultmux) directly for Bitwarden, 1Password, pass, or Windows Credential Manager integration.

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `VAULTMUX_BACKEND` | - | Backend type: `aws`, `gcp`, `azure` (required) |
| `VAULTMUX_PREFIX` | `vaultmux` | Secret name prefix |
| `AWS_REGION` | - | AWS region (required for AWS backend) |
| `AWS_ENDPOINT` | - | AWS endpoint override (for LocalStack/emulators) |
| `GCP_PROJECT_ID` | - | GCP project ID (required for GCP backend) |

---

## Helm Installation

```bash
# Install from Helm repo (once published)
helm repo add blackwell-systems https://blackwell-systems.github.io/charts

# Deploy as cluster service
helm install vaultmux blackwell-systems/vaultmux-server \
  --set backend.type=aws \
  --set aws.region=us-east-1

# Deploy with sidecar injection
helm install vaultmux blackwell-systems/vaultmux-server \
  --set sidecar.enabled=true
```

**Custom values:**

```yaml
# values.yaml
replicaCount: 3

backend:
  type: gcp
  prefix: myapp

gcp:
  projectId: prod-project-123

resources:
  limits:
    cpu: 200m
    memory: 256Mi
```

```bash
helm install vaultmux blackwell-systems/vaultmux-server -f values.yaml
```

See [helm/vaultmux-server/](helm/vaultmux-server/) for full chart documentation.

---

## Architecture

```
┌─────────────────────────────────────────┐
│           Kubernetes Cluster            │
├─────────────────────────────────────────┤
│  ┌────────────┐     ┌────────────┐      │
│  │ Python App │────▶│ Node.js App│     │
│  └─────┬──────┘     └──────┬─────┘      │
│        │ HTTP               │ HTTP      │
│        └────────┬───────────┘           │
│                 ▼                       │
│        ┌────────────────┐               │
│        │ vaultmux-server│               │
│        │   (REST API)   │               │
│        └────────┬───────┘               │
│                 │                       │
│        ┌────────┴────────┐              │
│        ▼                 ▼              │
│   [pass (dev)]    [AWS Secrets (prod)]  │
│                    [GCP Secrets (stg)]  │
└─────────────────────────────────────────┘
```

---

## Deployment Patterns Comparison

| Pattern | Latency | Resource Usage | Isolation | Best For |
|---------|---------|----------------|-----------|----------|
| **Sidecar** | ~1ms (localhost) | High (one per pod) | Per-app | Different backends per namespace, strict isolation |
| **Cluster Service** | ~5-10ms (in-cluster) | Low (2-3 replicas total) | Shared | Centralized management, cost optimization |

**Recommendation:** Start with sidecar for flexibility, move to cluster service if resource usage is a concern.

---

## Security Considerations

### Network Isolation
- vaultmux-server runs **inside** the cluster, not exposed to internet
- Use Kubernetes NetworkPolicies to restrict pod-to-pod access
- Consider service mesh (Istio, Linkerd) for mTLS between pods

### Authentication & Authorization

**Sidecar Pattern (Recommended for Multi-Tenant):**

Namespace isolation via Kubernetes service accounts + cloud IAM:
- Each namespace has its own vaultmux-server pod
- Different Kubernetes service account per namespace
- Cloud provider maps service account → IAM identity
  - AWS: IAM Roles for Service Accounts (IRSA)
  - GCP: Workload Identity
  - Azure: Managed Identity
- `test` namespace → `test` IAM role → `test/*` secrets only
- `prod` namespace → `prod` IAM role → `prod/*` secrets only

Cloud provider enforces the authorization boundary. No HTTP-level RBAC needed.

**[📖 Complete sidecar setup guide](docs/SIDECAR_RBAC.md)** - Step-by-step instructions for AWS IRSA, GCP Workload Identity, and Azure Managed Identity.

**Cluster Service Pattern:**

Currently relies on network isolation (any pod in cluster can call API). For multi-tenant isolation without IAM, HTTP-level RBAC is on the roadmap (authenticate Kubernetes service account tokens, authorize based on namespace).

Use sidecar pattern if you need namespace isolation today.

### Secrets in Transit
- Sidecar: localhost traffic (no TLS needed)
- Cluster service: use service mesh for mTLS

---

## Local Development

### Run Without Kubernetes

```bash
# Install
go install github.com/blackwell-systems/vaultmux-server/cmd/server@latest

# Run with AWS (using LocalStack for local testing)
VAULTMUX_BACKEND=aws \
  AWS_REGION=us-east-1 \
  AWS_ENDPOINT=http://localhost:4566 \
  server

# Run with GCP Secret Manager
VAULTMUX_BACKEND=gcp \
  GCP_PROJECT_ID=my-project \
  server
```

### Build from Source

```bash
# Clone
git clone https://github.com/blackwell-systems/vaultmux-server
cd vaultmux-server

# Build
go build -o vaultmux-server ./cmd/server

# Run
./vaultmux-server
```

### Docker

```bash
# Build
docker build -t vaultmux-server:latest .

# Run with LocalStack
docker run -p 8080:8080 \
  -e VAULTMUX_BACKEND=aws \
  -e AWS_REGION=us-east-1 \
  -e AWS_ENDPOINT=http://localstack:4566 \
  vaultmux-server:latest
```

---

## Trade-offs: Library vs Server

| Aspect | vaultmux (library) | vaultmux-server |
|--------|-------------------|-----------------|
| **Performance** | Fastest (in-process) | Network hop (~1-10ms) |
| **Language support** | Go only | All languages |
| **Type safety** | Full Go types | JSON over HTTP |
| **Deployment complexity** | App-level | Kubernetes sidecar/service |
| **Configuration** | Per-app code | Centralized ConfigMap |
| **Best for** | Go-only teams | Polyglot Kubernetes environments |

**Use the library when:** Building Go-only applications where type safety matters.

**Use vaultmux-server when:** Kubernetes cluster with multiple languages or want centralized backend switching.

---

## Observability

### Metrics (Roadmap)

vaultmux-server will expose Prometheus metrics on `/metrics`:

- `vaultmux_requests_total{method, status}` - Request count by endpoint
- `vaultmux_request_duration_seconds{method}` - Request latency
- `vaultmux_backend_errors_total{backend}` - Backend error count

### Logging

Structured JSON logs:

```json
{
  "level": "info",
  "method": "GET",
  "path": "/v1/secrets/api-key",
  "status": 200,
  "latency": "12ms",
  "client_ip": "10.244.1.5"
}
```

---

## FAQ

### Why not External Secrets Operator?

**External Secrets Operator** syncs secrets from cloud providers into Kubernetes Secrets. vaultmux-server fetches secrets on-demand via HTTP.

**Use External Secrets when:** You want secrets as native Kubernetes Secrets.

**Use vaultmux-server when:** You want runtime secret fetching with backend flexibility (dev uses pass, prod uses AWS).

### Why not a Kubernetes Operator?

CRDs and operators inject external state into the Kubernetes control plane: data lives in etcd, operators reconcile it. That's powerful, but it makes the control plane part of your secret lifecycle.

vaultmux-server takes the opposite approach: keep secrets outside cluster state entirely. Kubernetes is just one runtime it can live in, not the system of record.

**Operator pattern (External Secrets Operator):**
```
K8s API → etcd → operator → external secret backend
```
Secrets stored in cluster state, declarative reconciliation

**vaultmux-server pattern:**
```
App → vaultmux-server → external secret backend
```
No reconciliation loop, no control-plane storage, runtime access only

**Use operators when:** You want secrets as native Kubernetes Secrets with declarative management

**Use vaultmux-server when:**
- Secrets must stay outside cluster state (security requirement)
- Runtime backend switching without YAML changes (dev uses pass, prod uses AWS)  
- Polyglot environments preferring HTTP over K8s client libraries
- Running outside Kubernetes (VMs, CI, local dev with same API)

### Can I use this outside Kubernetes?

Yes, vaultmux-server is just an HTTP server. Run locally or in VMs. Kubernetes patterns (sidecar, service) are optional.

### Does this cache secrets?

Not currently. Each request hits the backend. Future versions may add caching with configurable TTL.

### What about secret rotation?

Secret rotation happens at the backend level (AWS Secrets Manager rotation, Bitwarden sync). vaultmux-server always fetches the latest version.

---

## Roadmap

See [ROADMAP.md](ROADMAP.md) for planned features and development priorities.

**Upcoming:**
- HTTP-level RBAC for cluster service pattern (under consideration)
- Prometheus metrics
- OpenAPI spec generation

---

## Contributing

Contributions welcome! See [ROADMAP.md](ROADMAP.md) for planned features and [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## Related Projects

- **[vaultmux](https://github.com/blackwell-systems/vaultmux)** - The Go library this server wraps
- **[vaultmux-rs](https://github.com/blackwell-systems/vaultmux-rs)** - Rust port of vaultmux

---

## License

Licensed under either of:

- Apache License, Version 2.0 ([LICENSE-APACHE](LICENSE-APACHE) or http://www.apache.org/licenses/LICENSE-2.0)
- MIT license ([LICENSE-MIT](LICENSE-MIT) or http://opensource.org/licenses/MIT)

at your option.

---

## Brand

The **code** in this repository is dual-licensed (MIT OR Apache 2.0). The **Blackwell Systems™** name and logo are protected trademarks. See [BRAND.md](BRAND.md) for usage guidelines.

---

## Maintained By

**Dayna Blackwell** — founder of Blackwell Systems, building reference infrastructure for cloud-native development.

[GitHub](https://github.com/blackwell-systems) · [LinkedIn](https://linkedin.com/in/dayna-blackwell) · [Blog](https://blog.blackwell-systems.com)
