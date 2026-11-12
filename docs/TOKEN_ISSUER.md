# Generic Core JWT issuer

Core provides one optional `POST /api/v1/tokens/issue`, shared by applications
such as CEM through Core's implementation. It certifies an authenticated Core
participant or agent, optionally with authorized native service context. It does
**not** grant access to an external application.

Existing `/tokens` CRUD, opaque incoming API credentials, Keycloak authentication,
bootstrap and credential revocation remain unchanged. Issued JWTs do not
authenticate back into Core, even when their requested audience names Core.
They are not stored as Core API credentials. Revoking the incoming credential
prevents further issuance; already issued JWTs remain valid until expiry at
recipients. No introspection, single-use enforcement or rotation platform is
provided.

## Server configuration

Settings are optional when unused, generic and server-owned. No signing settings
means issuance returns 503 for a valid supported principal, public discovery
returns `{"keys":[]}`, and other Core APIs continue to work. Partial or invalid
configuration fails startup before database initialization. A non-default TTL
without signing settings is also an error.

| Environment variable | JSON field under `jwt` | Meaning |
| --- | --- | --- |
| `FULCRUM_JWT_SIGNING_KEY` | `signingKey` | PEM EC P-256 private key, SEC1 or PKCS#8 |
| `FULCRUM_JWT_KEY_ID` | `keyId` | Nonblank signing key ID, published as `kid` |
| `FULCRUM_JWT_ISSUER` | `issuer` | Nonblank issuer recipients explicitly trust |
| `FULCRUM_JWT_PARTICIPANT_TTL` | `participantTtl` | Participant lifetime, default `90s` |
| `FULCRUM_JWT_AGENT_TTL` | `agentTtl` | Agent lifetime, default `5m` |

Environment lifetimes use Go duration syntax; JSON duration values are integer
nanoseconds. Omitted/zero lifetimes select the role default; configured lifetimes
must be positive whole seconds. Lifetimes are not caller-controlled. Keep them
short: recipients validate JWTs offline and cannot observe Core credential
revocation. Supply keys through private operator configuration; never commit,
log or send private signing material to a client. The issuer has no legacy CMP
signing configuration aliases or per-agent application grant map.

`GET /.well-known/jwks.json` is public, at the origin root, outside `/api/v1`.
It exposes one current public EC P-256 key with `use: sig`, `alg: ES256` and `kid`.
It never exposes private material. Replacing the key on restart invalidates old
tokens; overlapping-key rotation is not implemented.

## Request and response

Authenticate with an existing Core credential. Only participant and agent roles
are supported; admin/unknown roles and incomplete or contradictory scopes are
rejected rather than impersonating a supported principal.

```http
POST /api/v1/tokens/issue
Authorization: Bearer <existing-Core-credential>
Content-Type: application/json

{"audience":"recipient-example","serviceId":"44444444-4444-4444-8444-444444444444","ext":{"orderRef":"ORD-42"}}
```

- `audience`: required nonblank string, used as `aud`. Caller selection of an
  audience is **not** evidence of permission to that application.
- `serviceId`: optional nonzero UUID. Core verifies existing native service read
  authorization and persisted provider/consumer relationships. An agent must
  additionally be assigned to that service with a consistent provider. Even
  another service at the same provider is not valid agent context.
- `ext`: optional opaque object, capped at 4096 bytes after JSON serialization.
  It remains **caller-asserted context**, not certified identity, ownership or
  application privileges. It is nested, never flattened. Direct keys `iss`,
  `sub`, `aud`, `iat`, `nbf`, `exp`, `jti`, `role`, `participant_id`, `agent_id`,
  `service_id`, `relation` and `ext` are rejected; nested content is opaque.

Only these fields are accepted. Unknown top-level fields, malformed/trailing JSON,
invalid UUIDs and invalid extensions return 400. Query parameters cannot set
claims or policy. Core does not accept caller-selected principals, roles,
relationships, expiry, signing settings or arbitrary top-level claims.

