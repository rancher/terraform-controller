# Generating Dev Certs

Terraform requires signed certs to work properly, if you want to run this on your localhost you can 
use `go run .\certs\main.go` and add the `./.tls/cert.pem` to your Trusted Root Certs on your local machine following
the guides below.

## Windows
You can add the certs in the Management Console, follow [this guide](https://community.spiceworks.com/how_to/1839-installing-self-signed-ca-certificate-in-windows) 
and add the cert.pem as a Trusted Root Cert.

## MacOS
//todo

## Linux
//todo