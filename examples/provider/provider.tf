terraform {
  required_providers {
    serverscom = {
      source  = "AdconnectDevOps/serverscom-extras"
      version = "~> 0"
    }
  }
}

provider "serverscom" {
  # token is read from SERVERSCOM_TOKEN env var when omitted.
}
