# InfisicalKubernetesAuth

`InfisicalKubernetesAuth` attaches a Kubernetes Auth method to an `InfisicalIdentity`.

Use `spec.templateID` when the Kubernetes Auth settings are managed by an Infisical Kubernetes Auth template. When it is set, omit the host, CA Secret, token reviewer Secret, review mode, gateway, and audience fields; those values come from Infisical’s template. The template ID is immutable in this resource, so delete and recreate the resource to change it.

```yaml
apiVersion: infisical.infisical-operator.io/v1alpha1
kind: InfisicalKubernetesAuth
metadata:
  name: payments-workload-kubernetes
spec:
  connectionRef:
    name: infisical
  identityRef:
    name: payments-workload
  kubernetesHost: https://kubernetes.default.svc
  allowedNamespaces:
    - payments
  allowedNames:
    - payments-workload
  tokenReviewMode: api
  creationPolicy: CreateOrAdopt
  deletionPolicy: Orphan
```

The resource supports API-server or gateway token review, optional CA and reviewer-token Secret references, audience restrictions, trusted IPs, and access-token TTL/use limits. Secret references are same-namespace and only their presence is reported in status. When a template supplies the reviewer JWT, status reports that fact without exposing the JWT.

The connection and identity references are immutable. The configured Kubernetes API URL, network path, CA, reviewer token, and Infisical plan must all permit the selected token-review mode. See the [generated schema](../reference/api.md#infisicalkubernetesauth) and [local operations guide](../operations/local-kind.md).
