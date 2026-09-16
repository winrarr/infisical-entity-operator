/*
Copyright 2026 winrarr.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package infisical

import (
	"fmt"
	"strings"

	infisicalv1alpha1 "github.com/winrarr/infisical-entity-operator/api/infisical/v1alpha1"
)

// The repeated action names intentionally keep this OpenAPI-derived matrix readable.
//
//nolint:goconst
var projectRoleActionsBySubject = map[string][]string{
	"secrets":                       {"create", "delete", "describeSecret", "edit", "read", "readValue"},
	"secret-folders":                {"create", "delete", "edit", "read"},
	"secret-imports":                {"create", "delete", "edit", "read"},
	"dynamic-secrets":               {"create-root-credential", "delete-root-credential", "edit-root-credential", "lease", "read-root-credential"},
	"identity":                      {"assign-additional-privileges", "assign-role", "assume-privileges", "create", "create-token", "delete", "delete-token", "edit", "edit-auth", "get-token", "grant-privileges", "read", "revoke-auth"},
	"pki-subscribers":               {"create", "delete", "edit", "issue-cert", "list-certs", "read"},
	"certificate-templates":         {"create", "delete", "edit", "issue-cert", "list-certs", "read"},
	"secret-rotation":               {"create", "delete", "edit", "read", "read-generated-credentials", "rotate-secrets"},
	"secret-syncs":                  {"create", "delete", "edit", "import-secrets", "read", "remove-secrets", "sync-secrets"},
	"pki-syncs":                     {"create", "delete", "edit", "import-certificates", "read", "remove-certificates", "set-health-check-command", "set-post-sync-command", "set-target-host", "sync-certificates"},
	"secret-event-subscriptions":    {"subscribe-to-creation-events", "subscribe-to-deletion-events", "subscribe-to-import-mutation-events", "subscribe-to-update-events"},
	"certificate-profiles":          {"create", "delete", "edit", "issue-cert", "manage-application-attachments", "read", "reveal-acme-eab-secret", "rotate-acme-eab-secret"},
	"certificate-policies":          {"create", "delete", "edit", "read"},
	"certificate-application":       {"create", "list", "read"},
	"certificate-authorities":       {"create", "delete", "edit", "issue-ca-certificate", "read", "sign-intermediate"},
	"certificates":                  {"create", "delete", "edit", "import", "read", "read-private-key"},
	"secret-approval":               {"create", "delete", "edit", "read"},
	"secret-rollback":               {"create", "read"},
	"member":                        {"assign-additional-privileges", "assign-role", "assume-privileges", "create", "delete", "edit", "grant-privileges", "read"},
	"groups":                        {"assign-role", "create", "delete", "edit", "grant-privileges", "read"},
	"role":                          {"create", "delete", "edit", "read"},
	"integrations":                  {"create", "delete", "edit", "read"},
	"webhooks":                      {"create", "delete", "edit", "read"},
	"service-tokens":                {"create", "delete", "edit", "read"},
	"settings":                      {"create", "delete", "edit", "read"},
	"secret-validation-rules":       {"create", "delete", "edit", "read"},
	"environments":                  {"create", "delete", "edit", "read"},
	"tags":                          {"create", "delete", "edit", "read"},
	"audit-logs":                    {"read"},
	"insights":                      {"delete-report", "generate-report", "read"},
	"ip-allowlist":                  {"create", "delete", "edit", "read"},
	"pki-alerts":                    {"create", "delete", "edit", "read"},
	"pki-collections":               {"create", "delete", "edit", "read"},
	"certificate-inventory-views":   {"create", "delete", "edit", "read"},
	"pki-discovery":                 {"create", "delete", "edit", "read", "run-scan"},
	"pki-certificate-installations": {"delete", "edit", "read"},
	"code-signers":                  {"create", "delete", "edit", "read", "sign"},
	"workspace":                     {"delete", "edit"},
	"kms":                           {"edit"},
	"cmek":                          {"create", "decrypt", "delete", "edit", "encrypt", "export-private-key", "generate-mac", "read", "rotate", "sign", "verify", "verify-mac"},
	"kmip":                          {"create-clients", "delete-clients", "generate-client-certificates", "read-clients", "update-clients"},
	"commits":                       {"perform-rollback", "read"},
	"secret-scanning-data-sources":  {"create-data-sources", "delete-data-sources", "edit-data-sources", "read-data-source-resources", "read-data-source-scans", "read-data-sources", "trigger-data-source-scans"},
	"secret-scanning-findings":      {"read-findings", "update-findings"},
	"secret-scanning-configs":       {"read-configs", "update-configs"},
	"app-connections":               {"connect-app-connections", "create-app-connections", "delete-app-connections", "edit-app-connections", "read-app-connections", "rotate-credentials"},
	"hsm-connectors":                {"attach-hsm-connectors", "create-hsm-connectors", "delete-hsm-connectors", "edit-hsm-connectors", "read-hsm-connectors", "test-hsm-connectors"},
	"honey-tokens":                  {"create", "edit", "read", "read-credentials", "reset", "revoke"},
	"proxied-services":              {"create", "delete", "edit", "proxy", "read", "report-usage"},
	"agent-vault-access-bundles":    {"create", "delete", "edit", "manage-members", "read"},
	"agent-vault-sessions":          {"create", "read", "revoke"},
	"agent-vault-proxies":           {"create", "delete", "edit", "issue-token", "read", "revoke"},
	"approval-requests":             {"create", "read"},
	"approval-request-grants":       {"read", "revoke"},
	"secret-approval-request":       {"read"},
	"project-folder-grant":          {"create-grant", "read-grant", "revoke-grant"},
}

func validateProjectRoleSpec(role *infisicalv1alpha1.InfisicalProjectRole) error {
	for i, permission := range role.Spec.Permissions {
		subject := string(permission.Subject)
		allowed, ok := projectRoleActionsBySubject[subject]
		if !ok {
			return fmt.Errorf("permissions[%d].subject %q is not supported by the current Infisical API contract", i, subject)
		}
		for j, action := range permission.Action {
			if !containsString(allowed, string(action)) {
				return fmt.Errorf("permissions[%d].action[%d] %q is not valid for subject %q", i, j, action, subject)
			}
		}
		if err := validateProjectRoleConditions(subject, permission.Conditions, i); err != nil {
			return err
		}
	}
	return nil
}

func validateProjectRoleConditions(subject string, conditions *infisicalv1alpha1.ProjectRoleConditions, index int) error {
	if conditions == nil {
		return nil
	}
	secretConditions := conditions.Environment != nil || conditions.SecretPath != nil || conditions.SecretName != nil || conditions.SecretTags != nil
	secretSubjects := strings.Contains("|secrets|secret-folders|secret-imports|dynamic-secrets|secret-rotation|secret-syncs|secret-event-subscriptions|honey-tokens|proxied-services|project-folder-grant|commits|", "|"+subject+"|")
	if secretConditions && !secretSubjects {
		return fmt.Errorf("permissions[%d].conditions has secret resource conditions that are invalid for subject %q", index, subject)
	}
	if (conditions.SecretName != nil || conditions.SecretTags != nil || conditions.EventType != nil) && subject != "secrets" {
		return fmt.Errorf("permissions[%d].conditions contains a secrets-only condition for subject %q", index, subject)
	}
	return nil
}
