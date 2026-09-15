# Security

The operator is a control-plane integration. It does not synchronize Infisical secret values into Kubernetes.

## Credentials and references

`InfisicalConnection.spec.authSecretRef` points to a bearer token in a Secret in the same namespace. All other references are also same-namespace names. The manager reads Secrets through Kubernetes RBAC, and the connection client sends the token only in authenticated Infisical API requests.

`InfisicalKubernetesAuth` may reference a CA certificate Secret and a token-reviewer JWT Secret. The reconciler reports only presence flags in status (`hasCACertificate` and `hasTokenReviewerJWT`), never the data itself.

## Kubernetes permissions

The default chart uses a cluster-wide manager because the operator watches namespaced custom resources across the cluster. Its ClusterRole can read Secrets and manage these CRDs in all namespaces. This is a deliberate trusted-platform deployment choice, not a Kubernetes tenant-isolation claim.

For tenant principals, the platform creates one top-level Infisical organization per tenant and an organization-scoped machine identity inside it. Set `organizationRole: admin` when the tenant must create its own projects; the identity can then create any project in that organization but cannot access another organization. The operator grants that creating identity project-admin membership on organization-bound projects so it can manage their children. `InfisicalProject.spec.organizationRef` and `InfisicalIdentity.spec.organizationRef` make the intended boundary explicit, and the controllers reject mismatched observed organization IDs. This is still not a Kubernetes admission boundary: Kubernetes RBAC, Capsule, Kyverno, or the per-vCluster deployment boundary must restrict which CRs a tenant can submit.

The recommended vCluster model is one operator deployment per tenant cluster, using a machine-identity credential created by the trusted platform installation. The tenant operator’s Kubernetes RBAC and Infisical roles are the enforcement boundary. A shared cluster-wide operator should be treated as trusted because its cache and Secret permissions span namespaces.

Run namespace-scoped deployments and narrower RBAC only after completing that evaluation. Do not introduce cross-namespace references or shared bearer-token Secrets as a shortcut.

## Network and runtime posture

The chart runs the manager as non-root with a read-only root filesystem, no privilege escalation, and all capabilities dropped. Network egress should be limited to the configured Infisical API and required Kubernetes API access. The [local Kind workflow](../operations/local-kind.md) demonstrates this with Cilium.

Avoid placing tokens in manifests committed to Git, controller logs, events, status, generated documentation, or support bundles.
