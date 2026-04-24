package domain

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/fulcrumproject/core/pkg/schema"
)

func TestGenerateSecureToken_Unique(t *testing.T) {
	const n = 256
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		seen = make(map[string]struct{}, n)
	)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			tok, err := generateSecureToken()
			if err != nil {
				t.Errorf("generateSecureToken: %v", err)
				return
			}
			mu.Lock()
			seen[tok] = struct{}{}
			mu.Unlock()
		}()
	}
	wg.Wait()
	if len(seen) != n {
		t.Errorf("expected %d unique tokens, got %d", n, len(seen))
	}
}

func TestGenerateSecureToken_Length(t *testing.T) {
	tok, err := generateSecureToken()
	if err != nil {
		t.Fatalf("generateSecureToken: %v", err)
	}
	// 32 bytes → base64url of length 44 (with padding).
	if len(tok) != 44 {
		t.Errorf("unexpected token length %d; token=%q", len(tok), tok)
	}
}

func TestBuildInstallURL(t *testing.T) {
	tests := []struct {
		base, token, want string
	}{
		{"http://localhost:8080", "abc", "http://localhost:8080/api/v1/agents/install/abc/config"},
		{"http://localhost:8080/", "abc", "http://localhost:8080/api/v1/agents/install/abc/config"},
		{"https://fulcrum.example.com///", "xyz", "https://fulcrum.example.com/api/v1/agents/install/xyz/config"},
	}
	for _, tc := range tests {
		got := BuildInstallURL(tc.base, tc.token)
		if got != tc.want {
			t.Errorf("BuildInstallURL(%q,%q) = %q; want %q", tc.base, tc.token, got, tc.want)
		}
	}
}

func TestRenderCmdTemplate(t *testing.T) {
	tests := []struct {
		name      string
		at        *AgentType
		data      map[string]any
		url       string
		authToken string
		want      string
		wantErr   string
	}{
		{
			name: "empty template returns empty",
			at:   &AgentType{CmdTemplate: ""},
			want: "",
		},
		{
			name:      "happy path with configUrl, authToken and property",
			at:        &AgentType{CmdTemplate: "curl -H 'Authorization: Bearer {{.authToken}}' {{.configUrl}} > /etc/{{.name}}.conf"},
			data:      map[string]any{"name": "svc"},
			url:       "http://host/install/tok",
			authToken: "bearer-123",
			want:      "curl -H 'Authorization: Bearer bearer-123' http://host/install/tok > /etc/svc.conf",
		},
		{
			name:    "unknown reference surfaces missingkey error",
			at:      &AgentType{CmdTemplate: "{{.nope}}"},
			wantErr: "nope",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RenderCmdTemplate(tc.at, tc.data, tc.url, tc.authToken)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("want error containing %q; got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q; want %q", got, tc.want)
			}
		})
	}
}

func TestRenderCmdTemplate_DoesNotMutateCaller(t *testing.T) {
	at := &AgentType{CmdTemplate: "{{.configUrl}}-{{.authToken}}-{{.k}}"}
	data := map[string]any{"k": "v"}
	if _, err := RenderCmdTemplate(at, data, "URL", "TOKEN"); err != nil {
		t.Fatal(err)
	}
	if _, has := data["configUrl"]; has {
		t.Errorf("caller's map was mutated with configUrl")
	}
	if _, has := data["authToken"]; has {
		t.Errorf("caller's map was mutated with authToken")
	}
}

func TestRenderConfigTemplate(t *testing.T) {
	tests := []struct {
		name    string
		at      *AgentType
		data    map[string]any
		want    string
		wantErr string
	}{
		{
			name: "empty template returns empty",
			at:   &AgentType{ConfigTemplate: ""},
			want: "",
		},
		{
			name: "happy path",
			at:   &AgentType{ConfigTemplate: "name={{.name}}"},
			data: map[string]any{"name": "svc"},
			want: "name=svc",
		},
		{
			name:    "configUrl is not injected into config template",
			at:      &AgentType{ConfigTemplate: "{{.configUrl}}"},
			data:    map[string]any{},
			wantErr: "configUrl",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RenderConfigTemplate(tc.at, tc.data)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("want error containing %q; got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q; want %q", got, tc.want)
			}
		})
	}
}

func TestResolveVaultRefs(t *testing.T) {
	sch := schema.Schema{
		Properties: map[string]schema.PropertyDefinition{
			"password": {
				Type:   "string",
				Secret: &schema.SecretConfig{Type: "persistent"},
			},
			"plainNote": {
				Type: "string",
				// Not secret — even a vault://-looking value stays untouched.
			},
			"nested": {
				Type: "object",
				Properties: map[string]schema.PropertyDefinition{
					"apiKey": {
						Type:   "string",
						Secret: &schema.SecretConfig{Type: "persistent"},
					},
				},
			},
			"tokens": {
				Type: "array",
				Items: &schema.PropertyDefinition{
					Type:   "string",
					Secret: &schema.SecretConfig{Type: "persistent"},
				},
			},
		},
	}

	t.Run("resolves secret-marked leaves and leaves the rest alone", func(t *testing.T) {
		ctx := context.Background()
		vault := schema.NewMockVault(t)
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

		resolved, err := ResolveVaultRefs(ctx, vault, sch, props)
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
		vault := schema.NewMockVault(t)
		boom := errors.New("vault down")
		vault.EXPECT().Get(ctx, "pw/1").Return(nil, boom).Once()
		props := map[string]any{"password": "vault://pw/1"}
		_, err := ResolveVaultRefs(ctx, vault, sch, props)
		if err == nil || !errors.Is(err, boom) {
			t.Fatalf("want wrapped error %v; got %v", boom, err)
		}
	})

	t.Run("does not mutate input map", func(t *testing.T) {
		ctx := context.Background()
		vault := schema.NewMockVault(t)
		vault.EXPECT().Get(ctx, "pw/1").Return("secret", nil).Once()
		input := map[string]any{"password": "vault://pw/1"}
		_, err := ResolveVaultRefs(ctx, vault, sch, input)
		if err != nil {
			t.Fatal(err)
		}
		if input["password"] != "vault://pw/1" {
			t.Errorf("input map was mutated: %v", input["password"])
		}
	})

	t.Run("nil props returns nil", func(t *testing.T) {
		vault := schema.NewMockVault(t)
		got, err := ResolveVaultRefs(context.Background(), vault, sch, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}
