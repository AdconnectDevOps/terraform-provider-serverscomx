---
page_title: "Servers.com Extras Provider"
description: |-
  Terraform provider for Servers.com Public API endpoints missing from the official serverscom/serverscom provider.
---

# Servers.com Extras Provider

Gap-fill Terraform provider for Servers.com Public API endpoints not covered by the official [`serverscom/serverscom`](https://registry.terraform.io/providers/serverscom/serverscom/latest) provider.

The official provider, at v0.2.2, manages only `cloud_computing_instance`, `dedicated_server`, `l2_segment`, `sbm_server`, `ssh_key`, and `subnetwork`. This provider adds resources for the endpoints the Servers.com Public API exposes but the official provider does not wrap.

## Resources

| Resource | Endpoint |
|---|---|
| `serverscom_ptr_record` | `/hosts/dedicated_servers/{host_id}/ptr_records` |

## Authentication

API tokens are issued in the [Servers.com Customer Portal](https://portal.servers.com/#/profile/api-tokens).

## Using with the official provider

Both providers register `serverscom` as the resource-type prefix. To use them in the same Terraform root, alias one via the local name in `required_providers`:

```terraform
terraform {
  required_providers {
    serverscom = {
      source  = "serverscom/serverscom"
      version = "~> 0.2"
    }
    serverscom_extras = {
      source  = "AdconnectDevOps/serverscom-extras"
      version = "~> 0"
    }
  }
}
```

When using only this provider, the local name can stay `serverscom` and the resource name is `serverscom_ptr_record` (no `_extras` infix).

## Example Usage

```terraform
provider "serverscom" {
  # token is required. Falls back to SERVERSCOM_TOKEN env var when omitted.
}

resource "serverscom_ptr_record" "rev" {
  host_id = "aBcDeFgH"
  ip      = "203.0.113.10"
  domain  = "mta1.example.com"
}
```

## Schema

### Optional

- `token` (String, Sensitive) — Servers.com API token. Falls back to the `SERVERSCOM_TOKEN` environment variable when omitted.
- `endpoint` (String) — API base URL. Defaults to `https://api.servers.com/v1`.
- `request_interval` (Number) — Minimum seconds between API requests. Defaults to `1`.
