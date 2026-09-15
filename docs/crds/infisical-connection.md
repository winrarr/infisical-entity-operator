# InfisicalConnection

`InfisicalConnection` defines the Infisical API endpoint and the same-namespace Secret containing a bearer token. Other operator resources reference it by name and wait until it is ready.

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

The Secret is intentionally the connection boundary, not a Universal Auth configuration. Universal Auth client ID and client secret credentials are exchanged with Infisical for a short-lived bearer token; this resource consumes that resulting token. A future auth-method resource could manage that exchange and write the token to a Secret, but it would remain separate from `InfisicalConnection`.

The status contains only the `Ready` condition and `observedGeneration`. See the [generated schema](../reference/api.md#infisicalconnection) and [security guidance](../reference/security.md).