Successful responses have `Cache-Control: no-store` and **only**:

```json
{"token":"<short-lived-JWT>","expiresAt":"2026-06-22T10:01:30Z"}
```

There is no redirect URL, application profile or module grant. Missing credentials
return 401; invalid credentials or unsupported/inconsistent scopes and denied
service access return 403. Missing persisted principals/services return 404;
repository/audit failures return 500. No JWT is returned on failure.

## Certified claims

Registered claims are server-owned `iss`, `sub`, `iat`, `nbf`, `exp` and a fresh
UUID `jti`; `aud` contains the requested audience. `iat` and `nbf` are issuance
time; `exp` uses server caller-role lifetime policy. `sub` is the **domain
principal UUID**, not a credential-record ID or Keycloak user subject.

| Claim | Participant | Agent |
| --- | --- | --- |
| `role` | `participant` | `agent` |
| `sub` | Persisted participant UUID | Persisted agent UUID |
| `participant_id` | Authenticated persisted participant UUID | Persisted agent's provider UUID, checked against authenticated scope |
| `agent_id` | Absent | Persisted agent UUID |
| `service_id` | Only when requested and authorized | Only when requested, authorized and assigned to this agent |
| `relation` | Only with service: `provider` or `consumer` from Core data | Only with service: `provider`, consistent with agent/service data |
| `ext` | Nonempty supplied caller-asserted object only | Nonempty supplied caller-asserted object only |

Provider relation takes precedence if a participant is both provider and consumer
of the same service. Without service context, neither `service_id` nor `relation`
is included. Core loads the persisted participant for both caller kinds, and the
persisted agent/provider linkage for agents. Raw request IDs and `ext` prove no
ownership. No `app`, external `roles` or module permissions are issued.

A mandatory `token.issued` event records the authenticated initiator, persisted
principal/resource scope, Core role, requested audience and service relation when
present. Audit contains no JWT, extensions or private key. Audit failure prevents
returning the JWT.

## Recipient and frontend responsibilities

Recipients must validate ES256 signature using the trusted issuer's public JWKS,
issuer, intended audience, time, Core role and applicable resource scope, then
apply their **own** authorization policy. A recipient unable to enforce an
included service restriction must reject it, never widen access. Treat `ext` as
untrusted application input, not Core-certified privileges. FAE must map trusted
agent/provider identity to its own module permissions. The old `app`/module-role
proposal is superseded; real FAE agreement on this generic contract, audience
and transport remains pending. Mock success is not evidence of compatibility.

Core neither interprets application routing metadata nor selects a recipient URL.
CEM's separate frontend migration can read ordinary opaque agent configuration
from `GET /api/v1/services/{id}`, validate its routing metadata, request a generic
JWT with audience and service ID, then compose its existing hidden POST form.
Only the short-lived JWT belongs in that form, never the long-lived Core
credential. That frontend migration and remaining agent/CMP/mock implementation
are not part of this Core change.

## Shared wiring and verification

Build `jwtissuer.New(cfg.JWT)` once. Mount
`api.NewTokenIssueHandler(domain.NewTokenIssueCommander(store, authorizer), signer).Routes()`
under the authenticated `/tokens` group alongside token CRUD. The static `/issue`
route does not go through the CRUD `/{id}` UUID middleware. Mount
`api.NewJwksHandler(signer).JWKS` publicly at `/.well-known/jwks.json`.
The command uses existing service read authorization; it adds no external
application policy engine. `App.JWTSigner` and `App.TokenIssueHandler` expose the
same generic seam for embedding applications.

Focused tests: `go test -race ./pkg/jwtissuer ./pkg/config ./pkg/domain ./pkg/api`.
The e2e-tagged `TestJWTInvalidBoot` and `TestJWTIssueRoute` run real Core processes
with synthetic PostgreSQL and Keycloak fixtures, alongside the existing full
E2E suite. Test only isolated resources; do not use personal/shared databases.
