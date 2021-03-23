terraform {
  backend "remote" {
    hostname = "localhost:8080"
    organization = "suse"

    workspaces {
      name = "bridle"
    }
  }
}