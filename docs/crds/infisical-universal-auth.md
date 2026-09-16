# InfisicalUniversalAuth

`InfisicalUniversalAuth` attaches Universal Auth to an `InfisicalIdentity`, creates a remote client secret, and publishes the one-time credential to a same-namespace Kubernetes Secret. The client secret value is never written to custom-resource status, logs, or events.

The referenced identity must be ready. The output Secret contains `clientId` and `clientSecret` keys by default; configure `clientIDKey` and `clientSecretKey` when another key layout is required.

```yaml
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalUniversalAuth
metadata:
  name: tenant-credentials
  namespace: infisical-demo
spec:
  connectionRef:
    name: platform-infisical
  identityRef:
    name: tenant-identity
  clientSecret:
    secretRef:
      name: tenant-credentials
    description: tenant operator credential
    rotationNonce: initial
  deletionPolicy: Delete
```

Rotate explicitly by changing `spec.clientSecret.rotationNonce`. The operator publishes the replacement before revoking the previous remote credential. If the output Secret is deleted or the remote client secret is revoked, the operator creates and publishes a replacement. The generated Secret is owned by this resource.

Use the generated Secret from an `InfisicalConnection` like this:

```yaml
spec:
  universalAuth:
    secretRef:
      name: tenant-credentials
    organizationSlug: tenant-org
```

The connection exchanges the client credentials for short-lived bearer tokens and refreshes them automatically. The existing `authSecretRef` bearer-token mode remains supported. See the [generated schema](../reference/api.md#infisicaluniversalauth) for configuration and status fields.
