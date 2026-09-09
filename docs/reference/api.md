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
- [InfisicalProject](#infisicalproject)
- [InfisicalProjectRole](#infisicalprojectrole)



#### CreationPolicy

_Underlying type:_ _string_

CreationPolicy controls how an external entity is acquired.

_Validation:_
- Enum: [Create Adopt CreateOrAdopt]

_Appears in:_
- [InfisicalEnvironmentSpec](#infisicalenvironmentspec)
- [InfisicalIdentitySpec](#infisicalidentityspec)
- [InfisicalKubernetesAuthSpec](#infisicalkubernetesauthspec)
- [InfisicalProjectRoleSpec](#infisicalprojectrolespec)
- [InfisicalProjectSpec](#infisicalprojectspec)

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
- [InfisicalProjectRoleSpec](#infisicalprojectrolespec)
- [InfisicalProjectSpec](#infisicalprojectspec)

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
- [InfisicalProjectRoleSpec](#infisicalprojectrolespec)
- [InfisicalProjectSpec](#infisicalprojectspec)

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
| `authSecretRef` _[SecretKeyReference](#secretkeyreference)_ | AuthSecretRef references a Secret containing a bearer token under Key.<br />The Secret must be in the same namespace as this connection. |  |  |
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
| `projectRef` _[LocalObjectReference](#localobjectreference)_ | ProjectRef references the InfisicalProject resource that owns this identity. |  |  |
| `identityName` _string_ | IdentityName is the Infisical identity name. If omitted, metadata.name is used. |  | MinLength: 1 <br />Optional: \{\} <br /> |
| `hasDeleteProtection` _boolean_ | HasDeleteProtection configures Infisical-side delete protection. | false | Optional: \{\} <br /> |
| `metadata` _[IdentityMetadata](#identitymetadata) array_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `roleSlugs` _string array_ | RoleSlugs declares the permanent project role slugs assigned to the identity.<br />When omitted, the existing project membership is not managed. Use no-access<br />explicitly when the identity should have no project permissions. |  | MinItems: 1 <br />Optional: \{\} <br /> |
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
| `tokenReviewMode` _[KubernetesTokenReviewMode](#kubernetestokenreviewmode)_ | TokenReviewMode selects the API-server or gateway token review path. | api | Enum: [api gateway] <br />Optional: \{\} <br /> |
| `gatewayID` _string_ | GatewayID selects an Infisical gateway for token review when gateway mode is used. |  | Optional: \{\} <br /> |
| `gatewayPoolID` _string_ | GatewayPoolID selects an Infisical gateway pool for token review when gateway mode is used. |  | Optional: \{\} <br /> |
| `accessTokenTrustedIPs` _[KubernetesTrustedIP](#kubernetestrustedip) array_ | AccessTokenTrustedIPs limits where issued access tokens may be used. |  | Optional: \{\} <br /> |
| `accessTokenTTL` _integer_ | AccessTokenTTL is the access token lifetime in seconds. |  | Maximum: 3.1536e+08 <br />Minimum: 0 <br />Optional: \{\} <br /> |
| `accessTokenMaxTTL` _integer_ | AccessTokenMaxTTL is the maximum access token lifetime in seconds. |  | Maximum: 3.1536e+08 <br />Minimum: 0 <br />Optional: \{\} <br /> |
| `accessTokenNumUsesLimit` _integer_ | AccessTokenNumUsesLimit limits access token uses. Zero means unlimited. |  | Minimum: 0 <br />Optional: \{\} <br /> |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls whether the operator attaches or adopts Kubernetes Auth. | Create | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the remote auth method is removed with this resource. | Orphan | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |


#### InfisicalProject



InfisicalProject is the Schema for the infisicalprojects API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `infisical.infisical-operator.io/v1alpha1` | | |
| `kind` _string_ | `InfisicalProject` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[InfisicalProjectSpec](#infisicalprojectspec)_ | spec defines the desired state of InfisicalProject |  | Required: \{\} <br /> |


#### InfisicalProjectRole



InfisicalProjectRole is the Schema for the infisicalprojectroles API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `infisical.infisical-operator.io/v1alpha1` | | |
| `kind` _string_ | `InfisicalProjectRole` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[InfisicalProjectRoleSpec](#infisicalprojectrolespec)_ | Spec defines the desired state of InfisicalProjectRole. |  | Required: \{\} <br /> |


#### InfisicalProjectRoleSpec



InfisicalProjectRoleSpec defines the desired state of InfisicalProjectRole.



_Appears in:_
- [InfisicalProjectRole](#infisicalprojectrole)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[InfisicalConnectionReference](#infisicalconnectionreference)_ | ConnectionRef selects the Infisical API connection. |  |  |
| `projectRef` _[LocalObjectReference](#localobjectreference)_ | ProjectRef references the InfisicalProject that owns this role. |  |  |
| `roleName` _string_ | RoleName is the Infisical role display name. If omitted, metadata.name is used. |  | MinLength: 1 <br />Optional: \{\} <br /> |
| `slug` _string_ | Slug is the stable Infisical role slug. If omitted, metadata.name is used. |  | MaxLength: 64 <br />MinLength: 1 <br />Optional: \{\} <br /> |
| `description` _string_ | Description is an optional role description. |  | MaxLength: 1024 <br />Optional: \{\} <br /> |
| `permissions` _[ProjectRolePermission](#projectrolepermission) array_ | Permissions contains the role permission rules. |  | MinItems: 1 <br /> |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls whether the operator creates or adopts a role. | Create | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the external role is deleted with this resource. | Orphan | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |


#### InfisicalProjectSpec



InfisicalProjectSpec defines the desired state of InfisicalProject



_Appears in:_
- [InfisicalProject](#infisicalproject)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[InfisicalConnectionReference](#infisicalconnectionreference)_ | ConnectionRef selects the Infisical API connection. |  |  |
| `projectName` _string_ | ProjectName is the Infisical project name. If omitted, metadata.name is used. |  | MaxLength: 64 <br />MinLength: 1 <br />Optional: \{\} <br /> |
| `description` _string_ | Description is an optional project description. |  | MaxLength: 1024 <br />Optional: \{\} <br /> |
| `slug` _string_ | Slug is an optional unique project slug. |  | MaxLength: 64 <br />MinLength: 5 <br />Optional: \{\} <br /> |
| `type` _[ProjectType](#projecttype)_ | Type selects the Infisical product type. | secret-manager | Enum: [secret-manager cert-manager kms ssh secret-scanning pam ai] <br />Optional: \{\} <br /> |
| `shouldCreateDefaultEnvs` _boolean_ | ShouldCreateDefaultEnvs controls whether Infisical creates its default environments.<br />It is only used during project creation. | true | Optional: \{\} <br /> |
| `hasDeleteProtection` _boolean_ | HasDeleteProtection configures Infisical-side delete protection. | false | Optional: \{\} <br /> |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls whether the operator creates or adopts a project. | Create | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the external project is deleted with this resource. | Orphan | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |


#### KubernetesTokenReviewMode

_Underlying type:_ _string_

KubernetesTokenReviewMode selects how Infisical validates Kubernetes service account tokens.

_Validation:_
- Enum: [api gateway]

_Appears in:_
- [InfisicalKubernetesAuthSpec](#infisicalkubernetesauthspec)

| Field | Description |
| --- | --- |
| `api` |  |
| `gateway` |  |


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
- [InfisicalEnvironmentSpec](#infisicalenvironmentspec)
- [InfisicalIdentitySpec](#infisicalidentityspec)
- [InfisicalKubernetesAuthSpec](#infisicalkubernetesauthspec)
- [InfisicalProjectRoleSpec](#infisicalprojectrolespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the referenced resource name. |  | MinLength: 1 <br /> |


#### ProjectRoleConditions



ProjectRoleConditions limits a permission to matching project resources.



_Appears in:_
- [ProjectRolePermission](#projectrolepermission)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `environment` _[ProjectRoleStringCondition](#projectrolestringcondition)_ | Environment limits the environment. |  | Optional: \{\} <br /> |
| `secretPath` _[ProjectRoleStringCondition](#projectrolestringcondition)_ | SecretPath limits the secret path. |  | Optional: \{\} <br /> |
| `secretName` _[ProjectRoleStringCondition](#projectrolestringcondition)_ | SecretName limits the secret name. |  | Optional: \{\} <br /> |
| `secretTags` _[ProjectRoleSecretTagsCondition](#projectrolesecrettagscondition)_ | SecretTags limits secret tags. |  | Optional: \{\} <br /> |
| `eventType` _[ProjectRoleStringCondition](#projectrolestringcondition)_ | EventType limits audit or event permissions. |  | Optional: \{\} <br /> |


#### ProjectRolePermission



ProjectRolePermission defines one subject/action permission rule.



_Appears in:_
- [InfisicalProjectRoleSpec](#infisicalprojectrolespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `subject` _string_ | Subject identifies the Infisical resource subject. |  | MinLength: 1 <br /> |
| `action` _string array_ | Action lists one or more operations allowed on the subject. |  | MinItems: 1 <br /> |
| `inverted` _boolean_ | Inverted turns the rule into a deny rule when true. |  | Optional: \{\} <br /> |
| `conditions` _[ProjectRoleConditions](#projectroleconditions)_ | Conditions limits the rule to matching resources. |  | Optional: \{\} <br /> |


#### ProjectRoleSecretTagsCondition



ProjectRoleSecretTagsCondition describes secret tag matching.



_Appears in:_
- [ProjectRoleConditions](#projectroleconditions)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `$in` _string array_ | In matches any listed tag. |  | Optional: \{\} <br /> |
| `$all` _string array_ | All requires all listed tags. |  | Optional: \{\} <br /> |


#### ProjectRoleStringCondition



ProjectRoleStringCondition describes a string condition using Infisical's comparison operators.



_Appears in:_
- [ProjectRoleConditions](#projectroleconditions)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `$eq` _string_ | Eq matches an exact value. |  | Optional: \{\} <br /> |
| `$ne` _string_ | Ne excludes an exact value. |  | Optional: \{\} <br /> |
| `$in` _string array_ | In matches one of the listed values. |  | Optional: \{\} <br /> |
| `$glob` _string_ | Glob matches a glob pattern. |  | Optional: \{\} <br /> |


#### ProjectType

_Underlying type:_ _string_

ProjectType is an Infisical product type.

_Validation:_
- Enum: [secret-manager cert-manager kms ssh secret-scanning pam ai]

_Appears in:_
- [InfisicalProjectSpec](#infisicalprojectspec)

| Field | Description |
| --- | --- |
| `secret-manager` |  |
| `cert-manager` |  |
| `kms` |  |
| `ssh` |  |
| `secret-scanning` |  |
| `pam` |  |
| `ai` |  |


#### SecretKeyReference



SecretKeyReference identifies a key in a same-namespace Secret.



_Appears in:_
- [InfisicalConnectionSpec](#infisicalconnectionspec)
- [InfisicalKubernetesAuthSpec](#infisicalkubernetesauthspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the Secret name. |  | MinLength: 1 <br /> |
| `key` _string_ | Key is the Secret data key. |  | MinLength: 1 <br /> |
