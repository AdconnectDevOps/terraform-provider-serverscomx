# Allocate a single alias IPv4 (/32) on a dedicated server.
resource "serverscomx_public_ipv4" "alias" {
  host_id = "aBcDeFgH" # GET /hosts/dedicated_servers?search_pattern=<hostname>
}

# Full lifecycle in one apply: allocate the alias, then set its PTR record.
resource "serverscomx_ptr_record" "alias_rev" {
  host_id = serverscomx_public_ipv4.alias.host_id
  ip      = serverscomx_public_ipv4.alias.ip_address
  domain  = "mta1.example.com"
}

output "allocated_cidr" {
  value = serverscomx_public_ipv4.alias.cidr # e.g. "203.0.113.10/32"
}
