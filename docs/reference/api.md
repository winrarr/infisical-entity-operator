# API Reference

## Packages
- [infisical.infisical-operator.io/v1alpha1](#infisicalinfisical-operatoriov1alpha1)


## infisical.infisical-operator.io/v1alpha1

Package v1alpha1 contains API Schema definitions for the infisical v1alpha1 API group.

### Resource Types
- [InfisicalConnection](#infisicalconnection)
- [InfisicalEnvironment](#infisicalenvironment)
- [InfisicalIdentity](#infisicalidentity)
- [InfisicalKubernetesAuth](#infisicalkubernetesauth)
- [InfisicalOrganization](#infisicalorganization)
- [InfisicalProject](#infisicalproject)
- [InfisicalUniversalAuth](#infisicaluniversalauth)



#### CreationPolicy

_Underlying type:_ _string_

CreationPolicy controls how an external entity is acquired.

_Validation:_
- Enum: [Create Adopt CreateOrAdopt]

_Appears in:_
- [InfisicalEnvironmentSpec](#infisicalenvironmentspec)
- [InfisicalIdentitySpec](#infisicalidentityspec)
- [InfisicalKubernetesAuthSpec](#infisicalkubernetesauthspec)
- [InfisicalOrganizationSpec](#infisicalorganizationspec)
- [InfisicalProjectSpec](#infisicalprojectspec)
- [InfisicalUniversalAuthSpec](#infisicaluniversalauthspec)

| Field | Description |
| --- | --- |
| `Create` |  |
| `Adopt` |  |
| `CreateOrAdopt` |  |


#### DeletionPolicy

_Underlying type:_ _string_

DeletionPolicy controls what happens to the external entity on deletion.

_Validation:_
- Enum: [Delete Orphan]

_Appears in:_
- [InfisicalEnvironmentSpec](#infisicalenvironmentspec)
- [InfisicalIdentitySpec](#infisicalidentityspec)
- [InfisicalKubernetesAuthSpec](#infisicalkubernetesauthspec)
- [InfisicalOrganizationSpec](#infisicalorganizationspec)
- [InfisicalProjectSpec](#infisicalprojectspec)
- [InfisicalUniversalAuthSpec](#infisicaluniversalauthspec)

| Field | Description |
| --- | --- |
| `Delete` |  |
| `Orphan` |  |


#### IdentityMetadata



IdentityMetadata is a key/value pair attached to an Infisical identity.



_Appears in:_
- [InfisicalIdentitySpec](#infisicalidentityspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `key` _string_ | Key is the metadata key. |  | MinLength: 1 <br /> |
| `value` _string_ | Value is the metadata value. |  |  |


#### IdentityProjectRoleBinding



IdentityProjectRoleBinding declares the permanent project roles for an organization-scoped identity.



_Appears in:_
- [InfisicalIdentitySpec](#infisicalidentityspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `projectRef` _[LocalObjectReference](#localobjectreference)_ | ProjectRef references the project receiving this identity's roles. |  |  |
| `roleSlugs` _string array_ | RoleSlugs is the complete permanent role set for this project membership. |  | MinItems: 1 <br /> |


#### IdentityScope

_Underlying type:_ _string_

IdentityScope selects the Infisical ownership boundary for a machine identity.
Project is the backward-compatible default.

_Validation:_
- Enum: [Project Organization]

_Appears in:_
- [InfisicalIdentitySpec](#infisicalidentityspec)

| Field | Description |
| --- | --- |
| `Project` | IdentityScopeProject creates a project-managed machine identity.<br /> |
| `Organization` | IdentityScopeOrganization creates an organization-managed machine identity.<br /> |


#### InfisicalConnection



InfisicalConnection is the Schema for the infisicalconnections API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `infisical.infisical-operator.io/v1alpha1` | | |
| `kind` _string_ | `InfisicalConnection` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[InfisicalConnectionSpec](#infisicalconnectionspec)_ | spec defines the desired state of InfisicalConnection |  | Required: \{\} <br /> |


#### InfisicalConnectionReference



InfisicalConnectionReference identifies a same-namespace connection.



_Appears in:_
- [InfisicalEnvironmentSpec](#infisicalenvironmentspec)
- [InfisicalIdentitySpec](#infisicalidentityspec)
- [InfisicalKubernetesAuthSpec](#infisicalkubernetesauthspec)
- [InfisicalOrganizationSpec](#infisicalorganizationspec)
- [InfisicalProjectSpec](#infisicalprojectspec)
- [InfisicalUniversalAuthSpec](#infisicaluniversalauthspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the InfisicalConnection name. |  | MinLength: 1 <br /> |


#### InfisicalConnectionSpec



InfisicalConnectionSpec defines the desired state of InfisicalConnection



_Appears in:_
- [InfisicalConnection](#infisicalconnection)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `hostAPI` _string_ | HostAPI is the Infisical API base URL, including the /api path.<br />It defaults to Infisical Cloud. | https://app.infisical.com/api | Pattern: `^https?://` <br />Optional: \{\} <br /> |
| `authSecretRef` _[SecretKeyReference](#secretkeyreference)_ | AuthSecretRef references a Secret containing a bearer token under Key.<br />The Secret must be in the same namespace as this connection. |  | Optional: \{\} <br /> |
| `universalAuth` _[UniversalAuthConnectionSpec](#universalauthconnectionspec)_ | UniversalAuth selects a same-namespace Secret containing an Infisical Universal Auth<br />client ID and client secret. The client secret is exchanged for short-lived bearer tokens. |  | Optional: \{\} <br /> |
| `requestTimeout` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#duration-v1-meta)_ | RequestTimeout bounds each request made to Infisical. | 30s | Optional: \{\} <br /> |


#### InfisicalEnvironment



InfisicalEnvironment is the Schema for the infisicalenvironments API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `infisical.infisical-operator.io/v1alpha1` | | |
| `kind` _string_ | `InfisicalEnvironment` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[InfisicalEnvironmentSpec](#infisicalenvironmentspec)_ | Spec defines the desired state of InfisicalEnvironment. |  | Required: \{\} <br /> |


#### InfisicalEnvironmentSpec



InfisicalEnvironmentSpec defines the desired state of InfisicalEnvironment.



_Appears in:_
- [InfisicalEnvironment](#infisicalenvironment)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[InfisicalConnectionReference](#infisicalconnectionreference)_ | ConnectionRef selects the Infisical API connection. |  |  |
| `projectRef` _[LocalObjectReference](#localobjectreference)_ | ProjectRef references the InfisicalProject that owns this environment. |  |  |
| `environmentName` _string_ | EnvironmentName is the Infisical environment display name. If omitted, metadata.name is used. |  | MaxLength: 255 <br />MinLength: 1 <br />Optional: \{\} <br /> |
| `slug` _string_ | Slug is the stable Infisical environment slug. If omitted, metadata.name is used. |  | MaxLength: 64 <br />MinLength: 1 <br />Optional: \{\} <br /> |
| `position` _integer_ | Position controls the environment order. Lower values appear first. |  | Minimum: 1 <br />Optional: \{\} <br /> |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls whether the operator creates or adopts an environment. | Create | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the external environment is deleted with this resource. | Orphan | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |


#### InfisicalIdentity



InfisicalIdentity is the Schema for the infisicalidentities API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `infisical.infisical-operator.io/v1alpha1` | | |
| `kind` _string_ | `InfisicalIdentity` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[InfisicalIdentitySpec](#infisicalidentityspec)_ | spec defines the desired state of InfisicalIdentity |  | Required: \{\} <br /> |


#### InfisicalIdentitySpec



InfisicalIdentitySpec defines the desired state of InfisicalIdentity



_Appears in:_
- [InfisicalIdentity](#infisicalidentity)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[InfisicalConnectionReference](#infisicalconnectionreference)_ | ConnectionRef selects the Infisical API connection. |  |  |
| `scope` _[IdentityScope](#identityscope)_ | Scope selects whether Infisical manages this identity at project or organization scope.<br />It defaults to Project. Organization-scoped identities are useful as tenant principals;<br />use organizationRole: admin when the tenant should create its own projects. | Project | Enum: [Project Organization] <br />Optional: \{\} <br /> |
| `projectRef` _[LocalObjectReference](#localobjectreference)_ | ProjectRef references the InfisicalProject resource that owns a project-scoped identity. |  | Optional: \{\} <br /> |
| `organizationRef` _[LocalObjectReference](#localobjectreference)_ | OrganizationRef references an InfisicalOrganization resource whose observed organization<br />owns an organization-scoped identity. The project itself need not be listed in projectRoleBindings. |  | Optional: \{\} <br /> |
| `identityName` _string_ | IdentityName is the Infisical identity name. If omitted, metadata.name is used. |  | MinLength: 1 <br />Optional: \{\} <br /> |
| `hasDeleteProtection` _boolean_ | HasDeleteProtection configures Infisical-side delete protection. | false | Optional: \{\} <br /> |
| `metadata` _[IdentityMetadata](#identitymetadata) array_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `roleSlugs` _string array_ | RoleSlugs declares the permanent built-in project role slugs assigned to the identity.<br />When omitted, the existing project membership is not managed. Use no-access<br />explicitly when the identity should have no project permissions. |  | MinItems: 1 <br />Optional: \{\} <br /> |
| `organizationRole` _string_ | OrganizationRole is the built-in Infisical organization role for an organization-scoped identity.<br />Leave it empty to use Infisical's least-privilege no-access role. Project-scoped identities<br />must omit this field. |  | Enum: [admin member no-access] <br />MinLength: 1 <br />Optional: \{\} <br /> |
| `projectRoleBindings` _[IdentityProjectRoleBinding](#identityprojectrolebinding) array_ | ProjectRoleBindings grants an organization-scoped identity roles in selected projects.<br />A binding manages that project's complete permanent role list; omitted projects are left<br />unmanaged. Every referenced project must belong to the organization from organizationRef. |  | Optional: \{\} <br /> |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls whether the operator creates or adopts an identity. | Create | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the external identity is deleted with this resource. | Orphan | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |


#### InfisicalKubernetesAuth



InfisicalKubernetesAuth is the Schema for the infisicalkubernetesauths API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `infisical.infisical-operator.io/v1alpha1` | | |
| `kind` _string_ | `InfisicalKubernetesAuth` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[InfisicalKubernetesAuthSpec](#infisicalkubernetesauthspec)_ | Spec defines the desired state of InfisicalKubernetesAuth. |  | Required: \{\} <br /> |


#### InfisicalKubernetesAuthSpec



InfisicalKubernetesAuthSpec defines the desired state of Infisical Kubernetes Auth.



_Appears in:_
- [InfisicalKubernetesAuth](#infisicalkubernetesauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[InfisicalConnectionReference](#infisicalconnectionreference)_ | ConnectionRef selects the Infisical API connection. |  |  |
| `identityRef` _[LocalObjectReference](#localobjectreference)_ | IdentityRef references the InfisicalIdentity receiving this auth method. |  |  |
| `kubernetesHost` _string_ | KubernetesHost is the Kubernetes API server URL Infisical uses for token review. |  | MaxLength: 255 <br />Optional: \{\} <br /> |
| `allowedNamespaces` _string array_ | AllowedNamespaces lists the Kubernetes namespaces trusted to authenticate. |  | MinItems: 1 <br /> |
| `allowedNames` _string array_ | AllowedNames lists the service account names trusted to authenticate. |  | MinItems: 1 <br /> |
| `allowedAudience` _string_ | AllowedAudience is the optional audience required in the service account token. |  | MaxLength: 1000 <br />Optional: \{\} <br /> |
| `caCertSecretRef` _[SecretKeyReference](#secretkeyreference)_ | CACertSecretRef references a Secret containing the PEM-encoded Kubernetes API CA certificate. |  | Optional: \{\} <br /> |
| `verifyTLSCertificate` _boolean_ | VerifyTLSCertificate controls Kubernetes API server certificate verification. |  | Optional: \{\} <br /> |
| `tokenReviewerJWTSecretRef` _[SecretKeyReference](#secretkeyreference)_ | TokenReviewerJWTSecretRef references a Secret containing a token for the Kubernetes TokenReview API.<br />If omitted, Infisical can use the authenticating workload token when configured for that mode. |  | Optional: \{\} <br /> |
| `tokenReviewMode` _[KubernetesTokenReviewMode](#kubernetestokenreviewmode)_ | TokenReviewMode selects the API-server token review path. |  | Enum: [api] <br />Optional: \{\} <br /> |
| `accessTokenTrustedIPs` _[KubernetesTrustedIP](#kubernetestrustedip) array_ | AccessTokenTrustedIPs limits where issued access tokens may be used. |  | Optional: \{\} <br /> |
| `accessTokenTTL` _integer_ | AccessTokenTTL is the access token lifetime in seconds. |  | Maximum: 3.1536e+08 <br />Minimum: 0 <br />Optional: \{\} <br /> |
| `accessTokenMaxTTL` _integer_ | AccessTokenMaxTTL is the maximum access token lifetime in seconds. |  | Maximum: 3.1536e+08 <br />Minimum: 0 <br />Optional: \{\} <br /> |
| `accessTokenNumUsesLimit` _integer_ | AccessTokenNumUsesLimit limits access token uses. Zero means unlimited. |  | Minimum: 0 <br />Optional: \{\} <br /> |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls whether the operator attaches or adopts Kubernetes Auth. | Create | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the remote auth method is removed with this resource. | Orphan | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |


#### InfisicalOrganization



InfisicalOrganization is the Schema for the infisicalorganizations API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `infisical.infisical-operator.io/v1alpha1` | | |
| `kind` _string_ | `InfisicalOrganization` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[InfisicalOrganizationSpec](#infisicalorganizationspec)_ | Spec defines the desired state of InfisicalOrganization. |  | Required: \{\} <br /> |


#### InfisicalOrganizationSpec



InfisicalOrganizationSpec defines the desired state of an Infisical organization.



_Appears in:_
- [InfisicalOrganization](#infisicalorganization)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[InfisicalConnectionReference](#infisicalconnectionreference)_ | ConnectionRef selects the Infisical API connection used for organization lifecycle.<br />Creating or deleting an organization requires a user JWT or API key; machine identity<br />tokens can adopt an explicit organization after the platform has provisioned it. |  |  |
| `organizationID` _string_ | OrganizationID identifies an existing organization to adopt. It is also the explicit<br />boundary used by tenant operators, which may not be able to call Infisical's user-only<br />organization lookup endpoint. |  | MinLength: 1 <br />Optional: \{\} <br /> |
| `organizationName` _string_ | OrganizationName is the Infisical display name. If omitted, metadata.name is used.<br />It is required when creating an organization and is used for name-based adoption. |  | MaxLength: 64 <br />MinLength: 1 <br />Optional: \{\} <br /> |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls whether the operator creates or adopts an organization. | Create | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the external organization is deleted with this resource. | Orphan | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |


#### InfisicalProject



InfisicalProject is the Schema for the infisicalprojects API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `infisical.infisical-operator.io/v1alpha1` | | |
| `kind` _string_ | `InfisicalProject` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[InfisicalProjectSpec](#infisicalprojectspec)_ | spec defines the desired state of InfisicalProject |  | Required: \{\} <br /> |


#### InfisicalProjectSpec



InfisicalProjectSpec defines the desired state of InfisicalProject



_Appears in:_
- [InfisicalProject](#infisicalproject)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[InfisicalConnectionReference](#infisicalconnectionreference)_ | ConnectionRef selects the Infisical API connection. |  |  |
| `organizationRef` _[LocalObjectReference](#localobjectreference)_ | OrganizationRef optionally identifies the Infisical organization in which this project<br />must be created or adopted. When set, the observed project organization must match it. |  | Optional: \{\} <br /> |
| `projectName` _string_ | ProjectName is the Infisical project name. If omitted, metadata.name is used. |  | MaxLength: 64 <br />MinLength: 1 <br />Optional: \{\} <br /> |
| `description` _string_ | Description is an optional project description. |  | MaxLength: 1024 <br />Optional: \{\} <br /> |
| `slug` _string_ | Slug is an optional unique project slug. |  | MaxLength: 64 <br />MinLength: 5 <br />Optional: \{\} <br /> |
| `type` _[ProjectType](#projecttype)_ | Type selects the Infisical product type. | secret-manager | Enum: [secret-manager cert-manager] <br />Optional: \{\} <br /> |
| `shouldCreateDefaultEnvs` _boolean_ | ShouldCreateDefaultEnvs controls whether Infisical creates its default environments.<br />It is only used during project creation. | true | Optional: \{\} <br /> |
| `hasDeleteProtection` _boolean_ | HasDeleteProtection configures Infisical-side delete protection. | false | Optional: \{\} <br /> |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls whether the operator creates or adopts a project. | Create | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the external project is deleted with this resource. | Orphan | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |


#### InfisicalUniversalAuth



InfisicalUniversalAuth is the Schema for the infisicaluniversalauths API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `infisical.infisical-operator.io/v1alpha1` | | |
| `kind` _string_ | `InfisicalUniversalAuth` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[InfisicalUniversalAuthSpec](#infisicaluniversalauthspec)_ |  |  |  |


#### InfisicalUniversalAuthSpec



InfisicalUniversalAuthSpec defines the desired state of an Infisical Universal Auth method.



_Appears in:_
- [InfisicalUniversalAuth](#infisicaluniversalauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[InfisicalConnectionReference](#infisicalconnectionreference)_ | ConnectionRef selects the Infisical API connection used to manage the auth method. |  |  |
| `identityRef` _[LocalObjectReference](#localobjectreference)_ | IdentityRef references the same-namespace machine identity receiving Universal Auth. |  |  |
| `config` _[UniversalAuthConfigSpec](#universalauthconfigspec)_ | Config contains optional Universal Auth settings. |  | Optional: \{\} <br /> |
| `clientSecret` _[UniversalAuthClientSecretSpec](#universalauthclientsecretspec)_ | ClientSecret declares the remote client secret and Kubernetes Secret publication. |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls whether the operator attaches or adopts Universal Auth. | Create | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the remote Universal Auth configuration is removed. | Orphan | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |


#### KubernetesTokenReviewMode

_Underlying type:_ _string_

KubernetesTokenReviewMode selects how Infisical validates Kubernetes service account tokens.
The API-server mode is the only Kubernetes Auth mode supported by this free-tier API.

_Validation:_
- Enum: [api]

_Appears in:_
- [InfisicalKubernetesAuthSpec](#infisicalkubernetesauthspec)

| Field | Description |
| --- | --- |
| `api` |  |


#### KubernetesTrustedIP



KubernetesTrustedIP identifies an IP address or CIDR range allowed to use an access token.



_Appears in:_
- [InfisicalKubernetesAuthSpec](#infisicalkubernetesauthspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ipAddress` _string_ | IPAddress is an IP address or CIDR range. |  | MinLength: 1 <br /> |


#### LocalObjectReference



LocalObjectReference identifies a same-namespace custom resource.



_Appears in:_
- [IdentityProjectRoleBinding](#identityprojectrolebinding)
- [InfisicalEnvironmentSpec](#infisicalenvironmentspec)
- [InfisicalIdentitySpec](#infisicalidentityspec)
- [InfisicalKubernetesAuthSpec](#infisicalkubernetesauthspec)
- [InfisicalProjectSpec](#infisicalprojectspec)
- [InfisicalUniversalAuthSpec](#infisicaluniversalauthspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the referenced resource name. |  | MinLength: 1 <br /> |


#### ProjectType

_Underlying type:_ _string_

ProjectType is an Infisical product type supported by the Infisical free plans.
Secret Manager is available in the core free plan and Certificate Manager has
a separate free plan with its documented certificate and CA limits.

_Validation:_
- Enum: [secret-manager cert-manager]

_Appears in:_
- [InfisicalProjectSpec](#infisicalprojectspec)

| Field | Description |
| --- | --- |
| `secret-manager` |  |
| `cert-manager` |  |


#### SecretKeyReference



SecretKeyReference identifies a key in a same-namespace Secret.



_Appears in:_
- [InfisicalConnectionSpec](#infisicalconnectionspec)
- [InfisicalKubernetesAuthSpec](#infisicalkubernetesauthspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the Secret name. |  | MinLength: 1 <br /> |
| `key` _string_ | Key is the Secret data key. |  | MinLength: 1 <br /> |


#### UniversalAuthClientSecretSpec



UniversalAuthClientSecretSpec declares the Kubernetes Secret to publish and its remote secret lifetime.



_Appears in:_
- [InfisicalUniversalAuthSpec](#infisicaluniversalauthspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[UniversalAuthSecretReference](#universalauthsecretreference)_ | SecretRef identifies the same-namespace Secret that will contain clientId and clientSecret. |  |  |
| `description` _string_ | Description is the Infisical client secret description. |  | MaxLength: 255 <br />Optional: \{\} <br /> |
| `numUsesLimit` _integer_ | NumUsesLimit limits client-secret exchanges. Zero means unlimited. |  | Minimum: 0 <br />Optional: \{\} <br /> |
| `ttl` _integer_ | TTL is the client secret lifetime in seconds. Zero means no expiry. |  | Maximum: 3.1536e+08 <br />Minimum: 0 <br />Optional: \{\} <br /> |
| `rotationNonce` _string_ | RotationNonce forces a new remote client secret when changed. |  | Optional: \{\} <br /> |


#### UniversalAuthConfigSpec



UniversalAuthConfigSpec contains the optional Infisical Universal Auth configuration.
Omitted fields retain Infisical's server-side defaults and are not managed by the operator.



_Appears in:_
- [InfisicalUniversalAuthSpec](#infisicaluniversalauthspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `clientSecretTrustedIPs` _[UniversalAuthTrustedIP](#universalauthtrustedip) array_ | ClientSecretTrustedIPs limits where client secrets may be exchanged for access tokens. |  | Optional: \{\} <br /> |
| `accessTokenTrustedIPs` _[UniversalAuthTrustedIP](#universalauthtrustedip) array_ | AccessTokenTrustedIPs limits where issued access tokens may be used. |  | Optional: \{\} <br /> |
| `accessTokenTTL` _integer_ | AccessTokenTTL is the access token lifetime in seconds. |  | Maximum: 3.1536e+08 <br />Minimum: 0 <br />Optional: \{\} <br /> |
| `accessTokenMaxTTL` _integer_ | AccessTokenMaxTTL is the maximum access token lifetime in seconds. |  | Maximum: 3.1536e+08 <br />Minimum: 0 <br />Optional: \{\} <br /> |
| `accessTokenNumUsesLimit` _integer_ | AccessTokenNumUsesLimit limits access token uses. Zero means unlimited. |  | Minimum: 0 <br />Optional: \{\} <br /> |
| `accessTokenPeriod` _integer_ | AccessTokenPeriod is the renewable access-token period in seconds. Zero disables periodic renewal. |  | Minimum: 0 <br />Optional: \{\} <br /> |
| `lockoutEnabled` _boolean_ | LockoutEnabled enables lockout after failed Universal Auth logins. |  | Optional: \{\} <br /> |
| `lockoutThreshold` _integer_ | LockoutThreshold is the number of failed login attempts before lockout. |  | Maximum: 30 <br />Minimum: 1 <br />Optional: \{\} <br /> |
| `lockoutDurationSeconds` _integer_ | LockoutDurationSeconds is how long a locked identity remains locked. |  | Maximum: 86400 <br />Minimum: 30 <br />Optional: \{\} <br /> |
| `lockoutCounterResetSeconds` _integer_ | LockoutCounterResetSeconds is how long until a failed-login counter resets. |  | Maximum: 3600 <br />Minimum: 5 <br />Optional: \{\} <br /> |


#### UniversalAuthConnectionSpec



UniversalAuthConnectionSpec selects a Universal Auth credential Secret for the connection.



_Appears in:_
- [InfisicalConnectionSpec](#infisicalconnectionspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secretRef` _[UniversalAuthSecretReference](#universalauthsecretreference)_ | SecretRef references a same-namespace Secret containing clientId and clientSecret. |  |  |
| `organizationSlug` _string_ | OrganizationSlug optionally scopes Universal Auth login to an Infisical organization.<br />When omitted, Infisical uses the organization where the machine identity was created. |  | MaxLength: 64 <br />Optional: \{\} <br /> |


#### UniversalAuthSecretReference



UniversalAuthSecretReference references a same-namespace Secret containing
an Infisical Universal Auth client ID and client secret.



_Appears in:_
- [UniversalAuthClientSecretSpec](#universalauthclientsecretspec)
- [UniversalAuthConnectionSpec](#universalauthconnectionspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the Secret name. |  | MinLength: 1 <br /> |
| `clientIDKey` _string_ | ClientIDKey is the Secret key containing the client ID. It defaults to clientId. |  | Optional: \{\} <br /> |
| `clientSecretKey` _string_ | ClientSecretKey is the Secret key containing the client secret. It defaults to clientSecret. |  | Optional: \{\} <br /> |


#### UniversalAuthTrustedIP



UniversalAuthTrustedIP identifies an IP address or CIDR range allowed by Universal Auth.



_Appears in:_
- [UniversalAuthConfigSpec](#universalauthconfigspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ipAddress` _string_ | IPAddress is an IP address or CIDR range. |  | MinLength: 1 <br /> |
