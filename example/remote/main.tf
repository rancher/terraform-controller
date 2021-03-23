terraform {
  backend "remote" {
    hostname = "localhost:8080"
    organization = "suse"
    token = "fake-token"

    workspaces {
      name = "bridle"
    }
  }
}
