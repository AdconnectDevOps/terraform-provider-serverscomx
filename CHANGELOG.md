# Changelog

## Unreleased

### Added

- Initial provider scaffold.
- `serverscom_ptr_record` resource — create / read / delete / import PTR records on Servers.com dedicated servers.
- Provider config: `token` (env fallback `SERVERSCOM_TOKEN`), `endpoint`, `request_interval`.
- Rate-limited HTTP client with 1-second floor + 429 exponential backoff (max 3 retries).
- ImportState supports composite ID `host_id:ptr_id`.

### Notes

- Servers.com PTR records are immutable post-create (`OPTIONS /ptr_records/{id}` returns `allow: OPTIONS, DELETE`). Every schema attribute is `RequiresReplace`.
- Per-host PTR collection is paginated (`?per_page=100&page=N`); `Read` and `Import` paginate transparently.
