# InfisicalConnection

`InfisicalConnection` defines the Infisical API endpoint and the same-namespace credentials used by other resources. Other operator resources reference it by name and wait until it is ready.

```yaml
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalConnection
metadata:
  name: infisical
spec:
  hostAPI: https://app.infisical.com/api
  authSecretRef:
    name: infisical-token
    key: token
  requestTimeout: 30s
```

`hostAPI` defaults to Infisical Cloud and must include the `/api` path. Use the endpoint’s API base URL for self-hosted Infisical. The Secret must remain in the same namespace; the operator does not support cross-namespace Secret references.

For tenant-specific credentials, use `InfisicalUniversalAuth` to publish a Secret and reference it instead:

```yaml
spec:
  universalAuth:
    secretRef:
      name: tenant-credentials
    organizationSlug: tenant-org
```

`universalAuth.secretRef` reads `clientId` and `clientSecret` by default. InfisicalConnection exchanges them for a short-lived bearer token and refreshes it automatically. Configure `clientIDKey` or `clientSecretKey` for custom Secret keys. Exactly one of `authSecretRef` and `universalAuth` must be configured.

The status contains only the `Ready` condition and `observedGeneration`. See the [generated schema](../reference/api.md#infisicalconnection) and [security guidance](../reference/security.md).
