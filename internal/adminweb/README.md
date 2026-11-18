# Administrative HTTP contracts

This package prepares HTTP adapters for roadmap issues 011–015. It does not mount
routes, render pages, or enable new writes by itself. Existing JSON APIs retain
their payloads and do not require the form tokens introduced here.

An authenticated action must resolve `SessionIdentity` on every request, check the
method and same-origin policy, and then call `CSRF.ParseSessionForm`. Sessions and
form tokens authenticate either account role; administrative adapters must use
`AdminIdentity` to check the current role before reading data or issuing tickets.
Tickets prevent replay and never grant permission. Login forms
use `LoginToken` and `ParseLoginForm` with the signed preauthentication cookie.
Keep the existing login rate limit when introducing the HTML endpoint. Wrap all
administrative responses, including rejected requests, with `NoStore`.

Create a single `CSRF` instance using `NewCSRF` at server initialization. Its key
comes from crypto/rand and stays in memory. Restarting invalidates outstanding
forms; independently keyed replicas cannot share forms. The current contract is
one server instance. This is not a durable or multi-instance token service.

Session form tokens are bound to the authenticated session ID; login form tokens
are bound to a separate signed nonce cookie. Both expire within 30 minutes. A GET
does not extend the nonce cookie. Login success clears it. Tokens are reusable
until expiration and do not prevent double submission; the operation must enforce
its own invariants. Do not accept session IDs, actors or token purposes from
submitted fields.

Only the body supplies a form token. Query and X-CSRF-Token sources are rejected,
and duplicate fields are rejected. URL-encoded bodies are limited to 64 KiB in
addition to the action's MaxBytes. Multipart requires the first part to be the
text field csrf_token, at most 512 bytes. The parser limits fields and parts and
validates that token before creating a temporary file. It reads the entire
multipart envelope before returning PreparedForm. A late duplicate, malformed
suffix or size failure removes staged files. Use a positive MaxBytes per action;
the limit includes the multipart envelope, not just the uploaded file.

Only after Parse succeeds may an adapter call an administrative service. Always
defer PreparedForm.Close, including if the subsequent operation fails. Uploaded
files have server-generated staging paths; Filename is metadata for display, not
a storage destination. SingleValue preserves exact input, including whitespace;
normalization is explicit per field so passwords and external tokens are intact.
Adapters must still validate their field allowlist and operation-specific values.

Use ErrorStatus/PublicError for public output; record underlying infrastructure
errors only in the server log. SafeReturn allows known GET pages and a bounded
query allowlist. It rejects action paths, external/encoded paths, unknown query
fields and duplicates. A redirect after a successful action uses 303 and an
application-selected fallback if SafeReturn rejects a supplied destination.
Do not extend safe-return prefixes to authorize whole action namespaces.

SecureRequest preserves the existing TLS/X-Forwarded-Proto contract. The reverse
proxy must control that header; this package does not establish a new trusted
proxy policy. No session cookie or form token should be written to logs or URLs.
