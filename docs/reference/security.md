# Security

The operator is a control-plane integration. It does not synchronize Infisical secret values into Kubernetes.

## Credentials and references

`InfisicalConnection.spec.authSecretRef` points to a bearer token in a Secret in the same namespace. All other references are also same-namespace names. The manager reads Secrets through Kubernetes RBAC, and the connection client sends the token only in authenticated Infisical API requests.

`InfisicalKubernetesAuth` may reference a CA certificate Secret and a token-reviewer JWT Secret. The reconciler reports only presence flags in status (`hasCACertificate` and `hasTokenReviewerJWT`), never the data itself.

## Kubernetes permissions

The default chart uses a cluster-wide manager because the operator watches namespaced custom resources across the cluster. Its ClusterRole can read Secrets and manage these CRDs in all namespaces. This is a deliberate trusted-platform deployment choice, not a Kubernetes tenant-isolation claim.

For tenant principals, `InfisicalIdentity` can create an organization-scoped machine identity with Infisical’s `no-access` organization role and explicit project-role bindings. A binding is accepted only when the referenced project reports the same organization as `organizationRef`. This limits the identity’s intended Infisical scope, but it cannot revoke memberships created outside the listed bindings and does not restrict Kubernetes users from creating arbitrary CRs.

The recommended vCluster model is one operator deployment per tenant cluster, using a machine-identity credential created by the trusted platform installation. The tenant operator’s Kubernetes RBAC and Infisical roles are the enforcement boundary. A shared cluster-wide operator should be treated as trusted because its cache and Secret permissions span namespaces.

Run namespace-scoped deployments and narrower RBAC only after completing that evaluation. Do not introduce cross-namespace references or shared bearer-token Secrets as a shortcut.

## Network and runtime posture

The chart runs the manager as non-root with a read-only root filesystem, no privilege escalation, and all capabilities dropped. Network egress should be limited to the configured Infisical API and required Kubernetes API access. The [local Kind workflow](../operations/local-kind.md) demonstrates this with Cilium.

Avoid placing tokens in manifests committed to Git, controller logs, events, status, generated documentation, or support bundles.
