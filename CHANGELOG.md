# Changelog

## v0.2.0 — 2026-06-05

### Added

- `serverscomx_public_ipv4` resource — allocate / read / delete / import an additional public IPv4 (alias) address on a dedicated server. Wraps `POST /hosts/dedicated_servers/{host_id}/networks/public_ipv4` and `DELETE .../networks/{id}`.
- Exposes `cidr` and `ip_address` (bare address) computed attributes — `ip_address` feeds directly into `serverscomx_ptr_record.ip` for allocate-and-reverse-DNS in one apply.
- ImportState supports composite ID `host_id:network_id`.

### Notes

- Allocation is asynchronous: the API returns `202` with a null CIDR, then assigns the address within ~1s. Create polls the network id until `status: active` (rate-limited GETs space the poll ~`request_interval` apart, max 60 attempts).
- Allocation lives on the `/networks/public_ipv4` sub-resource — the bare `/networks` collection is read-only (POST there `404`s). `OPTIONS` under-reports allowed methods on nested `/networks/...` paths (reports `OPTIONS, POST` while GET/DELETE work); behaviour was verified with live calls.
- No `PUT`/`PATCH` — `host_id`, `mask`, `distribution_method` are all `RequiresReplace`.

## v0.1.0 — 2026-05-28

### Added

- Initial provider scaffold under the `AdconnectDevOps/serverscomx` Registry namespace.
- `serverscomx_ptr_record` resource — create / read / delete / import PTR records on Servers.com dedicated servers.
- Provider config: `token` (env fallback `SERVERSCOM_TOKEN`), `endpoint`, `request_interval`.
- Rate-limited HTTP client with 1-second floor + 429 exponential backoff (max 3 retries).
- ImportState supports composite ID `host_id:ptr_id`.

### Notes

- Servers.com PTR records are immutable post-create (`OPTIONS /ptr_records/{id}` returns `allow: OPTIONS, DELETE`). Every schema attribute is `RequiresReplace`.
- Per-host PTR collection is paginated (`?per_page=100&page=N`); `Read` and `Import` paginate transparently.
- Namespace is single-token (no hyphen) so the Terraform Registry sidebar displays `serverscomx_ptr_record` identically to the TF resource type. A hyphenated namespace would render the sidebar with a hyphen while the resource type uses an underscore — a cosmetic but jarring mismatch.
