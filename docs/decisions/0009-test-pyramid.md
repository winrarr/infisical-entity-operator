# 0009: Keep live tests narrow and lifecycle tests local

Status: accepted

Date: 2026-09-16

## Decision

Keep the default Kind workflow as a small smoke test covering chart installation, manager startup, network-policy application, Infisical connectivity, and one representative project lifecycle.

Exercise controller lifecycle behavior—adoption, drift correction, deletion, credential rotation, and recovery—in Go tests using fake Kubernetes clients and `httptest` Infisical APIs. Keep Kubernetes Auth, vCluster, Capsule, Kyverno, Cilium, and other environment-specific checks as explicit opt-in scenarios.

## Rationale

Raw shell workflows that provision many resources are slower and more sensitive to Kubernetes, chart, and API changes than focused Go tests. The default CI signal should remain fast and stable while still proving that the packaged operator can start and reconcile against a real Infisical instance.

## Consequences

The default smoke test is not a complete lifecycle or policy suite. New lifecycle behavior should normally receive Go coverage first; a live scenario should be added only when it proves an integration boundary that local tests cannot represent. Specialized shell workflows remain useful for platform integrations, but they are not part of the default pull-request path.
