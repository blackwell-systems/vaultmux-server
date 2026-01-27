# Sidecar Pattern with Namespace Isolation

This guide shows how to deploy vaultmux-server as a sidecar with namespace isolation using cloud provider IAM.

---

## Why the Sidecar Pattern Exists

Kubernetes does not provide runtime secret isolation by default. If you use Kubernetes Secrets or CRDs, secrets ultimately live in **etcd** and are governed by **cluster RBAC**, not by cloud IAM.

The sidecar pattern solves a different problem:

> **Runtime access control enforced by the cloud provider, not the cluster.**

By running vaultmux-server as a sidecar:
- Each namespace gets its **own identity**
- Each identity maps to **cloud IAM**
- The cloud provider becomes the **source of truth**
- Secrets are **never stored in etcd**
- Test pods literally *cannot* access prod secrets, even if misconfigured

This gives you **hard isolation at the cloud boundary**, not just "best effort" isolation inside Kubernetes.

**The trust model:**
- Don't trust: Kubernetes RBAC (can be misconfigured)
- Don't trust: Network policies (can have holes)
- Trust: Cloud provider IAM (enforced at API level, outside cluster)

---

## Overview

The sidecar pattern provides namespace-level secret isolation by leveraging Kubernetes service accounts mapped to cloud IAM identities. Each namespace gets its own vaultmux-server pod with separate IAM permissions.

**Architecture:**
```
test namespace:
  test-app pod
    ├── test-app container
    └── vaultmux-server sidecar
         └── Uses test-sa service account
              └── Maps to test-secrets-role IAM role
                   └── Can only access test/* secrets

prod namespace:
  prod-app pod
    ├── prod-app container
    └── vaultmux-server sidecar
         └── Uses prod-sa service account
              └── Maps to prod-secrets-role IAM role
                   └── Can only access prod/* secrets
```

Cloud provider enforces the boundary: `test` pods cannot access `prod` secrets.

---

## AWS: IAM Roles for Service Accounts (IRSA)

### Prerequisites

- EKS cluster with OIDC provider enabled
- AWS CLI configured
- kubectl access to cluster

### Step 1: Enable OIDC Provider (One-Time)

```bash
# Get OIDC provider URL
aws eks describe-cluster --name my-cluster --query "cluster.identity.oidc.issuer" --output text

# Enable OIDC provider
eksctl utils associate-iam-oidc-provider --cluster my-cluster --approve
```

### Step 2: Create IAM Policies

**Test namespace policy** (`test-secrets-policy.json`):
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue",
        "secretsmanager:DescribeSecret"
      ],
      "Resource": "arn:aws:secretsmanager:us-east-1:123456789012:secret:test/*"
    }
  ]
}
```

**Prod namespace policy** (`prod-secrets-policy.json`):
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue",
        "secretsmanager:DescribeSecret"
      ],
      "Resource": "arn:aws:secretsmanager:us-east-1:123456789012:secret:prod/*"
    }
  ]
}
```

```bash
# Create policies
aws iam create-policy --policy-name test-secrets-policy --policy-document file://test-secrets-policy.json
aws iam create-policy --policy-name prod-secrets-policy --policy-document file://prod-secrets-policy.json
```

### Step 3: Create IAM Roles

**Test namespace role:**
```bash
eksctl create iamserviceaccount \
  --name test-vaultmux-sa \
  --namespace test \
  --cluster my-cluster \
  --attach-policy-arn arn:aws:iam::123456789012:policy/test-secrets-policy \
  --approve
```

**Prod namespace role:**
```bash
eksctl create iamserviceaccount \
  --name prod-vaultmux-sa \
  --namespace prod \
  --cluster my-cluster \
  --attach-policy-arn arn:aws:iam::123456789012:policy/prod-secrets-policy \
  --approve
```

### Step 4: Deploy Sidecar

**Test namespace deployment:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
  namespace: test
spec:
  template:
    spec:
      serviceAccountName: test-vaultmux-sa  # Uses test IAM role
      containers:
        - name: app
          image: myapp:latest
          env:
            - name: VAULTMUX_URL
              value: "http://localhost:8080"
        
        - name: vaultmux-server
          image: ghcr.io/blackwell-systems/vaultmux-server:v0.1.0
          ports:
            - containerPort: 8080
          env:
            - name: VAULTMUX_BACKEND
              value: awssecrets
            - name: AWS_REGION
              value: us-east-1
