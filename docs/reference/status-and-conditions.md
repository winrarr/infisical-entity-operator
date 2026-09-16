# Status and conditions

Every managed CRD exposes `Ready`, `Reconciling`, and `Stalled` conditions together with `status.observedGeneration`. A successful reconciliation sets `Ready=True`, `Reconciling=False`, and `Stalled=False`, and records the generation represented by that observation.

When reconciliation is waiting for a dependency or retrying after an external error, the operator sets `Ready=False`, `Reconciling=True`, and `Stalled=False`. When the resource has invalid configuration, an ownership mismatch, or a creation policy that forbids the required operation, it sets `Ready=False`, `Reconciling=False`, and `Stalled=True`. The `Ready` reason and message identify the specific dependency or failure.

Inspect the condition and remote identifier together:

```sh
kubectl -n NAMESPACE get infisicalprojects.infisical.infisical-operator.io NAME \
  -o jsonpath='{.status.conditions[?(@.type=="Ready")]}{"\n"}'
kubectl -n NAMESPACE describe infisicalproject NAME
```

Managed resources record only non-secret observed state: identifiers, names, slugs, policy configuration, and whether optional credentials are present. Tokens, CA material, and token-review JWTs are never written to status.

All seven CRDs emit the conventional kstatus conditions, so generic status tooling can distinguish successful, progressing, and stalled resources without relying on a particular controller. See the [verification guide](../verification.md) for the evidence boundary.

## Validation timing

The Kubernetes API server rejects structural and cross-field errors that are expressible in the CRD schema, including unsupported enum values, empty required lists, immutable references, invalid identity scope combinations, organization IDs paired with `creationPolicy: Create`, and inconsistent Kubernetes Auth TLS settings. Controller-side validation provides the same protection for fake clients and runtime values such as URL parsing, timeouts, IP/CIDR syntax, and free-tier-only modes. Missing referenced resources remain dependency conditions because they may become available later.
