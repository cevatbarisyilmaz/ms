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