```

**Prod namespace deployment:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prod-app
  namespace: prod
spec:
  template:
    spec:
      serviceAccountName: prod-vaultmux-sa  # Uses prod IAM role
      containers:
        - name: app
          image: myapp:latest
          env:
            - name: VAULTMUX_URL
              value: "http://localhost:8080"
        
        - name: vaultmux-server
          image: ghcr.io/blackwell-systems/vaultmux-server:v0.1.0
          ports:
            - containerPort: 8080
          env:
            - name: VAULTMUX_BACKEND
              value: awssecrets
            - name: AWS_REGION
              value: us-east-1
```

### Step 5: Test Isolation

**From test namespace pod:**
```bash
# This works (test pod can access test/* secrets)
kubectl exec -n test -it test-app-xxx -c app -- \
  curl http://localhost:8080/v1/secrets/test/api-key

# This fails (test pod CANNOT access prod/* secrets)
kubectl exec -n test -it test-app-xxx -c app -- \
  curl http://localhost:8080/v1/secrets/prod/api-key
# Error: AccessDeniedException
```

**From prod namespace pod:**
```bash
# This works (prod pod can access prod/* secrets)
kubectl exec -n prod -it prod-app-xxx -c app -- \
  curl http://localhost:8080/v1/secrets/prod/api-key

# This fails (prod pod CANNOT access test/* secrets)
kubectl exec -n prod -it prod-app-xxx -c app -- \
  curl http://localhost:8080/v1/secrets/test/api-key
# Error: AccessDeniedException
```

### Troubleshooting AWS IRSA

**Problem: "Error retrieving secret: AccessDeniedException"**

Check service account annotation:
```bash
kubectl get sa test-vaultmux-sa -n test -o yaml | grep eks.amazonaws.com/role-arn
```

Should show:
```yaml
annotations:
  eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/test-vaultmux-role
```

**Problem: Role exists but still denied**

Check trust relationship on IAM role:
```bash
aws iam get-role --role-name test-vaultmux-role --query 'Role.AssumeRolePolicyDocument'
```

Should include OIDC provider:
```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {
      "Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-east-1.amazonaws.com/id/EXAMPLED539D4633E53DE1B71EXAMPLE"
    },
    "Action": "sts:AssumeRoleWithWebIdentity",
    "Condition": {
      "StringEquals": {
        "oidc.eks.us-east-1.amazonaws.com/id/EXAMPLED539D4633E53DE1B71EXAMPLE:sub": "system:serviceaccount:test:test-vaultmux-sa"
      }
    }
  }]
}
```

---

## GCP: Workload Identity

### Prerequisites

- GKE cluster with Workload Identity enabled
- gcloud CLI configured
- kubectl access to cluster

### Step 1: Enable Workload Identity (One-Time)

```bash
# Enable on existing cluster
gcloud container clusters update my-cluster \
  --workload-pool=my-project.svc.id.goog

# Enable on node pool
gcloud container node-pools update default-pool \
  --cluster=my-cluster \
  --workload-metadata=GKE_METADATA
```

### Step 2: Create GCP Service Accounts

```bash
# Test namespace service account
gcloud iam service-accounts create test-vaultmux-sa \
  --display-name="Test namespace vaultmux-server"

# Prod namespace service account
gcloud iam service-accounts create prod-vaultmux-sa \
  --display-name="Prod namespace vaultmux-server"
```

### Step 3: Grant Secret Manager Permissions

```bash
# Test SA can only access test/* secrets
gcloud secrets add-iam-policy-binding test-api-key \
  --member="serviceAccount:test-vaultmux-sa@my-project.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"

# Prod SA can only access prod/* secrets
gcloud secrets add-iam-policy-binding prod-api-key \
  --member="serviceAccount:prod-vaultmux-sa@my-project.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

**Alternative: Use secret name prefixes**

Create custom IAM roles with resource conditions:
```bash
# Custom role for test secrets only
gcloud iam roles create testSecretsAccessor \
  --project=my-project \
  --title="Test Secrets Accessor" \
  --permissions=secretmanager.versions.access,secretmanager.secrets.get \
  --stage=GA

# Bind with condition
gcloud projects add-iam-policy-binding my-project \
  --member="serviceAccount:test-vaultmux-sa@my-project.iam.gserviceaccount.com" \
  --role="projects/my-project/roles/testSecretsAccessor" \
  --condition='resource.name.startsWith("projects/my-project/secrets/test-")'
