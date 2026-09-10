# Status and conditions

Every managed CRD exposes a `Ready` condition and `status.observedGeneration`. A successful reconciliation sets `Ready=True` and records the generation represented by that observation. A dependency, credential, validation, or Infisical API problem sets `Ready=False` with a reason and a human-readable message.

Inspect the condition and remote identifier together:

```sh
kubectl -n NAMESPACE get infisicalprojects.infisical.infisical-operator.io NAME \
  -o jsonpath='{.status.conditions[?(@.type=="Ready")]}{"\n"}'
kubectl -n NAMESPACE describe infisicalproject NAME
```

Managed resources record only non-secret observed state: identifiers, names, slugs, policy configuration, and whether optional credentials are present. Tokens, CA material, and token-review JWTs are never written to status.

The six CRDs are compatible with generic kstatus consumers through their `Ready` condition and observed generation. They do not currently emit kstatus’s conventional `Reconciling` and `Stalled` conditions, so consumers should use the condition reason and message for detailed failure classification. See the [verification guide](../verification.md) for the evidence boundary.
