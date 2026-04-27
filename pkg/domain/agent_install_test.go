package domain

import (
	"strings"
	"sync"
	"testing"
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