```

### Step 4: Bind Kubernetes SA to GCP SA

```bash
# Test namespace
gcloud iam service-accounts add-iam-policy-binding \
  test-vaultmux-sa@my-project.iam.gserviceaccount.com \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:my-project.svc.id.goog[test/test-vaultmux-sa]"

# Prod namespace
gcloud iam service-accounts add-iam-policy-binding \
  prod-vaultmux-sa@my-project.iam.gserviceaccount.com \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:my-project.svc.id.goog[prod/prod-vaultmux-sa]"
```

### Step 5: Create Kubernetes Service Accounts

**Test namespace:**
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: test-vaultmux-sa
  namespace: test
  annotations:
    iam.gke.io/gcp-service-account: test-vaultmux-sa@my-project.iam.gserviceaccount.com
```

**Prod namespace:**
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: prod-vaultmux-sa
  namespace: prod
  annotations:
    iam.gke.io/gcp-service-account: prod-vaultmux-sa@my-project.iam.gserviceaccount.com
```

### Step 6: Deploy Sidecar

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
  namespace: test
spec:
  template:
    spec:
      serviceAccountName: test-vaultmux-sa
      containers:
        - name: app
          image: myapp:latest
        
        - name: vaultmux-server
          image: ghcr.io/blackwell-systems/vaultmux-server:v0.1.0
          env:
            - name: VAULTMUX_BACKEND
              value: gcpsecrets
            - name: GCP_PROJECT_ID
              value: my-project
```

### Troubleshooting GCP Workload Identity

**Problem: "Error: Permission denied on secret"**

Check annotation:
```bash
kubectl get sa test-vaultmux-sa -n test -o yaml | grep iam.gke.io
```

Check binding:
```bash
gcloud iam service-accounts get-iam-policy \
  test-vaultmux-sa@my-project.iam.gserviceaccount.com
```

Should show `roles/iam.workloadIdentityUser` binding.

---

## Azure: Managed Identity

### Prerequisites

- AKS cluster with managed identity enabled
- Azure CLI configured
- kubectl access to cluster

### Step 1: Enable AAD Pod Identity (One-Time)

```bash
# Install AAD Pod Identity
kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml
```

Or use Azure Workload Identity (newer):
```bash
# Enable workload identity on cluster
az aks update \
  --resource-group myResourceGroup \
  --name myAKSCluster \
  --enable-oidc-issuer \
  --enable-workload-identity
```

### Step 2: Create Managed Identities

```bash
# Test namespace identity
az identity create \
  --resource-group myResourceGroup \
  --name test-vaultmux-identity

# Prod namespace identity
az identity create \
  --resource-group myResourceGroup \
  --name prod-vaultmux-identity
```

### Step 3: Grant Key Vault Permissions

```bash
# Get identity principal IDs
TEST_PRINCIPAL_ID=$(az identity show --resource-group myResourceGroup --name test-vaultmux-identity --query principalId -o tsv)
PROD_PRINCIPAL_ID=$(az identity show --resource-group myResourceGroup --name prod-vaultmux-identity --query principalId -o tsv)

# Test identity can only access test-keyvault
az keyvault set-policy \
  --name test-keyvault \
  --object-id $TEST_PRINCIPAL_ID \
  --secret-permissions get list

# Prod identity can only access prod-keyvault
az keyvault set-policy \
  --name prod-keyvault \
  --object-id $PROD_PRINCIPAL_ID \
  --secret-permissions get list
```

### Step 4: Create AzureIdentity Resources

**Test namespace:**
```yaml
apiVersion: aadpodidentity.k8s.io/v1
kind: AzureIdentity
metadata:
  name: test-vaultmux-identity
  namespace: test
spec:
  type: 0
  resourceID: /subscriptions/<subscription-id>/resourcegroups/myResourceGroup/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test-vaultmux-identity
  clientID: <test-identity-client-id>
---
apiVersion: aadpodidentity.k8s.io/v1
kind: AzureIdentityBinding
metadata:
  name: test-vaultmux-binding
  namespace: test
spec:
  azureIdentity: test-vaultmux-identity
  selector: test-vaultmux
```

**Prod namespace:**
```yaml
apiVersion: aadpodidentity.k8s.io/v1
kind: AzureIdentity
metadata:
  name: prod-vaultmux-identity
  namespace: prod
spec:
  type: 0
  resourceID: /subscriptions/<subscription-id>/resourcegroups/myResourceGroup/providers/Microsoft.ManagedIdentity/userAssignedIdentities/prod-vaultmux-identity
  clientID: <prod-identity-client-id>
---
apiVersion: aadpodidentity.k8s.io/v1
kind: AzureIdentityBinding
metadata:
  name: prod-vaultmux-binding
  namespace: prod
spec:
  azureIdentity: prod-vaultmux-identity
  selector: prod-vaultmux
```

