package schema

import (
	"context"
	"errors"
	"testing"
)

func TestResolveSecrets(t *testing.T) {
	sch := Schema{
		Properties: map[string]PropertyDefinition{
			"password": {
				Type:   "string",
				Secret: &SecretConfig{Type: "persistent"},
			},
			"plainNote": {
				Type: "string",
				// Not secret — even a vault://-looking value stays untouched.
			},
			"nested": {
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"apiKey": {
						Type:   "string",
						Secret: &SecretConfig{Type: "persistent"},
					},
				},
			},
			"tokens": {
				Type: "array",
				Items: &PropertyDefinition{
					Type:   "string",
					Secret: &SecretConfig{Type: "persistent"},
				},
			},
		},
	}

	t.Run("resolves secret-marked leaves and leaves the rest alone", func(t *testing.T) {
		ctx := context.Background()
		vault := NewMockVault(t)
		vault.EXPECT().Get(ctx, "pw/1").Return("secret-pw", nil).Once()
		vault.EXPECT().Get(ctx, "api/1").Return("secret-api", nil).Once()
		vault.EXPECT().Get(ctx, "t/1").Return("secret-t1", nil).Once()
		vault.EXPECT().Get(ctx, "t/2").Return("secret-t2", nil).Once()

		props := map[string]any{
			"password":  "vault://pw/1",
			"plainNote": "vault://not-a-secret", // must NOT be resolved
			"nested": map[string]any{
				"apiKey": "vault://api/1",
			},
			"tokens": []any{"vault://t/1", "vault://t/2"},
		}

		resolved, err := ResolveSecrets(ctx, vault, sch, props)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resolved["password"] != "secret-pw" {
			t.Errorf("password: got %v", resolved["password"])
		}
		if resolved["plainNote"] != "vault://not-a-secret" {
			t.Errorf("plainNote was mutated: got %v", resolved["plainNote"])
		}
		nested := resolved["nested"].(map[string]any)
		if nested["apiKey"] != "secret-api" {
			t.Errorf("nested.apiKey: got %v", nested["apiKey"])
		}
		tokens := resolved["tokens"].([]any)
		if tokens[0] != "secret-t1" || tokens[1] != "secret-t2" {
			t.Errorf("tokens: got %v", tokens)
		}
	})

	t.Run("vault error propagates", func(t *testing.T) {
		ctx := context.Background()
		vault := NewMockVault(t)
		boom := errors.New("vault down")
		vault.EXPECT().Get(ctx, "pw/1").Return(nil, boom).Once()
		props := map[string]any{"password": "vault://pw/1"}
		_, err := ResolveSecrets(ctx, vault, sch, props)
		if err == nil || !errors.Is(err, boom) {
			t.Fatalf("want wrapped error %v; got %v", boom, err)
		}
	})

	t.Run("does not mutate input map", func(t *testing.T) {
		ctx := context.Background()
		vault := NewMockVault(t)
		vault.EXPECT().Get(ctx, "pw/1").Return("secret", nil).Once()
		input := map[string]any{"password": "vault://pw/1"}
		_, err := ResolveSecrets(ctx, vault, sch, input)
		if err != nil {
			t.Fatal(err)
		}
		if input["password"] != "vault://pw/1" {
			t.Errorf("input map was mutated: %v", input["password"])
		}
	})

	t.Run("nil props returns nil", func(t *testing.T) {
		vault := NewMockVault(t)
		got, err := ResolveSecrets(context.Background(), vault, sch, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}
