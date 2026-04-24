package domain

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"text/template"

	"github.com/fulcrumproject/core/pkg/schema"
)

const vaultRefPrefix = "vault://"

// GenerateInstallToken returns a 32-byte base64url-encoded secret.
func GenerateInstallToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate install token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}

// BuildInstallURL joins the public base URL with the install token path.
// The endpoint is authenticated: callers must also supply a Bearer token
// (issued alongside the install token — see AgentInstallToken.BootstrapTokenID).
func BuildInstallURL(publicBaseURL, token string) string {
	return strings.TrimRight(publicBaseURL, "/") + "/api/v1/agents/install/" + token
}

// RenderCmdTemplate renders the AgentType's CmdTemplate with schemaData plus
// configUrl and authToken. The authToken is the plain value of the bootstrap
// bearer token and is expected to appear in the template's Authorization header.
// Returns "" when CmdTemplate is empty.
func RenderCmdTemplate(at *AgentType, schemaData map[string]any, configURL, authToken string) (string, error) {
	if at.CmdTemplate == "" {
		return "", nil
	}
	data := copyDataMap(schemaData)
	data[cmdTemplateExtraRef] = configURL
	data[cmdTemplateExtraAuthTokenRef] = authToken
	return renderTemplate("cmdTemplate", at.CmdTemplate, data)
}

// RenderConfigTemplate renders the AgentType's ConfigTemplate with schemaData.
// Returns "" when ConfigTemplate is empty.
func RenderConfigTemplate(at *AgentType, schemaData map[string]any) (string, error) {
	if at.ConfigTemplate == "" {
		return "", nil
	}
	return renderTemplate("configTemplate", at.ConfigTemplate, schemaData)
}

func renderTemplate(name, body string, data map[string]any) (string, error) {
	tmpl, err := template.New(name).Option("missingkey=error").Parse(body)
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return buf.String(), nil
}

func copyDataMap(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src)+1)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// ResolveVaultRefs walks `props` according to `sch` and, for every string value
// at a Secret-marked property that starts with "vault://", replaces it with the
// vault-resolved value. Non-secret fields are copied through unchanged. The
// input map is not mutated.
func ResolveVaultRefs(ctx context.Context, vault schema.Vault, sch schema.Schema, props map[string]any) (map[string]any, error) {
	if props == nil {
		return nil, nil
	}
	return resolvePropsMap(ctx, vault, sch.Properties, props)
}

func resolvePropsMap(ctx context.Context, vault schema.Vault, defs map[string]schema.PropertyDefinition, props map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(props))
	for key, value := range props {
		def, hasDef := defs[key]
		if !hasDef {
			out[key] = value
			continue
		}
		resolved, err := resolveValue(ctx, vault, def, value)
		if err != nil {
			return nil, fmt.Errorf("resolve %s: %w", key, err)
		}
		out[key] = resolved
	}
	return out, nil
}

func resolveValue(ctx context.Context, vault schema.Vault, def schema.PropertyDefinition, value any) (any, error) {
	if def.Secret != nil {
		s, ok := value.(string)
		if ok && strings.HasPrefix(s, vaultRefPrefix) {
			ref := strings.TrimPrefix(s, vaultRefPrefix)
			resolved, err := vault.Get(ctx, ref)
			if err != nil {
				return nil, err
			}
			return resolved, nil
		}
		return value, nil
	}

	switch def.Type {
	case "object":
		nested, ok := value.(map[string]any)
		if !ok || def.Properties == nil {
			return value, nil
		}
		return resolvePropsMap(ctx, vault, def.Properties, nested)
	case "array":
		items, ok := value.([]any)
		if !ok || def.Items == nil {
			return value, nil
		}
		out := make([]any, len(items))
		for i, item := range items {
			resolved, err := resolveValue(ctx, vault, *def.Items, item)
			if err != nil {
				return nil, fmt.Errorf("[%d]: %w", i, err)
			}
			out[i] = resolved
		}
		return out, nil
	}
	return value, nil
}
