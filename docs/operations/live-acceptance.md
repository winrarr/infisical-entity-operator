# Live acceptance

`make live-e2e` is the focused acceptance path for Infisical features that the disposable standalone chart cannot prove. It runs the operator locally against the current Kubernetes context and requires a separately provisioned Infisical environment with custom project roles enabled. A licensed self-hosted instance or an Infisical Cloud organization with the required plan is suitable; the default local standalone chart is not.

The test creates and removes a project, machine identity, custom project role, and Kubernetes Auth method. It then logs in with an allowed service account and verifies that a disallowed service account is rejected.

## Prerequisites

- `kubectl` is configured for a disposable or dedicated Kubernetes cluster where the test may create a temporary namespace and a `system:auth-delegator` binding.
- `INFISICAL_E2E_HOST_API` points to the Infisical API endpoint, including the `/api` suffix.
- `INFISICAL_E2E_TOKEN` is a dedicated, short-lived Infisical token with permission to create the test resources in one organization.
- `INFISICAL_E2E_KUBERNETES_HOST` is a Kubernetes API URL that refers to the same cluster as the current kubeconfig and is reachable from Infisical. The Infisical server must be able to perform TokenReview requests against it.
- `jq` and `curl` are installed.

The script reads the Kubernetes CA from the current kubeconfig. Set `INFISICAL_E2E_KUBERNETES_CA_FILE` when the CA is stored separately. The reviewer service-account token is generated inside the cluster and removed during cleanup.

Infisical’s [Kubernetes Auth documentation](https://infisical.com/docs/documentation/platform/identities/kubernetes-auth) describes the reviewer-token and `system:auth-delegator` requirements. Cluster-local Kubernetes URLs may be rejected by Infisical’s URL validation, so use a routable endpoint accepted by the provisioned server and its network controls.

## Run locally

Set the required values from a password manager or CI secret store without committing or printing them:

```sh
export INFISICAL_E2E_HOST_API='https://infisical.example.com/api'
export INFISICAL_E2E_TOKEN='use-a-dedicated-acceptance-token'
export INFISICAL_E2E_KUBERNETES_HOST='https://kubernetes.example.com:6443'
make live-e2e
```

The target installs the committed CRDs, starts the current manager binary against the active kubeconfig, runs the acceptance script, and cleans up only the temporary namespace and reviewer binding. It does not uninstall shared CRDs or change the provisioned Infisical deployment.

## GitHub Actions

The manually triggered `Live Acceptance` workflow uses the same target. Configure these repository secrets before dispatching it:

- `INFISICAL_E2E_KUBECONFIG_B64`: base64-encoded kubeconfig for the dedicated test cluster;
- `INFISICAL_E2E_HOST_API`: the provisioned Infisical API endpoint;
- `INFISICAL_E2E_TOKEN`: the dedicated acceptance token.

The workflow input `kubernetes_host` must be reachable from the Infisical server and correspond to the cluster in `INFISICAL_E2E_KUBECONFIG_B64`. The runner must also be able to reach both the Kubernetes API and Infisical API. Use a self-hosted runner or private network path when those endpoints are not public.

The workflow stores the kubeconfig only in the runner’s temporary directory. The acceptance script does not print Infisical credentials, Kubernetes tokens, or login response bodies.
