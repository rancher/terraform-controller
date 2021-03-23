package main

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	"github.com/rancher/terraform-controller/pkg/file"
)

const (
	DefaultHost = "localhost"
	CertFile    = ".tls/cert.pem"
	KeyFile     = ".tls/key.pem"
)

var (
	host     = flag.String("host", DefaultHost, "Comma-separated hostnames and IPs to generate a certificate for")
	certFile = flag.String("cert-file", CertFile, "Location to put the cert file, defaults .tls/cert.pem")
	keyFile  = flag.String("key-file", KeyFile, "Location to put the cert file, defaults .tls/key.pem")
)

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	case ed25519.PrivateKey:
		return k.Public().(ed25519.PublicKey)
	default:
		return nil
	}
}

func main() {
	flag.Parse()
	log.Printf("Generating dev certs for API for host(s) %s\n", *host)

	var priv interface{}
	var err error
	priv, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate private key: %v", err)
	}

	// ECDSA, ED25519 and RSA subject keys should have the DigitalSignature
	// KeyUsage bits set in the x509.Certificate template
	keyUsage := x509.KeyUsageDigitalSignature
	// Only RSA subject keys should have the KeyEncipherment KeyUsage bits set. In
	// the context of TLS this KeyUsage is particular to RSA key exchange and
	// authentication.
	if _, isRSA := priv.(*rsa.PrivateKey); isRSA {
		keyUsage |= x509.KeyUsageKeyEncipherment
	}

	// valid from now for 1yr
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("Failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"rancher"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              keyUsage,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	hosts := strings.Split(*host, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %v", err)
	}

	if file.Exists(*certFile) {
		if err := os.Remove(*certFile); err != nil {
			log.Fatalf("failed to remove old cert %s", *certFile)
		}
	}

	certOut, err := file.Touch(*certFile)
	if err != nil {
		log.Fatalf("Error creating file %s", *certFile)
	}

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		log.Fatalf("Failed to write data to %s: %v", *certFile, err)
	}
	if err := certOut.Close(); err != nil {
		log.Fatalf("Error closing %s: %v", *certFile, err)
	}
	log.Printf("wrote %s\n", *certFile)

	if file.Exists(*keyFile) {
		if err := os.Remove(*keyFile); err != nil {
			log.Fatalf("failed to remove old cert %s", *keyFile)
		}
	}
	keyOut, err := file.Touch(*keyFile)
	if err != nil {
		log.Fatalf("Error creating file %s", *keyFile)
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		log.Fatalf("Unable to marshal private key: %v", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		log.Fatalf("Failed to write data to %s: %v", *keyFile, err)
	}
	if err := keyOut.Close(); err != nil {
		log.Fatalf("Error closing %s: %v", *keyFile, err)
	}
	log.Printf("wrote %s\n", *keyFile)
}
