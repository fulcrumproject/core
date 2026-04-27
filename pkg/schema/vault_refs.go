// Public helpers for working with "vault://..." references in property maps.
// Counterpart to extractVaultReferences (which is internal to the engine).
package schema

import (
	"context"
	"fmt"
	"strings"
)

// VaultRefPrefix is the scheme used to mark a property value as an opaque
// pointer to a vault-stored secret. The remainder of the string after the
// prefix is the vault reference.
const VaultRefPrefix = "vault://"

// ResolveSecrets walks props according to sch and, for every string value at
// a Secret-marked property that starts with VaultRefPrefix, replaces it with
// the vault-resolved value. Non-secret fields are copied through unchanged.
// The input map is not mutated.
func ResolveSecrets(ctx context.Context, vault Vault, sch Schema, props map[string]any) (map[string]any, error) {
	if props == nil {
		return nil, nil
	}
	return resolvePropsMap(ctx, vault, sch.Properties, props)
}

func resolvePropsMap(ctx context.Context, vault Vault, defs map[string]PropertyDefinition, props map[string]any) (map[string]any, error) {
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

func resolveValue(ctx context.Context, vault Vault, def PropertyDefinition, value any) (any, error) {
	if def.Secret != nil {
		s, ok := value.(string)
		if ok && strings.HasPrefix(s, VaultRefPrefix) {
			ref := strings.TrimPrefix(s, VaultRefPrefix)
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
