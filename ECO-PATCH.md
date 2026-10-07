# ECO identity patch

This fork carries one patch on top of an upstream CLIProxyAPI release. It
exists so the gateway never presents another client's identity to a
provider. Upstream builds do that in places that no setting turns off.

It is the interim gateway for ECO: see `plan/18A-INTERIM-GATEWAY-SETUP.md`
in the ECO repository.

## What it changes

| Upstream behaviour | Patched behaviour | Where |
|---|---|---|
| Requests to `api.anthropic.com` use a TLS handshake copied from Claude Code; requests to `chatgpt.com` use one copied from Chrome. | Every upstream uses Go's standard TLS, or the configured proxy transport. The gateway connects as itself. | `internal/runtime/executor/helps/utls_client.go` (`NewUtlsHTTPClient`) |
| A Codex request without a `User-Agent` gets Codex's own. | It gets `CLIProxyAPI/<version>`. | `internal/runtime/executor/codex_executor_request.go`, `codex_websockets_request.go` |
| A Codex sign-in request without an `Originator` gets `codex-tui`. The alpha-search route always sends `codex_cli_rs`, and the live relay sends `Codex Desktop`. | No `Originator` is invented; the client's own is forwarded. | the two files above, `internal/api/server_routes.go`, `internal/client/codex/live/websocket.go` |
| The model catalog forces Codex's `User-Agent` and `Originator` for some models (`gpt-5.6-luna` in v8.0.17). | Catalog overrides never set `User-Agent` or `Originator`; other override headers still apply. | `codex_executor_request.go` (`applyModelHeaderOverrides`) |
| Codex cloaking (`upstream.codex.disable-codex-cloaking`) replaces `User-Agent` and `Originator` unless it is configured off. | Codex cloaking never applies, whatever the configuration. | `codex_executor_request.go` (`applyCodexCloakingHeaders`) |
| A background job fetches an Antigravity version manifest from a third-party URL every three hours, under another client's `User-Agent`. | It does not run. Antigravity requests keep the built-in fallback version. | `cmd/server/main.go` |

A client's own `User-Agent`, `Originator` and session headers still pass
through unchanged. Headers the operator sets explicitly (`header-defaults`,
or `headers` on a credential) still apply; ECO's configuration sets none.

**Not changed:**
- **The Claude executor.** Its Anthropic path also rewrites some of Claude
  Code's own headers, so ECO does not route Anthropic through this gateway
  at all. The Claude sign-in flow (`--claude-login`) is never used.
- **The Antigravity, Gemini CLI and other sign-in paths.** ECO does not use
  them.

## Tests

- `internal/runtime/executor/eco_identity_patch_test.go` and
  `internal/runtime/executor/helps/eco_identity_patch_test.go` pin each
  change.
- Upstream tests that asserted the replaced behaviour are adjusted in the
  same commit. Each adjustment carries an `ECO patch` comment.

## Branches and tags

- `main` follows upstream unchanged.
- `eco/v<version>` is the upstream release tag plus this patch, as one
  commit.
- `v<version>-eco.<n>` tags a patched build.

## Rebasing onto a new upstream release

```
git remote add upstream https://github.com/router-for-me/CLIProxyAPI.git   # once
git fetch upstream --tags
git checkout -b eco/v<new> v<new>
git cherry-pick <the patch commit from the previous eco/ branch>
go test ./...
```

Before tagging, check the new release for new identity behaviour:

```
grep -rn "utls\|UClient\|HelloChrome\|override_header\|codexUserAgent\|codexOriginator\|\"Originator\"\|StartAntigravityVersionUpdater" \
  --include=*.go --include=*.json cmd internal sdk
```

## Building

```
VERSION=<version>-eco.<n>
CGO_ENABLED=1 go build \
  -ldflags "-s -w -X main.Version=$VERSION -X main.Commit=$(git rev-parse --short HEAD) -X main.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o cli-proxy-api ./cmd/server
```

Or build the image from the repository's `Dockerfile` with
`--build-arg VERSION=$VERSION`.
