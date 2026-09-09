# Security

The operator is a control-plane integration. It does not synchronize Infisical secret values into Kubernetes.

## Credentials and references

`InfisicalConnection.spec.authSecretRef` points to a bearer token in a Secret in the same namespace. All other references are also same-namespace names. The manager reads Secrets through Kubernetes RBAC, and the connection client sends the token only in authenticated Infisical API requests.

`InfisicalKubernetesAuth` may reference a CA certificate Secret and a token-reviewer JWT Secret. The reconciler reports only presence flags in status (`hasCACertificate` and `hasTokenReviewerJWT`), never the data itself.

## Kubernetes permissions

The default chart uses a cluster-wide manager because the operator watches namespaced custom resources across the cluster. Its ClusterRole can read Secrets and manage these CRDs in all namespaces. This is a deliberate single-controller deployment choice, not a multi-tenant isolation claim. Multi-tenancy remains an explicit evaluation item in the [backlog](../../development/project/backlog.md).

Run namespace-scoped deployments and narrower RBAC only after completing that evaluation. Do not introduce cross-namespace references or shared bearer-token Secrets as a shortcut.

## Network and runtime posture

The chart runs the manager as non-root with a read-only root filesystem, no privilege escalation, and all capabilities dropped. Network egress should be limited to the configured Infisical API and required Kubernetes API access. The [local Kind workflow](../../development/operations/local-kind.md) demonstrates this with Cilium.

Avoid placing tokens in manifests committed to Git, controller logs, events, status, generated documentation, or support bundles.
