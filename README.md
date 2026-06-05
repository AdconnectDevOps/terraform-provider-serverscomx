# terraform-provider-serverscomx

Gap-fill Terraform provider for Servers.com Public API endpoints missing from the official [`serverscom/serverscom`](https://registry.terraform.io/providers/serverscom/serverscom/latest) provider.

Published to the Terraform Registry as **[`AdconnectDevOps/serverscomx`](https://registry.terraform.io/providers/AdconnectDevOps/serverscomx/latest)**.

## Why this exists

The official provider (`serverscom/serverscom`, v0.2.2 at the time of writing) wraps only six resources: `cloud_computing_instance`, `dedicated_server`, `l2_segment`, `sbm_server`, `ssh_key`, `subnetwork`. The Servers.com Public API exposes many more endpoints — this provider adds the ones we use internally.

## Available resources

| Resource | Purpose |
|---|---|
| `serverscomx_ptr_record` | Reverse DNS (PTR) record on a dedicated server. Wraps `POST/DELETE /hosts/dedicated_servers/{host_id}/ptr_records[/{id}]`. |
| `serverscomx_public_ipv4` | Additional public IPv4 (alias) address on a dedicated server. Wraps `POST /hosts/dedicated_servers/{host_id}/networks/public_ipv4` + `DELETE .../networks/{id}`. |

## Using both providers in one root

The official provider and this one both use the `serverscom` resource-type prefix. To use them in the same Terraform root, alias one of them via the local name in `required_providers`:

```hcl
terraform {
  required_providers {
    serverscom = {
      source  = "serverscom/serverscom"
      version = "~> 0.2"
    }
    serverscom_extras = {
      source  = "AdconnectDevOps/serverscomx"
      version = "~> 0"
    }
  }
}

resource "serverscom_dedicated_server" "node" {       # official
  # ...
}

resource "serverscom_extras_ptr_record" "rev" {       # this provider
  # ...
}
```

When using only this provider, the resource name stays `serverscomx_ptr_record` (no `_extras` prefix).

## Configuration

```hcl
provider "serverscomx" {
  # token is required. Falls back to SERVERSCOM_TOKEN env var when omitted.
  # token = "your-api-token"

  # endpoint = "https://api.servers.com/v1"  # default
  # request_interval = 1                     # min seconds between requests
}
```

API tokens are issued in the [Servers.com Customer Portal](https://portal.servers.com/#/profile/api-tokens).

## Example

```hcl
resource "serverscomx_ptr_record" "mta1_example" {
  host_id = "aBcDeFgH" # GET /hosts/dedicated_servers?search_pattern=<host>
  ip      = "203.0.113.10"
  domain  = "mta1.example.com"
  # priority = 0      # optional, defaults to 0
  # ttl      = 60     # optional, defaults to 60
}
```

## Importing existing records

The import ID is composite — `host_id:ptr_id` — because the API has no global PTR namespace.

```bash
terraform import 'serverscomx_ptr_record.rev' 'aBcDeFgH:recordId123'
```

## Development

```bash
make build      # local binary
make install    # build + drop into ~/.terraform.d/plugins/
make test
make fmt vet
```

Release flow: tag `vX.Y.Z` → GitHub Actions runs goreleaser → Terraform Registry picks up the new release within ~10-15 min.
