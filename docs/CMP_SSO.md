# Service-scoped CMP SSO

Core can sign short-lived browser launch tokens for a service's Cloud Management
Platform (CMP). This is the generic CEM launch flow, not a new login system.
Incoming API authentication remains the existing token/Keycloak chain. Core's
launch JWTs are **not** accepted as Core API credentials. No tenant/end-customer
model, session management, single-use/replay policy or real FAE integration is
implemented here. The separate [agent application token](AGENT_ACCESS_TOKENS.md)
profile shares the signer/JWKS but not browser claims or TTL.

## Server configuration

Signing is optional. Leave all three key/identity settings unset to keep it
disabled; Core boots and serves its existing APIs. Public discovery then returns
`{"keys":[]}` and an otherwise authorized launch returns HTTP 503.

| Environment variable | JSON field under `cmpSso` | Value |
| --- | --- | --- |
| `FULCRUM_CMP_SSO_SIGNING_KEY` | `signingKey` | Server-owned PEM EC P-256 private key (SEC1 or PKCS#8) |
| `FULCRUM_CMP_SSO_KEY_ID` | `keyId` | Non-empty key identifier, published as `kid` |
| `FULCRUM_CMP_SSO_ISSUER` | `issuer` | Non-empty issuer the CMP explicitly trusts |
| `FULCRUM_CMP_SSO_TOKEN_TTL` | `tokenTtl` | Duration, default `90s` |

Environment durations use Go duration syntax; JSON durations are integer
nanoseconds, as for other Core configuration. Zero/omitted TTL uses 90 seconds;
a negative TTL is invalid. A non-default TTL without signing settings is partial
configuration and fails. Setting any key/identity field requires all three;
invalid/partial settings or unusable/non-P-256 keys fail at startup, before
connecting to the database. RSA, other EC curves and multiple PEM blocks are
rejected. No algorithm is caller-selectable: issuance uses ES256 only.

Generate a development key with `openssl ecparam -name prime256v1 -genkey -noout`
and supply its complete PEM through your server's secret configuration. Never
commit, log or send the private key to a caller/CMP. The CMP needs only the
public `GET /.well-known/jwks.json` endpoint (outside `/api/v1` and without
Authorization). It publishes one public EC P-256 key with `kid`, `alg: ES256`
and `use: sig`. No overlapping-key rotation is implemented: replacing the key
at restart can invalidate in-flight tokens for up to the configured TTL.

## Trusted CMP target

The service's persisted agent configuration supplies these existing CEM fields:

```json
{
  "cmpLandingUrl": "https://cmp.example/sso/landing",
  "cmpAudience": "cmp-example"
}
```

Both must be non-empty strings. Define them in the agent type's
`configurationSchema`, with validators restricting the landing URL to the
administrator-approved HTTPS destination and the audience to the intended CMP.
As in CEM, the handler relies on that upstream schema; it does not implement a
new URL allowlist. Participants able to manage agent configuration must be
trusted to configure that agent's CMP. Neither the request body nor `ext`
selects a destination or overrides the configured audience.

## Launch API and claims

`POST /api/v1/services/{id}/cmp-launch-token` requires a participant credential
whose participant is the service's provider or consumer. Admin and agent roles
are denied (even when an agent credential carries the provider's participant
scope). Provider wins if both relations refer to the same participant.

Send `Content-Type: application/json` and at least `{}`. The real CEM decoder
behavior is preserved: an empty/malformed body is HTTP 400; JSON `null` is
accepted as an empty request; unknown top-level fields are ignored. The OpenAPI
request body is required to reflect this actual behavior, correcting CEM's
optional-body documentation. Optional `ext` must be an object (or null):

```json
{"ext":{"orderRef":"example-42"}}
```

`ext` is opaque marketplace-asserted data, not verified customer identity.
Its serialized JSON must be at most 4096 bytes. Its immediate keys cannot be
`iss`, `sub`, `aud`, `exp`, `nbf`, `iat`, `jti`, `participant_id`, `service_id`,
`relation` or `ext`. Nested contents remain opaque. Violations return HTTP 400.

The HTTP 200 response (not cacheable) contains `redirectUrl`, `token` and
`expiresAt` (RFC 3339). The browser submits `token` to `redirectUrl` by POST;
Core does not redirect the browser or append the token to a URL.

The JWT protected header contains `typ: JWT`, `alg: ES256` and the configured
`kid`. Claims retain the CEM names and semantics:

- `iss`: configured issuer; `aud`: configured CMP audience.
- `sub` and `participant_id`: authenticated participant UUID.
- `service_id`: persisted service UUID; `relation`: `provider` or `consumer`.
- `iat` and `nbf`: issuance time; `exp`: issuance plus TTL; `jti`: fresh UUID.
- `ext`: optional opaque extension object.

The CMP must validate ES256 signature against the matching public key, trusted
issuer, expected audience, `nbf` and `exp`, and restrict access to the signed
service. A fresh `jti` is not a server-side single-use guarantee.

Missing credentials return 401; denied roles/ownership return 403. Missing
services normally also return 403 because Core's existing authorization
middleware cannot load their scope (not CEM's documented 404). Missing CMP
configuration returns 422, disabled signing returns 503, and signing/audit
persistence failures return 500 using Core's standard error body. Successful
issuance must persist `service.cmp_launch` before returning the token; the event
has service/provider/consumer/agent scope, credential initiator, and only
`relation`/`audience` in its payload. Tokens, keys and `ext` are not audited.

## Reuse

Applications can build `sso.New(sso.Config)` once, share its `Signer.Sign`,
`LaunchClaims`, `ValidateExt` and public `JWKS`, and mount
`api.NewSsoHandler(signer).JWKS` on their root router. `New` returns nil without
error when disabled; nil signers expose empty JWKS and reject signing.

Mount `api.NewCmpLaunchHandler(serviceQuerier, agentQuerier,
 domain.NewCmpLaunchCommander(store), authorizer, signer).Routes()` under the
application's **authenticated** `/api/v1/services` router, alongside its own
service routes. Include Core's `authz.ActionCmpLaunch` rule (participant-only).
Core's `App.SsoSigner` and `App.CmpLaunchHandler` expose the same initialized
instances for extension wiring. There is no duplicate CEM implementation in
Core; a later CEM change must actually adopt these exports before downstream
reuse can be claimed. The separate agent application profile is documented in
[Agent application access tokens](AGENT_ACCESS_TOKENS.md).

## Verification

Focused tests: `go test -race ./pkg/sso ./pkg/api ./pkg/authz ./pkg/config`.
The `e2e`-tagged `TestCmpSso` uses synthetic PostgreSQL fixtures, a running Core
binary and the existing test Keycloak realm. It exercises enabled/disabled
boot, public JWKS verification, provider/consumer launches, denied agent/admin
access, retained external authentication and persisted audit. See CI for the
PostgreSQL/Keycloak provisioning and full-suite commands; never run database
tests against a shared or real environment.
