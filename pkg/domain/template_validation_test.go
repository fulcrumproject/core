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

func TestTemplateValidation_RenderCmdTemplate(t *testing.T) {
	tests := []struct {
		name      string
		tv        TemplateValidation
		data      map[string]any
		url       string
		authToken string
		want      string
		wantErr   string
	}{
		{
			name: "empty template returns empty",
			tv:   TemplateValidation{CmdTemplate: ""},
			want: "",
		},
		{
			name: "happy path with configUrl, authToken and property",
			tv: TemplateValidation{
				CmdTemplate: "curl -H 'Authorization: Bearer {{.authToken}}' {{.configUrl}} > /etc/{{.name}}.conf",
			},
			data:      map[string]any{"name": "svc"},
			url:       "http://host/install/tok",
			authToken: "bearer-123",
			want:      "curl -H 'Authorization: Bearer bearer-123' http://host/install/tok > /etc/svc.conf",
		},
		{
			name:    "unknown reference surfaces missingkey error",
			tv:      TemplateValidation{CmdTemplate: "{{.nope}}"},
			wantErr: "nope",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.tv.RenderCmdTemplate(tc.data, tc.url, tc.authToken)
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

func TestTemplateValidation_RenderCmdTemplate_DoesNotMutateCaller(t *testing.T) {
	tv := TemplateValidation{
		CmdTemplate: "{{.configUrl}}-{{.authToken}}-{{.k}}",
	}
	data := map[string]any{"k": "v"}
	if _, err := tv.RenderCmdTemplate(data, "URL", "TOKEN"); err != nil {
		t.Fatal(err)
	}
	if _, has := data["configUrl"]; has {
		t.Errorf("caller's map was mutated with configUrl")
	}
	if _, has := data["authToken"]; has {
		t.Errorf("caller's map was mutated with authToken")
	}
}

func TestTemplateValidation_RenderConfigTemplate(t *testing.T) {
	tests := []struct {
		name    string
		tv      TemplateValidation
		data    map[string]any
		want    string
		wantErr string
	}{
		{
			name: "empty template returns empty",
			tv:   TemplateValidation{ConfigTemplate: ""},
			want: "",
		},
		{
			name: "happy path",
			tv:   TemplateValidation{ConfigTemplate: "name={{.name}}"},
			data: map[string]any{"name": "svc"},
			want: "name=svc",
		},
		{
			name:    "configUrl is not injected into config template",
			tv:      TemplateValidation{ConfigTemplate: "{{.configUrl}}"},
			data:    map[string]any{},
			wantErr: "configUrl",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.tv.RenderConfigTemplate(tc.data)
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

// TestTemplateValidation_EmbeddedPromotion verifies the helpers are reachable
// via embedded promotion from both AgentType and InfrastructureType, so a
// single TemplateValidation method body serves both entity-type install flows.
func TestTemplateValidation_EmbeddedPromotion(t *testing.T) {
	tv := TemplateValidation{
		CmdTemplate:    "curl -H 'Authorization: Bearer {{.authToken}}' {{.configUrl}}",
		ConfigTemplate: "name={{.name}}",
	}
	data := map[string]any{"name": "svc"}
	wantCmd := "curl -H 'Authorization: Bearer t' http://u"
	wantCfg := "name=svc"

	at := &AgentType{TemplateValidation: tv}
	it := &InfrastructureType{TemplateValidation: tv}

	for _, tc := range []struct {
		name           string
		hasInstall     bool
		gotCmd, gotCfg string
		cmdErr, cfgErr error
	}{
		{
			name:       "agent type",
			hasInstall: at.HasInstallTemplates(),
		},
		{
			name:       "infrastructure type",
			hasInstall: it.HasInstallTemplates(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.hasInstall {
				t.Fatalf("HasInstallTemplates() should be true")
			}
		})
	}

	gotCmd, err := at.RenderCmdTemplate(data, "http://u", "t")
	if err != nil || gotCmd != wantCmd {
		t.Errorf("AgentType.RenderCmdTemplate: got %q, err %v; want %q", gotCmd, err, wantCmd)
	}
	gotCfg, err := at.RenderConfigTemplate(data)
	if err != nil || gotCfg != wantCfg {
		t.Errorf("AgentType.RenderConfigTemplate: got %q, err %v; want %q", gotCfg, err, wantCfg)
	}
	gotCmd, err = it.RenderCmdTemplate(data, "http://u", "t")
	if err != nil || gotCmd != wantCmd {
		t.Errorf("InfrastructureType.RenderCmdTemplate: got %q, err %v; want %q", gotCmd, err, wantCmd)
	}
	gotCfg, err = it.RenderConfigTemplate(data)
	if err != nil || gotCfg != wantCfg {
		t.Errorf("InfrastructureType.RenderConfigTemplate: got %q, err %v; want %q", gotCfg, err, wantCfg)
	}
}
