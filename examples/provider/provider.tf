terraform {
  required_providers {
    serverscom = {
      source  = "AdconnectDevOps/serverscomx"
      version = "~> 0"
    }
  }
}

provider "serverscomx" {
  # token is read from SERVERSCOM_TOKEN env var when omitted.
}
