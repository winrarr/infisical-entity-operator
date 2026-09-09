# Troubleshooting

Start with the resource’s condition, then inspect events and the manager logs:

```sh
kubectl -n NAMESPACE get infisicalconnection,infisicalproject,infisicalenvironment,infisicalprojectrole,infisicalidentity,infisicalkubernetesauth
kubectl -n NAMESPACE describe KIND NAME
kubectl -n infisical-entity-operator-system logs deployment/infisical-entity-operator-infisical-entity-operator
```

The exact Deployment name changes when the Helm release or chart name is overridden. Find it with `kubectl -n infisical-entity-operator-system get deployments`.

## Common conditions

| Symptom | Check |
| --- | --- |
| `Ready=False` mentions a dependency | Create the referenced connection or project in the same namespace and wait for its `Ready=True` condition. |
| Authentication or Secret errors | Confirm the Secret exists in the resource namespace, the key is correct, and the bearer token is valid. Never paste the token into logs or issue reports. |
| Remote entity is not found | Use `Create` for a new entity, or confirm the expected name/slug and use `Adopt` or `CreateOrAdopt` intentionally. |
| Changes are rejected by the API server | Immutable references and identity fields require delete-and-recreate. Review the validation message before changing deletion policy. |
| API calls time out | Check DNS, TLS, the configured `hostAPI`, proxy settings, and egress NetworkPolicies. |
| Delete is waiting | Inspect delete protection, child resources, the finalizer, and the `Ready` reason. `Orphan` avoids remote deletion when that is the desired outcome. |

For a reproducible end-to-end environment, use the [local Kind operations](../../development/operations/local-kind.md) guide. The [verification guide](../../development/project/verification.md) describes which local tests require a live Infisical plan or endpoint capability.