### Step 5: Deploy Sidecar

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
  namespace: test
spec:
  template:
    metadata:
      labels:
        aadpodidbinding: test-vaultmux  # Matches AzureIdentityBinding selector
    spec:
      containers:
        - name: app
          image: myapp:latest
        
        - name: vaultmux-server
          image: ghcr.io/blackwell-systems/vaultmux-server:v0.1.0
          env:
            - name: VAULTMUX_BACKEND
              value: azurekeyvault
            - name: AZURE_KEYVAULT_URL
              value: https://test-keyvault.vault.azure.net
```

### Troubleshooting Azure Managed Identity

**Problem: "Error: Authentication failed"**

Check identity binding:
```bash
kubectl get azureidentitybinding -n test
kubectl get azureidentity -n test
```

Check pod labels match selector:
```bash
kubectl get pod -n test -l aadpodidbinding=test-vaultmux
```

---

## Best Practices

### 1. Principle of Least Privilege

Grant only the minimum permissions needed:
```json
{
  "Effect": "Allow",
  "Action": [
    "secretsmanager:GetSecretValue"  // Only read, not write/delete
  ],
  "Resource": "arn:aws:secretsmanager:*:*:secret:test/*"  // Only test prefix
}
```

### 2. Use Secret Name Prefixes

Organize secrets by namespace:
```
test/api-key
test/database-password
prod/api-key
prod/database-password
```

IAM policies can match prefixes for clean isolation.

### 3. Audit Access

Enable cloud provider audit logs:
- AWS: CloudTrail
- GCP: Cloud Audit Logs
- Azure: Activity Log

Monitor for unauthorized secret access attempts.

### 4. Rotate Secrets Regularly

Set up automatic rotation at the backend level:
- AWS Secrets Manager: automatic rotation
- GCP Secret Manager: version management
- Azure Key Vault: expiration policies

vaultmux-server always fetches the latest version.

### 5. Test Isolation

After setup, verify test namespace cannot access prod secrets:
```bash
# Should fail with 403 Forbidden or AccessDenied
kubectl exec -n test test-app-xxx -c app -- \
  curl http://localhost:8080/v1/secrets/prod/api-key
```

---

## Comparison: Sidecar vs Cluster Service

| Aspect | Sidecar + IAM | Cluster Service + HTTP RBAC |
|--------|---------------|----------------------------|
| **Namespace isolation** | Cloud IAM (works today) | HTTP-level auth (roadmap) |
| **Setup complexity** | Medium (IAM roles per namespace) | Low (single deployment) |
| **Resource usage** | High (one pod per app) | Low (2-3 replicas total) |
| **Latency** | ~1ms (localhost) | ~5-10ms (network) |
| **Security boundary** | Cloud provider IAM | vaultmux-server RBAC |
| **Best for** | Production multi-tenant | Dev/staging, single-tenant |

---

## FAQ

**Q: Can I use the same IAM role for multiple namespaces?**

No - that defeats the isolation. Each namespace needs its own IAM identity with scoped permissions.

**Q: What if I have 20 namespaces?**

You need 20 IAM roles with 20 different policies. Consider using infrastructure-as-code (Terraform, Pulumi) to automate IAM role creation.

**Q: Does this work with Helm?**

Yes, but you need to create service accounts and IAM bindings separately (Helm can't manage cloud IAM resources). The Helm chart can reference existing service accounts:

```yaml
serviceAccount:
  create: false
  name: test-vaultmux-sa  # Pre-created with IAM binding
```

**Q: Can I mix sidecar and cluster service?**

Yes - use sidecar for prod (isolation), cluster service for dev/test (convenience).

**Q: What about secret caching?**

vaultmux-server doesn't cache by default (every request hits backend). For caching, consider:
- Application-level caching
- Backend-level caching (some cloud providers cache automatically)
- Future vaultmux-server feature (on roadmap)

---

## Related Documentation

- [AWS IRSA Documentation](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html)
- [GCP Workload Identity Documentation](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)
- [Azure Workload Identity Documentation](https://learn.microsoft.com/en-us/azure/aks/workload-identity-overview)
- [vaultmux-server ROADMAP](../ROADMAP.md) - HTTP-level RBAC for cluster service pattern
