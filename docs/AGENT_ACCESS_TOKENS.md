# Agent application access tokens

Core can issue short-lived **application/module** JWTs to authenticated agents.
This is separate from [browser CMP launch tokens](CMP_SSO.md), not OAuth, a new
Core credential, or a per-service/VDC permission system.

## Operator policy and shared signing

Use the existing `cmpSso` server signing settings (PEM EC P-256 key, key ID and
issuer). Both profiles use that one ES256 signer and public root
`GET /.well-known/jwks.json`; no second key, browser-claim shortcut, overlapping
key rotation or introspection endpoint is added. Never expose the private key.

The **Core operator**, not a provider/participant, supplies a grant map in Core's
existing JSON `-config /path/to/core.json` file. For example:

```json
{
  "applicationTokens": {
    "grants": {
      "33333333-3333-4333-8333-333333333333": {
        "audience": "fae-demo",
        "roles": ["pve"]
      }
    }
  }
}
```

Each key must be a canonical, nonzero agent UUID. Each grant requires one
nonblank audience and a nonempty list of nonblank module-role strings. These
strings are generic; Core has no hard-coded module such as `pve`. Grants are
never read from agent configuration or metadata: participants can register and
update their own agents, but cannot create application grants through Core APIs.
Protect the server configuration file and its deployment mechanism accordingly.
The existing configuration builder supports JSON maps, not environment maps.

`FULCRUM_APPLICATION_TOKEN_TTL` uses Go duration syntax, default `5m`. Its JSON
counterpart is `applicationTokens.tokenTtl` in integer nanoseconds. Omitted/zero
uses the default; otherwise the allowed range is 1 second to 15 minutes. It does
not change `FULCRUM_CMP_SSO_TOKEN_TTL` / `cmpSso.tokenTtl` (still 90 seconds).
Invalid grants/TTLs, grants without signing, non-default TTL without signing,
and partial/invalid signing settings fail at boot **before database access**.
All signing and grant settings may be omitted: existing APIs still boot, public
JWKS is empty, and issuance for an otherwise valid agent returns 503. With
signing configured but no grant for the caller, issuance returns 403.

Policy is copied and validated once at startup. Grant, audience, role, TTL or key
changes require a Core restart. A demo bootstrap can register its agent, write
its own operator configuration, then restart only its own Core stack.

## Endpoint and wire profile

`POST /api/v1/agents/me/access-token` uses Core's existing incoming
`Authorization: Bearer <Core agent credential>` authentication (opaque tokens
or the existing Keycloak chain). Participant/admin credentials are denied, even
if they contain agent scope. The agent UUID comes from authenticated
`identity.Scope.AgentID`, **not** the incoming token record's `identity.ID`.
Core loads that agent and requires its persisted ID/provider to match the
credential scope. It never accepts another agent ID from the request.

No request body is needed. **All bodies and query parameters are ignored**, even
malformed JSON and unsolicited `app`, `roles`, `aud`, `exp`, issuer or signing
settings. Content-Type is not required; there is no claim override decoder.

HTTP 200 has `Cache-Control: no-store` and camelCase JSON:

```json
{"token":"<signed JWT>","expiresAt":"2026-06-22T10:05:00Z"}
```

The protected header is `typ: JWT`, `alg: ES256`, and the shared configured
`kid`. The payload contains only:

- `app`: authenticated agent UUID;
- `roles`: module-role list from its operator grant;
- `iss`: shared signing issuer; `aud`: the grant's single audience;
- `iat`: issue time; `exp`: issue time plus application TTL.

No browser `sub`, `participant_id`, `service_id`, `relation`, `ext`, `nbf` or
`jti` is added. There is no tenant, human identity, IP claim or per-VDC permission.
The recipient must validate ES256 against matching-kid public JWKS, trusted
issuer, intended audience, expiry and required module role. These JWTs are
**not accepted as Core API credentials**; incoming authentication is unchanged.

Missing credentials return 401; invalid credentials, non-agent/missing/mismatched
scope or absent grants return 403. A missing persisted agent returns 404;
disabled signing returns 503. Signing, database or mandatory audit failures
return 500 without a token. Before returning success, Core persists
`agent.access_token_issued` through the domain commander, scoped to the agent
and provider with credential initiator. The payload contains only trusted
`audience` and `roles`, never the JWT, private key or request body.

## Trust and revocation limits

Removing the operator grant prevents new issuance after restart; deleting the
agent or revoking its Core credential also prevents new issuance. Existing
participant/admin authority to mint agent Core credentials is unchanged: keep
those credentials and provider accounts secure. Heartbeat/connectivity status
(including `Disabled`) is not an application revocation mechanism and is not
consulted by issuance. Tokens already returned remain valid until expiry unless
the recipient applies its own revocation policy. Replacing the one signing key
at restart can invalidate both profiles' in-flight tokens. No single-use or
refresh-token protocol is provided.

The initial FAE proposal is `app=<agent UUID>`, `roles:["pve"]`, five-minute TTL,
no `ip`, and a deployment-specific audience. Raw `Authorization: <JWT>` versus
Bearer, IP omission, audience and TTL remain **unconfirmed by the FAE team**.
Core only emits the token; these tests are not evidence of real FAE compatibility.

## Reuse and checks

Build `sso.New` once and share it with
`sso.NewApplicationTokens(signer, sso.ApplicationConfig)`. Mount
`api.NewAgentAccessTokenHandler(agentQuerier,
 domain.NewAgentAccessTokenCommander(store), authorizer, applicationTokens).Routes()`
under an **authenticated** `/api/v1/agents` router. Include Core's agent-only
`authz.ActionAccessToken` rule and public `api.NewSsoHandler(signer).JWKS` at the
root. `App.ApplicationTokens` and `App.AgentAccessTokenHandler` expose Core's
initialized instances. CEM has not yet adopted these exports.

Focused checks: `go test -race ./pkg/sso ./pkg/config ./pkg/api ./pkg/authz`.
Real-server checks: `go test -tags e2e -run
'TestApplicationInvalidBoot|TestAgentApplicationTokens|TestCmpSso' ./test/e2e`.
Use isolated synthetic PostgreSQL and the test Keycloak realm as in CI, never a
shared database or real FAE credentials.
