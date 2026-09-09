# Project and identity

This example assumes an `InfisicalConnection` named `infisical` and a bearer-token Secret already exist in the target namespace.

```yaml
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalProject
metadata:
  name: payments
spec:
  connectionRef:
    name: infisical
  projectName: payments
  slug: payments-platform
---
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalEnvironment
metadata:
  name: payments-production
spec:
  connectionRef:
    name: infisical
  projectRef:
    name: payments
  environmentName: Production
  slug: production
---
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalProjectRole
metadata:
  name: payments-secret-reader
spec:
  connectionRef:
    name: infisical
  projectRef:
    name: payments
  roleName: Payments Secret Reader
  slug: payments-secret-reader
  permissions:
    - subject: secrets
      action: [readValue]
      conditions:
        environment:
          $eq: production
---
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalIdentity
metadata:
  name: payments-workload
spec:
  connectionRef:
    name: infisical
  projectRef:
    name: payments
  identityName: payments-workload
  roleSlugs:
    - payments-secret-reader
```

Apply the document and watch the dependency graph converge:

```sh
kubectl apply -n NAMESPACE -f project-and-identity.yaml
kubectl get -n NAMESPACE infisicalproject,infisicalenvironment,infisicalprojectrole,infisicalidentity
```
