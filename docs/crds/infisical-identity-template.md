# InfisicalIdentityTemplate

`InfisicalIdentityTemplate` manages an organization-owned Infisical identity authentication template. It supports the `ldap`, `kubernetes`, and `oidc` variants exposed by the current Infisical API contract.

The organization must already be ready. Authentication method configuration is immutable; delete and recreate the resource to change it. Secret-backed values remain in same-namespace Secrets and status reports only whether sensitive values are present.

For example, a Kubernetes template can be declared like this:

```yaml
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalIdentityTemplate
metadata:
  name: cluster-auth
  namespace: infisical-demo
spec:
  connectionRef:
    name: infisical
  organizationRef:
    name: tenant
  authMethod: kubernetes
  kubernetes:
    kubernetesHost: https://kubernetes.default.svc
    caCertSecretRef:
      name: kubernetes-ca
      key: ca.crt
    tokenReviewerJWTSecretRef:
      name: kubernetes-reviewer
      key: token
    tokenReviewMode: api
    verifyTLSCertificate: true
    allowedAudience: infisical
```

`InfisicalKubernetesAuth` can then reference this resource through `spec.templateRef`; its allowlists and token-lifetime settings remain on the auth resource. See the [generated schema](../reference/api.md#infisicalidentitytemplate) for the LDAP and OIDC fields.
