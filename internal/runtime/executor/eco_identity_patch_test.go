package executor

import (
	"context"
	"net/http"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v8/internal/buildinfo"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/config"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
)

// These tests pin the ECO identity patch (see ECO-PATCH.md): the gateway never
// presents another client's identity to a provider.

func ecoPatchSignInAuth() *cliproxyauth.Auth {
	return &cliproxyauth.Auth{
		Provider:   "codex",
		Attributes: map[string]string{"auth_kind": "oauth"},
		Metadata:   map[string]any{"account_id": "acc-eco"},
	}
}

func ecoPatchCloakingOnConfig() *config.Config {
	return &config.Config{Codex: config.CodexConfig{DisableCodexCloaking: false}}
}

func TestEcoPatchCodexHTTPNamesGatewayWithoutClientHeaders(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://chatgpt.com/backend-api/codex/responses", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}

	applyCodexHeadersFromSources(req, ecoPatchSignInAuth(), "oauth-token", true, ecoPatchCloakingOnConfig(), http.Header{})

	if got, want := req.Header.Get("User-Agent"), "CLIProxyAPI/"+buildinfo.Version; got != want {
		t.Fatalf("User-Agent = %q, want %q", got, want)
	}
	if got := req.Header.Get("Originator"); got != "" {
		t.Fatalf("Originator = %q, want empty", got)
	}
}

func TestEcoPatchCodexHTTPForwardsClientIdentityWithCloakingConfigured(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://chatgpt.com/backend-api/codex/responses", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	client := http.Header{
		"User-Agent": []string{"codex_cli_rs/0.200.0 (Linux; x86_64)"},
		"Originator": []string{"codex_cli_rs"},
	}

	applyCodexHeadersFromSources(req, ecoPatchSignInAuth(), "oauth-token", true, ecoPatchCloakingOnConfig(), client)

	if got := req.Header.Get("User-Agent"); got != "codex_cli_rs/0.200.0 (Linux; x86_64)" {
		t.Fatalf("User-Agent = %q, want the client's own", got)
	}
	if got := req.Header.Get("Originator"); got != "codex_cli_rs" {
		t.Fatalf("Originator = %q, want the client's own", got)
	}
}

func TestEcoPatchCodexWebsocketNamesGatewayWithoutClientHeaders(t *testing.T) {
	headers := applyCodexWebsocketHeaders(context.Background(), http.Header{}, ecoPatchSignInAuth(), "oauth-token", ecoPatchCloakingOnConfig(), false)

	if got, want := headers.Get("User-Agent"), "CLIProxyAPI/"+buildinfo.Version; got != want {
		t.Fatalf("User-Agent = %q, want %q", got, want)
	}
	if got := headers.Get("Originator"); got != "" {
		t.Fatalf("Originator = %q, want empty", got)
	}
}

func TestEcoPatchCodexWebsocketForwardsClientIdentityWithCloakingConfigured(t *testing.T) {
	ctx := contextWithGinHeaders(map[string]string{
		"User-Agent": "codex_cli_rs/0.200.0 (Linux; x86_64)",
		"Originator": "codex_cli_rs",
	})

	headers := applyCodexWebsocketHeaders(ctx, http.Header{}, ecoPatchSignInAuth(), "oauth-token", ecoPatchCloakingOnConfig(), false)

	if got := headers.Get("User-Agent"); got != "codex_cli_rs/0.200.0 (Linux; x86_64)" {
		t.Fatalf("User-Agent = %q, want the client's own", got)
	}
	if got := headers.Get("Originator"); got != "codex_cli_rs" {
		t.Fatalf("Originator = %q, want the client's own", got)
	}
}

func TestEcoPatchCatalogOverridesSkipIdentityHeaders(t *testing.T) {
	headers := http.Header{}
	headers.Set("User-Agent", "codex_cli_rs/0.200.0 (Linux; x86_64)")
	headers.Set("Originator", "codex_cli_rs")

	// gpt-5.6-luna carries a Codex User-Agent and Originator override in the
	// embedded catalog.
	applyModelHeaderOverrides(headers, "gpt-5.6-luna")

	if got := headers.Get("User-Agent"); got != "codex_cli_rs/0.200.0 (Linux; x86_64)" {
		t.Fatalf("User-Agent = %q, want the client's own", got)
	}
	if got := headers.Get("Originator"); got != "codex_cli_rs" {
		t.Fatalf("Originator = %q, want the client's own", got)
	}
}
