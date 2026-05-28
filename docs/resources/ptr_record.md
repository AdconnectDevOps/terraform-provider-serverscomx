---
page_title: "serverscomx_ptr_record Resource"
description: |-
  Manages a PTR (reverse DNS) record on a Servers.com dedicated server.
---

# serverscomx_ptr_record

Manages a PTR (reverse DNS) record on a Servers.com dedicated server.

PTR records are created via `POST /hosts/dedicated_servers/{host_id}/ptr_records` and deleted via `DELETE /hosts/dedicated_servers/{host_id}/ptr_records/{id}`. The Servers.com API does not expose `PUT`/`PATCH` on individual PTR records — any change to a record's attributes forces a delete-and-recreate.

## Example Usage

```terraform
resource "serverscomx_ptr_record" "rev" {
  host_id = "aBcDeFgH"
  ip      = "203.0.113.10"
  domain  = "mta1.example.com"
}
```

With explicit priority and TTL:

```terraform
resource "serverscomx_ptr_record" "rev" {
  host_id  = "aBcDeFgH"
  ip       = "203.0.113.10"
  domain   = "mta1.example.com"
  priority = 0
  ttl      = 60
}
```

## Argument Reference

### Required

- `host_id` (String) — Servers.com dedicated server ID (8-character token). Look up via `GET /hosts/dedicated_servers?search_pattern=<hostname>`. Changing this forces a new record.
- `ip` (String) — Public IPv4 address to attach the PTR to. The address must already be allocated to `host_id` (alias or primary). Changing this forces a new record.
- `domain` (String) — Fully-qualified domain name returned by reverse DNS for `ip`. Changing this forces a new record.

### Optional

- `priority` (Number) — PTR priority. Defaults to `0`. Changing this forces a new record.
- `ttl` (Number) — TTL in seconds. Defaults to `60`. Changing this forces a new record.

### Read-Only

- `id` (String) — Servers.com-assigned PTR record ID.

## Import

The import ID is composite — `host_id:ptr_id` — because the Servers.com API has no global PTR namespace.

```bash
terraform import 'serverscomx_ptr_record.rev' 'aBcDeFgH:recordId123'
```

Find existing PTR IDs for a host via:

```bash
curl -s -H "Authorization: Bearer $SERVERSCOM_TOKEN" \
  'https://api.servers.com/v1/hosts/dedicated_servers/<host_id>/ptr_records?per_page=100' \
  | jq '.[] | {id, ip, domain}'
```
