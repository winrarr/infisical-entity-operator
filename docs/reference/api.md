# API Reference

## Packages
- [infisical.infisical-operator.io/v1alpha1](#infisicalinfisical-operatoriov1alpha1)


## infisical.infisical-operator.io/v1alpha1

Package v1alpha1 contains API Schema definitions for the infisical v1alpha1 API group.

### Resource Types
- [InfisicalConnection](#infisicalconnection)
- [InfisicalEnvironment](#infisicalenvironment)
- [InfisicalIdentity](#infisicalidentity)
- [InfisicalIdentityTemplate](#infisicalidentitytemplate)
- [InfisicalKubernetesAuth](#infisicalkubernetesauth)
- [InfisicalOrganization](#infisicalorganization)
- [InfisicalProject](#infisicalproject)
- [InfisicalProjectRole](#infisicalprojectrole)
- [InfisicalProjectTemplate](#infisicalprojecttemplate)
- [InfisicalUniversalAuth](#infisicaluniversalauth)



#### CreationPolicy

_Underlying type:_ _string_

CreationPolicy controls how an external entity is acquired.

_Validation:_
- Enum: [Create Adopt CreateOrAdopt]

_Appears in:_
- [InfisicalEnvironmentSpec](#infisicalenvironmentspec)
- [InfisicalIdentitySpec](#infisicalidentityspec)
- [InfisicalIdentityTemplateSpec](#infisicalidentitytemplatespec)
- [InfisicalKubernetesAuthSpec](#infisicalkubernetesauthspec)
- [InfisicalOrganizationSpec](#infisicalorganizationspec)
- [InfisicalProjectRoleSpec](#infisicalprojectrolespec)
- [InfisicalProjectSpec](#infisicalprojectspec)
- [InfisicalProjectTemplateSpec](#infisicalprojecttemplatespec)
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
- [InfisicalIdentityTemplateSpec](#infisicalidentitytemplatespec)
- [InfisicalKubernetesAuthSpec](#infisicalkubernetesauthspec)
- [InfisicalOrganizationSpec](#infisicalorganizationspec)
- [InfisicalProjectRoleSpec](#infisicalprojectrolespec)
- [InfisicalProjectSpec](#infisicalprojectspec)
- [InfisicalProjectTemplateSpec](#infisicalprojecttemplatespec)
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


#### IdentityTemplateAuthMethod

_Underlying type:_ _string_

IdentityTemplateAuthMethod selects the Infisical authentication method configured by a template.

_Validation:_
- Enum: [ldap kubernetes oidc]

_Appears in:_
- [InfisicalIdentityTemplateSpec](#infisicalidentitytemplatespec)

| Field | Description |
| --- | --- |
| `ldap` |  |
| `kubernetes` |  |
| `oidc` |  |


#### IdentityTemplateKubernetesSpec



IdentityTemplateKubernetesSpec defines Kubernetes Auth template fields.



_Appears in:_
- [InfisicalIdentityTemplateSpec](#infisicalidentitytemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `tokenReviewMode` _[KubernetesTokenReviewMode](#kubernetestokenreviewmode)_ | TokenReviewMode selects the API-server or gateway token review path. |  | Enum: [api gateway] <br />Optional: \{\} <br /> |
| `kubernetesHost` _string_ | KubernetesHost is the Kubernetes API server URL used for token review. |  | MaxLength: 255 <br />Optional: \{\} <br /> |
| `caCertSecretRef` _[SecretKeyReference](#secretkeyreference)_ | CACertSecretRef references a Secret containing the PEM-encoded API CA certificate. |  | Optional: \{\} <br /> |
| `verifyTLSCertificate` _boolean_ | VerifyTLSCertificate controls API server certificate verification. |  | Optional: \{\} <br /> |
| `tokenReviewerJWTSecretRef` _[SecretKeyReference](#secretkeyreference)_ | TokenReviewerJWTSecretRef references a Secret containing a TokenReview JWT. |  | Optional: \{\} <br /> |
| `gatewayID` _string_ | GatewayID selects an Infisical gateway. |  | Format: uuid <br />Optional: \{\} <br /> |
| `gatewayPoolID` _string_ | GatewayPoolID selects an Infisical gateway pool. |  | Format: uuid <br />Optional: \{\} <br /> |
| `allowedAudience` _string_ | AllowedAudience restricts the audience claim on authenticating tokens. |  | MaxLength: 1000 <br />Optional: \{\} <br /> |


#### IdentityTemplateLDAPSpec



IdentityTemplateLDAPSpec defines LDAP Auth template fields.



_Appears in:_
- [InfisicalIdentityTemplateSpec](#infisicalidentitytemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `url` _string_ | URL is the LDAP server URL. |  | MinLength: 1 <br /> |
| `bindDN` _string_ | BindDN is the LDAP bind distinguished name. |  | MinLength: 1 <br /> |
| `bindPasswordSecretRef` _[SecretKeyReference](#secretkeyreference)_ | BindPasswordSecretRef references a Secret containing the LDAP bind password. |  |  |
| `searchBase` _string_ | SearchBase is the LDAP search base. |  | MinLength: 1 <br /> |
| `caCertSecretRef` _[SecretKeyReference](#secretkeyreference)_ | CACertSecretRef references a Secret containing the LDAP CA certificate. |  | Optional: \{\} <br /> |


#### IdentityTemplateOIDCSpec



IdentityTemplateOIDCSpec defines OIDC Auth template fields.



_Appears in:_
- [InfisicalIdentityTemplateSpec](#infisicalidentitytemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `oidcDiscoveryURL` _string_ | OIDCDiscoveryURL is the identity provider discovery URL. |  | Format: uri <br />MaxLength: 2048 <br />MinLength: 1 <br /> |
| `boundIssuer` _string_ | BoundIssuer is the expected JWT issuer. |  | MaxLength: 2048 <br />MinLength: 1 <br /> |
| `boundAudiences` _string_ | BoundAudiences is the comma-separated audience list. |  | MaxLength: 2048 <br />Optional: \{\} <br /> |
| `caCertSecretRef` _[SecretKeyReference](#secretkeyreference)_ | CACertSecretRef references a Secret containing the identity provider CA certificate. |  | Optional: \{\} <br /> |


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
- [InfisicalIdentityTemplateSpec](#infisicalidentitytemplatespec)
- [InfisicalKubernetesAuthSpec](#infisicalkubernetesauthspec)
- [InfisicalOrganizationSpec](#infisicalorganizationspec)
- [InfisicalProjectRoleSpec](#infisicalprojectrolespec)
- [InfisicalProjectSpec](#infisicalprojectspec)
- [InfisicalProjectTemplateSpec](#infisicalprojecttemplatespec)
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
| `roleSlugs` _string array_ | RoleSlugs declares the permanent project role slugs assigned to the identity.<br />When omitted, the existing project membership is not managed. Use no-access<br />explicitly when the identity should have no project permissions. |  | MinItems: 1 <br />Optional: \{\} <br /> |
| `organizationRole` _string_ | OrganizationRole is the Infisical organization role for an organization-scoped identity.<br />Leave it empty to use Infisical's least-privilege no-access role. Project-scoped identities<br />must omit this field. |  | MinLength: 1 <br />Optional: \{\} <br /> |
| `projectRoleBindings` _[IdentityProjectRoleBinding](#identityprojectrolebinding) array_ | ProjectRoleBindings grants an organization-scoped identity roles in selected projects.<br />A binding manages that project's complete permanent role list; omitted projects are left<br />unmanaged. Every referenced project must belong to the organization from organizationRef. |  | Optional: \{\} <br /> |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls whether the operator creates or adopts an identity. | Create | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the external identity is deleted with this resource. | Orphan | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |


#### InfisicalIdentityTemplate



InfisicalIdentityTemplate is the Schema for the infisicalidentitytemplates API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `infisical.infisical-operator.io/v1alpha1` | | |
| `kind` _string_ | `InfisicalIdentityTemplate` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[InfisicalIdentityTemplateSpec](#infisicalidentitytemplatespec)_ |  |  |  |


#### InfisicalIdentityTemplateSpec



InfisicalIdentityTemplateSpec defines an organization-owned identity authentication template.



_Appears in:_
- [InfisicalIdentityTemplate](#infisicalidentitytemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[InfisicalConnectionReference](#infisicalconnectionreference)_ | ConnectionRef selects the Infisical API connection. |  |  |
| `organizationRef` _[LocalObjectReference](#localobjectreference)_ | OrganizationRef identifies the organization that owns this template. |  |  |
| `templateName` _string_ | TemplateName is the Infisical template name. If omitted, metadata.name is used. |  | MaxLength: 64 <br />MinLength: 1 <br />Optional: \{\} <br /> |
| `authMethod` _[IdentityTemplateAuthMethod](#identitytemplateauthmethod)_ | AuthMethod selects the template authentication method. |  | Enum: [ldap kubernetes oidc] <br /> |
| `ldap` _[IdentityTemplateLDAPSpec](#identitytemplateldapspec)_ | LDAP contains LDAP template settings when authMethod is ldap. |  | Optional: \{\} <br /> |
| `kubernetes` _[IdentityTemplateKubernetesSpec](#identitytemplatekubernetesspec)_ | Kubernetes contains Kubernetes Auth template settings when authMethod is kubernetes. |  | Optional: \{\} <br /> |
| `oidc` _[IdentityTemplateOIDCSpec](#identitytemplateoidcspec)_ | OIDC contains OIDC template settings when authMethod is oidc. |  | Optional: \{\} <br /> |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls whether the operator creates or adopts a template. | Create | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the external template is deleted with this resource. | Orphan | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |


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
| `templateRef` _[LocalObjectReference](#localobjectreference)_ | TemplateRef selects a managed Infisical Kubernetes Auth template. When set, Infisical manages<br />KubernetesHost, CACertSecretRef, TokenReviewerJWTSecretRef, TokenReviewMode, GatewayID,<br />GatewayPoolID, and AllowedAudience from that template; those fields must be omitted. |  | Optional: \{\} <br /> |
| `kubernetesHost` _string_ | KubernetesHost is the Kubernetes API server URL Infisical uses for token review. |  | MaxLength: 255 <br />Optional: \{\} <br /> |
| `allowedNamespaces` _string array_ | AllowedNamespaces lists the Kubernetes namespaces trusted to authenticate. |  | MinItems: 1 <br /> |
| `allowedNames` _string array_ | AllowedNames lists the service account names trusted to authenticate. |  | MinItems: 1 <br /> |
| `allowedAudience` _string_ | AllowedAudience is the optional audience required in the service account token. |  | MaxLength: 1000 <br />Optional: \{\} <br /> |
| `caCertSecretRef` _[SecretKeyReference](#secretkeyreference)_ | CACertSecretRef references a Secret containing the PEM-encoded Kubernetes API CA certificate. |  | Optional: \{\} <br /> |
| `verifyTLSCertificate` _boolean_ | VerifyTLSCertificate controls Kubernetes API server certificate verification. |  | Optional: \{\} <br /> |
| `tokenReviewerJWTSecretRef` _[SecretKeyReference](#secretkeyreference)_ | TokenReviewerJWTSecretRef references a Secret containing a token for the Kubernetes TokenReview API.<br />If omitted, Infisical can use the authenticating workload token when configured for that mode. |  | Optional: \{\} <br /> |
| `tokenReviewMode` _[KubernetesTokenReviewMode](#kubernetestokenreviewmode)_ | TokenReviewMode selects the API-server or gateway token review path. |  | Enum: [api gateway] <br />Optional: \{\} <br /> |
| `gatewayID` _string_ | GatewayID selects an Infisical gateway for token review when gateway mode is used. |  | Format: uuid <br />Optional: \{\} <br /> |
| `gatewayPoolID` _string_ | GatewayPoolID selects an Infisical gateway pool for token review when gateway mode is used. |  | Format: uuid <br />Optional: \{\} <br /> |
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
| `organizationRef` _[LocalObjectReference](#localobjectreference)_ | OrganizationRef optionally identifies the Infisical organization in which this project<br />must be created or adopted. When set, the observed project organization must match it. |  | Optional: \{\} <br /> |
| `templateRef` _[LocalObjectReference](#localobjectreference)_ | TemplateRef optionally selects an InfisicalProjectTemplate to apply during project creation.<br />Template changes are not propagated to an existing project because Infisical applies the<br />template only when the project is created. |  | Optional: \{\} <br /> |
| `projectName` _string_ | ProjectName is the Infisical project name. If omitted, metadata.name is used. |  | MaxLength: 64 <br />MinLength: 1 <br />Optional: \{\} <br /> |
| `description` _string_ | Description is an optional project description. |  | MaxLength: 1024 <br />Optional: \{\} <br /> |
| `slug` _string_ | Slug is an optional unique project slug. |  | MaxLength: 64 <br />MinLength: 5 <br />Optional: \{\} <br /> |
| `kmsKeyID` _string_ | KMSKeyID optionally selects the Infisical KMS key used to protect this project.<br />This is distinct from Type=kms, which selects Infisical's KMS product project type.<br />It is only used during project creation. |  | MinLength: 1 <br />Optional: \{\} <br /> |
| `type` _[ProjectType](#projecttype)_ | Type selects the Infisical product type. | secret-manager | Enum: [secret-manager cert-manager kms secret-scanning pam agent-vault] <br />Optional: \{\} <br /> |
| `shouldCreateDefaultEnvs` _boolean_ | ShouldCreateDefaultEnvs controls whether Infisical creates its default environments.<br />It is only used during project creation. | true | Optional: \{\} <br /> |
| `hasDeleteProtection` _boolean_ | HasDeleteProtection configures Infisical-side delete protection. | false | Optional: \{\} <br /> |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls whether the operator creates or adopts a project. | Create | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the external project is deleted with this resource. | Orphan | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |


#### InfisicalProjectTemplate



InfisicalProjectTemplate is the Schema for the infisicalprojecttemplates API.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `infisical.infisical-operator.io/v1alpha1` | | |
| `kind` _string_ | `InfisicalProjectTemplate` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[InfisicalProjectTemplateSpec](#infisicalprojecttemplatespec)_ | Spec defines the desired state of InfisicalProjectTemplate. |  | Required: \{\} <br /> |


#### InfisicalProjectTemplateSpec



InfisicalProjectTemplateSpec defines the desired state of an Infisical project template.



_Appears in:_
- [InfisicalProjectTemplate](#infisicalprojecttemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[InfisicalConnectionReference](#infisicalconnectionreference)_ | ConnectionRef selects the Infisical API connection. |  |  |
| `organizationRef` _[LocalObjectReference](#localobjectreference)_ | OrganizationRef optionally identifies the organization that owns this template. |  | Optional: \{\} <br /> |
| `templateName` _string_ | TemplateName is the Infisical template name. If omitted, metadata.name is used. |  | MaxLength: 64 <br />MinLength: 1 <br />Optional: \{\} <br /> |
| `description` _string_ | Description is an optional template description. |  | MaxLength: 256 <br />Optional: \{\} <br /> |
| `type` _[ProjectType](#projecttype)_ | Type selects the Infisical product type for projects created from this template. | secret-manager | Enum: [secret-manager cert-manager kms secret-scanning pam agent-vault] <br />Optional: \{\} <br /> |
| `roles` _[ProjectTemplateRole](#projecttemplaterole) array_ | Roles contains the custom project roles created by this template. |  | Optional: \{\} <br /> |
| `environments` _[ProjectTemplateEnvironment](#projecttemplateenvironment) array_ | Environments contains the environments created by this template. |  | Optional: \{\} <br /> |
| `users` _[ProjectTemplateUser](#projecttemplateuser) array_ | Users contains users automatically added to projects created from this template. |  | Optional: \{\} <br /> |
| `groups` _[ProjectTemplateGroup](#projecttemplategroup) array_ | Groups contains groups automatically added to projects created from this template.<br />Group support depends on the Infisical plan and group configuration. |  | Optional: \{\} <br /> |
| `identities` _[ProjectTemplateIdentity](#projecttemplateidentity) array_ | Identities contains organization-owned identities automatically added to projects. |  | Optional: \{\} <br /> |
| `projectManagedIdentities` _[ProjectTemplateManagedIdentity](#projecttemplatemanagedidentity) array_ | ProjectManagedIdentities contains project-owned identities created from this template. |  | Optional: \{\} <br /> |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls whether the operator creates or adopts a template. | Create | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether the external template is deleted with this resource. | Orphan | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |


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

_Validation:_
- Enum: [api gateway]

_Appears in:_
- [IdentityTemplateKubernetesSpec](#identitytemplatekubernetesspec)
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
- [IdentityProjectRoleBinding](#identityprojectrolebinding)
- [InfisicalEnvironmentSpec](#infisicalenvironmentspec)
- [InfisicalIdentitySpec](#infisicalidentityspec)
- [InfisicalIdentityTemplateSpec](#infisicalidentitytemplatespec)
- [InfisicalKubernetesAuthSpec](#infisicalkubernetesauthspec)
- [InfisicalProjectRoleSpec](#infisicalprojectrolespec)
- [InfisicalProjectSpec](#infisicalprojectspec)
- [InfisicalProjectTemplateSpec](#infisicalprojecttemplatespec)
- [InfisicalUniversalAuthSpec](#infisicaluniversalauthspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the referenced resource name. |  | MinLength: 1 <br /> |


#### ProjectRoleAction

_Underlying type:_ _string_

ProjectRoleAction identifies an Infisical project-role action.

_Validation:_
- Enum: [assign-additional-privileges assign-role assume-privileges attach-hsm-connectors connect-app-connections create create-app-connections create-clients create-data-sources create-grant create-hsm-connectors create-root-credential create-token decrypt delete delete-app-connections delete-clients delete-data-sources delete-hsm-connectors delete-report delete-root-credential delete-token describeSecret edit edit-app-connections edit-auth edit-data-sources edit-hsm-connectors edit-root-credential encrypt export-private-key generate-client-certificates generate-mac generate-report get-token grant-privileges import import-certificates import-secrets issue-ca-certificate issue-cert issue-token lease list list-certs manage-application-attachments manage-members perform-rollback proxy read read-app-connections read-clients read-configs read-credentials read-data-source-resources read-data-source-scans read-data-sources read-findings read-generated-credentials read-grant read-hsm-connectors read-private-key read-root-credential readValue remove-certificates remove-secrets report-usage reset reveal-acme-eab-secret revoke revoke-auth revoke-grant rotate rotate-acme-eab-secret rotate-credentials rotate-secrets run-scan set-health-check-command set-post-sync-command set-target-host sign sign-intermediate subscribe-to-creation-events subscribe-to-deletion-events subscribe-to-import-mutation-events subscribe-to-update-events sync-certificates sync-secrets test-hsm-connectors trigger-data-source-scans update-clients update-configs update-findings verify verify-mac]

_Appears in:_
- [ProjectRolePermission](#projectrolepermission)



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
- [ProjectTemplateRole](#projecttemplaterole)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `subject` _[ProjectRoleSubject](#projectrolesubject)_ | Subject identifies the Infisical resource subject. |  | Enum: [secrets secret-folders secret-imports dynamic-secrets identity pki-subscribers certificate-templates secret-rotation secret-syncs pki-syncs secret-event-subscriptions certificate-profiles certificate-policies certificate-application certificate-authorities certificates secret-approval secret-rollback member groups role integrations webhooks service-tokens settings secret-validation-rules environments tags audit-logs insights ip-allowlist pki-alerts pki-collections certificate-inventory-views pki-discovery pki-certificate-installations code-signers workspace kms cmek kmip commits secret-scanning-data-sources secret-scanning-findings secret-scanning-configs app-connections hsm-connectors honey-tokens proxied-services agent-vault-access-bundles agent-vault-sessions agent-vault-proxies approval-requests approval-request-grants secret-approval-request project-folder-grant] <br /> |
| `action` _[ProjectRoleAction](#projectroleaction) array_ | Action lists one or more operations allowed on the subject. |  | Enum: [assign-additional-privileges assign-role assume-privileges attach-hsm-connectors connect-app-connections create create-app-connections create-clients create-data-sources create-grant create-hsm-connectors create-root-credential create-token decrypt delete delete-app-connections delete-clients delete-data-sources delete-hsm-connectors delete-report delete-root-credential delete-token describeSecret edit edit-app-connections edit-auth edit-data-sources edit-hsm-connectors edit-root-credential encrypt export-private-key generate-client-certificates generate-mac generate-report get-token grant-privileges import import-certificates import-secrets issue-ca-certificate issue-cert issue-token lease list list-certs manage-application-attachments manage-members perform-rollback proxy read read-app-connections read-clients read-configs read-credentials read-data-source-resources read-data-source-scans read-data-sources read-findings read-generated-credentials read-grant read-hsm-connectors read-private-key read-root-credential readValue remove-certificates remove-secrets report-usage reset reveal-acme-eab-secret revoke revoke-auth revoke-grant rotate rotate-acme-eab-secret rotate-credentials rotate-secrets run-scan set-health-check-command set-post-sync-command set-target-host sign sign-intermediate subscribe-to-creation-events subscribe-to-deletion-events subscribe-to-import-mutation-events subscribe-to-update-events sync-certificates sync-secrets test-hsm-connectors trigger-data-source-scans update-clients update-configs update-findings verify verify-mac] <br />MinItems: 1 <br /> |
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


#### ProjectRoleSubject

_Underlying type:_ _string_

ProjectRoleSubject identifies an Infisical project-role subject.

_Validation:_
- Enum: [secrets secret-folders secret-imports dynamic-secrets identity pki-subscribers certificate-templates secret-rotation secret-syncs pki-syncs secret-event-subscriptions certificate-profiles certificate-policies certificate-application certificate-authorities certificates secret-approval secret-rollback member groups role integrations webhooks service-tokens settings secret-validation-rules environments tags audit-logs insights ip-allowlist pki-alerts pki-collections certificate-inventory-views pki-discovery pki-certificate-installations code-signers workspace kms cmek kmip commits secret-scanning-data-sources secret-scanning-findings secret-scanning-configs app-connections hsm-connectors honey-tokens proxied-services agent-vault-access-bundles agent-vault-sessions agent-vault-proxies approval-requests approval-request-grants secret-approval-request project-folder-grant]

_Appears in:_
- [ProjectRolePermission](#projectrolepermission)



#### ProjectTemplateEnvironment



ProjectTemplateEnvironment describes an environment created when a project uses a template.



_Appears in:_
- [InfisicalProjectTemplateSpec](#infisicalprojecttemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the environment display name. |  | MinLength: 1 <br /> |
| `slug` _string_ | Slug is the stable environment slug. |  | MaxLength: 64 <br />MinLength: 1 <br /> |
| `position` _integer_ | Position controls the environment order in the project. |  | Minimum: 1 <br /> |


#### ProjectTemplateGroup



ProjectTemplateGroup assigns roles to a group added to projects created from a template.



_Appears in:_
- [InfisicalProjectTemplateSpec](#infisicalprojecttemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `groupSlug` _string_ | GroupSlug identifies the Infisical group. |  | MinLength: 1 <br /> |
| `roles` _string array_ | Roles contains role slugs assigned to the group. |  | MinItems: 1 <br /> |


#### ProjectTemplateIdentity



ProjectTemplateIdentity assigns roles to an organization-owned identity added to projects.



_Appears in:_
- [InfisicalProjectTemplateSpec](#infisicalprojecttemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `identityID` _string_ | IdentityID is the Infisical machine identity identifier. |  | Format: uuid <br /> |
| `roles` _string array_ | Roles contains role slugs assigned to the identity. |  | MinItems: 1 <br /> |


#### ProjectTemplateManagedIdentity



ProjectTemplateManagedIdentity creates a project-owned identity and assigns roles to it.



_Appears in:_
- [InfisicalProjectTemplateSpec](#infisicalprojecttemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the project-owned identity name. |  | MinLength: 1 <br /> |
| `roles` _string array_ | Roles contains role slugs assigned to the identity. |  | MinItems: 1 <br /> |


#### ProjectTemplateRole



ProjectTemplateRole describes one custom role in a project template.



_Appears in:_
- [InfisicalProjectTemplateSpec](#infisicalprojecttemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the role display name. |  | MinLength: 1 <br /> |
| `slug` _string_ | Slug is the stable role slug. |  | MaxLength: 64 <br />MinLength: 1 <br /> |
| `permissions` _[ProjectRolePermission](#projectrolepermission) array_ | Permissions contains the role permission rules. |  | Optional: \{\} <br /> |


#### ProjectTemplateUser



ProjectTemplateUser assigns roles to a user added to projects created from a template.



_Appears in:_
- [InfisicalProjectTemplateSpec](#infisicalprojecttemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `username` _string_ | Username identifies the Infisical user, normally by username or email. |  | MinLength: 1 <br /> |
| `roles` _string array_ | Roles contains role slugs assigned to the user. |  | MinItems: 1 <br /> |


#### ProjectType

_Underlying type:_ _string_

ProjectType is an Infisical product type.

_Validation:_
- Enum: [secret-manager cert-manager kms secret-scanning pam agent-vault]

_Appears in:_
- [InfisicalProjectSpec](#infisicalprojectspec)
- [InfisicalProjectTemplateSpec](#infisicalprojecttemplatespec)

| Field | Description |
| --- | --- |
| `secret-manager` |  |
| `cert-manager` |  |
| `kms` |  |
| `secret-scanning` |  |
| `pam` |  |
| `agent-vault` |  |


#### SecretKeyReference



SecretKeyReference identifies a key in a same-namespace Secret.



_Appears in:_
- [IdentityTemplateKubernetesSpec](#identitytemplatekubernetesspec)
- [IdentityTemplateLDAPSpec](#identitytemplateldapspec)
- [IdentityTemplateOIDCSpec](#identitytemplateoidcspec)
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
