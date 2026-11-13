# One-time administrative bootstrap

Core is under development, not production-ready. On a genuinely fresh, empty
Core database, the first startup creates **one known development credential**:
`change-me`, named `Admin Token`, with the native `admin` role and expiry one
calendar day after creation. This value is public and unsafe for normal use.
Restrict access during first setup; the operator/demo bootstrap must regenerate
it through `POST /api/v1/tokens/{id}/regenerate`, securely retain the returned
value, and set an appropriate name/expiry with `PATCH /api/v1/tokens/{id}` before
normal use. Listing/getting tokens does not return their plaintext values.

## Fresh, existing and restarted stores

GORM auto-migrates a small `bootstrap_states` table before seeding. Its
`administrative-bootstrap` row is a durable initialization guard, independent of
token existence. The guard and initial credential are committed in one
transaction; a failed initialization can safely retry. Concurrent seed calls
after migration create at most one initial credential.

On the first guarded startup of a pre-existing store, **any persisted token**
(any role, expired or otherwise), native event/audit history (including token
deletion), or other Core domain data causes adoption: only the guard is recorded;
no credentials are added, reset or removed. Absence of an active admin is never
a freshness test. Legacy tokens and duplicate/default credentials left by the
old seeding bug are preserved. Operators must explicitly inspect and regenerate
or delete unwanted legacy credentials through the native token APIs; this fix
neither silently cleans them up nor makes them safe.

Once initialized or adopted, subsequent seed calls/restarts leave credentials,
IDs, names, hashes and expirations unchanged. Regenerating, expiring or deleting
the initial credential does not restore `change-me`. Deleting **all** credentials
does not trigger automatic recovery or a backdoor on restart. Keep any required
administrative access secured before revoking the last credential.

## Revocation is not a database reset

Token DELETE/revocation preserves the initialization guard and native deletion
events. An explicit full database reset removes the guard **and** all data/history
and is a separate, destructive operator action, not credential recovery. A fully
erased store is fresh again and can receive the known development credential
once; erased history cannot be reconstructed. Do not reset a persistent demo DB
merely to restart Core, and never reset a shared database for testing.

Incoming opaque/Keycloak authentication, native token CRUD/regeneration/revocation
and optional JWT issuance are otherwise unchanged. Already issued JWTs still
expire according to issuer policy; revocation does not invalidate them offline.
See [the JWT issuer contract](TOKEN_ISSUER.md). The first-setup credential and this
guard do not constitute production credential provisioning or production readiness.
