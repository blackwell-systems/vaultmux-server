# Sidecar Pattern Examples

This directory contains examples for deploying vaultmux-server as a sidecar container alongside your application.

## Native Sidecar Pattern (Kubernetes 1.28+)

**Recommended for new deployments.**

The native sidecar pattern uses Kubernetes 1.28+ init containers with `restartPolicy: Always`. This ensures proper lifecycle management, especially for Job and CronJob workloads.

**File:** `native-sidecar-deployment.yaml`

### Key Features

- vaultmux-server starts before main container
- Automatically restarts if it crashes
- **Properly terminates when main container exits** (critical for Jobs)
- No pod lifecycle issues with batch workloads

### How It Works

```yaml
spec:
  initContainers:
  - name: vaultmux-server
    restartPolicy: Always  # Native sidecar feature
    # ... rest of config
    
  containers:
  - name: app
    # Main application
```

**Lifecycle:**
1. vaultmux-server starts first (init container)
2. Main app starts after vaultmux is ready
3. Main app completes and exits
4. **Kubernetes terminates vaultmux-server automatically**
5. Pod completes successfully

### When to Use

- **Job/CronJob workloads** - Prevents pods from hanging after completion
- **Kubernetes 1.28+** - Requires native sidecar support
- **New deployments** - Recommended default pattern

### Usage

```bash
kubectl apply -f native-sidecar-deployment.yaml
```

---

## Legacy Sidecar Pattern (Kubernetes < 1.28)

**For older Kubernetes versions.**

The legacy pattern runs vaultmux-server as a regular container alongside the main application.

**File:** `deployment.yaml`

### Limitations

- vaultmux-server continues running after main container exits
- **Not suitable for Job/CronJob workloads** without workarounds
- Pods may hang indefinitely waiting for sidecar termination

### When to Use

- **Kubernetes < 1.28** - Native sidecars not available
- **Long-running services** - Deployment, StatefulSet (not Jobs)
- **Backward compatibility** - Existing deployments

### Workarounds for Jobs (if using legacy pattern)

If you must use the legacy pattern with Jobs:

```yaml
# Option 1: preStop hook with forced termination
lifecycle:
  preStop:
    exec:
      command: ["/bin/sh", "-c", "sleep 5"]

# Option 2: Shared process namespace (advanced)
spec:
  shareProcessNamespace: true
  # Main container can signal sidecar to exit
```

**Better solution:** Upgrade to Kubernetes 1.28+ and use native sidecars.

### Usage

```bash
kubectl apply -f deployment.yaml
```

---

## Comparison

| Feature | Native Sidecar (K8s 1.28+) | Legacy Sidecar (K8s < 1.28) |
|---------|----------------------------|------------------------------|
| **Job/CronJob support** | ✅ Terminates automatically | ❌ Hangs indefinitely |
| **Startup order** | ✅ Guaranteed (init container) | ⚠️ Parallel start |
| **Auto-restart** | ✅ Yes | ⚠️ Pod restartPolicy applies |
| **Kubernetes version** | 1.28+ | Any |
| **Recommended for** | All deployments | Legacy clusters only |

---

## Configuration

Both examples use AWS Secrets Manager. To use a different backend:

```yaml
env:
- name: VAULTMUX_BACKEND
  value: "gcpsecrets"  # or "azurekeyvault"
- name: GCP_PROJECT_ID  # For GCP
  value: "my-project"
```

See [main README](../../README.md) for full backend configuration options.

---

## Cloud IAM Integration

For namespace isolation, configure cloud IAM per namespace:

**AWS (IRSA):**
```yaml
serviceAccountName: app-sa
# Annotate service account with IAM role
```

**GCP (Workload Identity):**
```yaml
serviceAccountName: app-sa
# Annotate service account with GSA email
```

**Azure (Managed Identity):**
```yaml
# Configure pod identity per namespace
```

See [docs/SIDECAR_RBAC.md](../../docs/SIDECAR_RBAC.md) for complete setup instructions.
