# ms

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/cevatbarisyilmaz/ms)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/cevatbarisyilmaz/ms?sort=semver&style=flat-square)](https://github.com/cevatbarisyilmaz/ms/releases)
[![GitHub](https://img.shields.io/github/license/cevatbarisyilmaz/ms?style=flat-square)](https://github.com/cevatbarisyilmaz/ms/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/cevatbarisyilmaz/ms?style=flat-square)](https://goreportcard.com/report/github.com/cevatbarisyilmaz/ms)

Simple mail service to send mails to remote SMTP servers.

## How to Properly Send Emails for Beginners

To be successfully delivered, an email need to pass 3 spam checks:

### Sender Policy Framework (SPF)

SPF checks if your IP address is actually authorized to send emails for your domain.

To pass SPF, you need to have a TXT DNS record for your domain.

Checkout https://dmarcian.com/spf-syntax-table/ for detailed SPF syntax.

A sample record is `v=spf1 a mx -all` which tells remote SMTP servers to accept the IP addresses have either A, AAA or MX DNS record for the domain.

### DomainKeys Identified Mail (DKIM)

DKIM is a bit trickier than SPF. It's basically a public key infrastructure for emails. For this, you'll need a public and private key.
You can generate the keys by third party tools like openssl or with crypto packages of Go.

Checkout https://knowledge.ondmarc.redsift.com/en/articles/2141527-generating-1024-bits-dkim-public-and-private-keys-using-openssl-on-a-mac
for openssl.

You can use the following code to generate keys in Go:

```go
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		log.Fatal(err)
	}
	privateKeyData := pem.EncodeToMemory(&pem.Block{
		Type:    "RSA PRIVATE KEY",
		Bytes:   x509.MarshalPKCS1PrivateKey(privateKey),
	})
	err = ioutil.WriteFile("private.key", privateKeyData, os.ModeType)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("private key is saved to private.key")
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		log.Fatal(err)
	}
	publicKeyData := pem.EncodeToMemory(&pem.Block{
		Type:    "PUBLIC KEY",
		Bytes:   publicKeyBytes,
	})
	log.Println(string(publicKeyData))
}
```

Keep private.key file to use with ms later.

Create a DNS TXT record for hostname
`<dkim-selector>._domainkey.<your-domain.com>` with the value `v=DKIM1; p=<base64-encoded-public-key>`

Replace `<dkim-selector>` with any identifier you like for the public key, replace `<your-domain.com>` with your domain,
Replace `<base64-encoded-public-key>` with base64 encoded version of your public key without the whitespace, for example hostname is `default._domainkey.example.com` and value is
`v=DKIM1; p=MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQC4O4fVFHCIcBx5TlQHypEmS1efHnhSQ3aUXOTdttHGrVQaVj8/7Mzzc1xKtjl3UO2Y6JU2h6jGBid8umbxQ14PpnfHyX4B7oWlVbm8ipUabRIr1hLRH9BGFOxHYfGjgESx0LdnyWJ6S2OnB7YMlQ/DR2TLArh8hoLCqs1YwNm3QwIDAQAB`

### Domain-based Message Authentication, Reporting & Conformance (DMARC)

DMARC is a policy based on SPF and DKIM, again works via TXT DNS records. Checkout https://en.wikipedia.org/wiki/DMARC for more info.

A basic TXT DNS record for DMARC looks like:
`hostname`: `_dmarc.<your-domain.com>`, `value`: `v=DMARC1; p=reject; rua=mailto:admin@<your-domain.com>` which tells remote SMTP servers to reject the mails that fails previous spam checks and send reports to admin@<your-domain.com>

## Sending Mail via ms

After completing the previous steps, now we're ready to use ms. A sample program looks like: (Be sure you run the
program within a host whose IP address passes the SPF check)

```go
package ms_test

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/cevatbarisyilmaz/ms"
	"io/ioutil"
	"log"
)

func Example() {
	const (
		domain            = "yourdomain.com" // Your Domain
		dkimSelector      = "default"        // DKIM Selector To Use
		privateKeyPemFile = "private.key"    // Location of DKIM Private Key File
	)

	// Get the Private Key for DKIM Signatures
	var privateKey *rsa.PrivateKey
	privateKeyData, err := ioutil.ReadFile(privateKeyPemFile)
	if err != nil {
		log.Fatal("reading private key file has failed:", err)
	}
	pemBlock, _ := pem.Decode(privateKeyData)
	if pemBlock == nil {
		log.Fatal("pem block is corrupted")
	}
	privateKey, err = x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
	if err != nil {
		log.Fatal("parsing der form of private key has failed:", err)
	}
	log.Println("private key is loaded from the pem file")

	// Create the mail service
	mailService := ms.New(domain, dkimSelector, privateKey)

	// Create a new mail
	mail := &ms.Mail{
		Headers: map[string][]byte{
			"From":    []byte("\"Your Domain\" <noreply@yourdomain.com>"),
			"To":      []byte("someuser@anotherdomain.com"),
			"Subject": []byte("Don't Mind Me, Just Testing Some Mail Library"),
		},
		Body: []byte("This mail confirms that you successfully setup ms!"),
	}

	// Send the mail via mail service
	err = mailService.Send(mail)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("success")
}
```

