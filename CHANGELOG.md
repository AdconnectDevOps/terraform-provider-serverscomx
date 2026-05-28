# Changelog

## v0.1.0 — 2026-05-28

### Added

- Initial provider scaffold under the `AdconnectDevOps/serverscomx` Registry namespace (renamed from the abandoned `AdconnectDevOps/serverscom-extras` namespace — that version line is orphaned on the Registry; do not use).
- `serverscomx_ptr_record` resource — create / read / delete / import PTR records on Servers.com dedicated servers.
- Provider config: `token` (env fallback `SERVERSCOM_TOKEN`), `endpoint`, `request_interval`.
- Rate-limited HTTP client with 1-second floor + 429 exponential backoff (max 3 retries).
- ImportState supports composite ID `host_id:ptr_id`.

### Notes

- Servers.com PTR records are immutable post-create (`OPTIONS /ptr_records/{id}` returns `allow: OPTIONS, DELETE`). Every schema attribute is `RequiresReplace`.
- Per-host PTR collection is paginated (`?per_page=100&page=N`); `Read` and `Import` paginate transparently.
- Rename rationale: Registry sidebar derives the displayed resource name from the namespace name. A namespace with a hyphen (`serverscom-extras`) produced `serverscom-extras_ptr_record` in the sidebar while the actual TF resource type was `serverscom_ptr_record` — a cosmetic but jarring mismatch. The `serverscomx` namespace (no hyphen) keeps sidebar and code identical: `serverscomx_ptr_record` everywhere.
