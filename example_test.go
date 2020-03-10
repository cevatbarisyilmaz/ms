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
		dkimSelector      = "default"		 // DKIM Selector To Use
		privateKeyPemFile = "private.key"	 // Location of DKIM Private Key File
	)

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

	mailService := ms.New(domain, dkimSelector, privateKey)

	mail := &ms.Mail{
		Headers: map[string][]byte{
			"from":    []byte("\"Your Domain\" <noreply@yourdomain.com>"),
			"to":      []byte("someuser@anotherdomain.com"),
			"subject": []byte("Don't Mind Me, Just Testing Some Mail Library"),
		},
		Body: []byte("This mail confirms that you successfully setup ms!"),
	}
	err = mailService.Send(mail)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("success")
}