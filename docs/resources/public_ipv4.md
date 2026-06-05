---
page_title: "serverscomx_public_ipv4 Resource"
description: |-
  Allocates an additional public IPv4 (alias) address on a Servers.com dedicated server.
---

# serverscomx_public_ipv4

Allocates an additional public IPv4 (alias) address on a Servers.com dedicated server.

Allocation uses `POST /hosts/dedicated_servers/{host_id}/networks/public_ipv4` and removal uses `DELETE /hosts/dedicated_servers/{host_id}/networks/{id}`. The bare `/networks` collection is read-only — allocation lives on the `/public_ipv4` sub-resource. The API has no `PUT`/`PATCH`, so any attribute change forces a delete-and-recreate.

Allocation is **asynchronous**: the API returns `202` with a null CIDR, then assigns the address within ~1s. The resource polls until the network reports `status: active` and returns the assigned CIDR.

## Example Usage

```terraform
resource "serverscomx_public_ipv4" "alias" {
  host_id = "aBcDeFgH"
}
```

Full lifecycle in one apply — allocate the alias, then set its PTR record:

```terraform
resource "serverscomx_public_ipv4" "alias" {
  host_id = "aBcDeFgH"
}

resource "serverscomx_ptr_record" "alias_rev" {
  host_id = serverscomx_public_ipv4.alias.host_id
  ip      = serverscomx_public_ipv4.alias.ip_address
  domain  = "mta1.example.com"
}
```

## Argument Reference

### Required

- `host_id` (String) — Servers.com dedicated server ID (8-character token). Look up via `GET /hosts/dedicated_servers?search_pattern=<hostname>`. Changing this forces a new allocation.

### Optional

- `mask` (Number) — Prefix length to allocate. Defaults to `32` (a single alias address). Changing this forces a new allocation.
- `distribution_method` (String) — How the address is bound: `route` (default, alias IP) or `gateway`. Changing this forces a new allocation.

### Read-Only

- `id` (String) — Servers.com-assigned network ID.
- `cidr` (String) — Allocated network in CIDR notation, e.g. `203.0.113.10/32`.
- `ip_address` (String) — Bare allocated address without the prefix, e.g. `203.0.113.10`. Convenient for feeding `serverscomx_ptr_record.ip`.

## Import

The import ID is composite — `host_id:network_id` — because networks are per-host with no global namespace.

```bash
terraform import 'serverscomx_public_ipv4.alias' 'aBcDeFgH:networkId123'
```

Find existing network IDs for a host via:

```bash
curl -s -H "Authorization: Bearer $SERVERSCOM_TOKEN" \
  'https://api.servers.com/v1/hosts/dedicated_servers/<host_id>/networks?per_page=100' \
  | jq '.[] | {id, cidr, title, status}'
```
