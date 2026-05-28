resource "serverscomx_ptr_record" "rev" {
  host_id = "aBcDeFgH" # GET /hosts/dedicated_servers?search_pattern=<hostname>
  ip      = "203.0.113.10"
  domain  = "mta1.example.com"
}
