package helps

import (
	"context"
	"net/http"
	"testing"

	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
)

// TestEcoPatchProtectedHostsUseStandardTransport pins the ECO identity patch
// (see ECO-PATCH.md): Anthropic and ChatGPT hosts use the standard transport,
// not round trippers that copy Claude Code's or Chrome's TLS handshake.
func TestEcoPatchProtectedHostsUseStandardTransport(t *testing.T) {
	client := NewUtlsHTTPClient(context.Background(), nil, nil, 0)

	routes, ok := client.Transport.(*fallbackRoundTripper)
	if !ok {
		t.Fatalf("Transport = %T, want *fallbackRoundTripper", client.Transport)
	}
	if routes.anthropic != http.DefaultTransport {
		t.Fatalf("anthropic route = %T, want http.DefaultTransport", routes.anthropic)
	}
	if routes.chrome != http.DefaultTransport {
		t.Fatalf("chrome route = %T, want http.DefaultTransport", routes.chrome)
	}
	if routes.fallback != http.DefaultTransport {
		t.Fatalf("fallback route = %T, want http.DefaultTransport", routes.fallback)
	}
}

// TestEcoPatchProtectedHostsUseProxyTransport checks that with a configured
// proxy, Anthropic and ChatGPT traffic takes the same standard proxy transport
// as every other host.
func TestEcoPatchProtectedHostsUseProxyTransport(t *testing.T) {
	auth := &cliproxyauth.Auth{ProxyURL: "http://127.0.0.1:3128"}
	client := NewUtlsHTTPClient(context.Background(), nil, auth, 0)

	routes, ok := client.Transport.(*fallbackRoundTripper)
	if !ok {
		t.Fatalf("Transport = %T, want *fallbackRoundTripper", client.Transport)
	}
	if routes.fallback == http.DefaultTransport {
		t.Fatal("fallback route ignores the configured proxy")
	}
	if routes.anthropic != routes.fallback || routes.chrome != routes.fallback {
		t.Fatalf("routes differ: anthropic=%T chrome=%T fallback=%T", routes.anthropic, routes.chrome, routes.fallback)
	}
}
