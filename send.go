// Package ms provides a simple email service to send emails to remote SMTP servers
package ms

import (
	"bytes"
	"context"
	"crypto"
	"errors"
	"github.com/emersion/go-msgauth/dkim"
	"github.com/emersion/go-smtp"
	"math/rand"
	"net"
	"net/mail"
	"strconv"
	"strings"
	"sync"
	"time"
)

const mxLookUpTimeout = time.Second * 8

var smtpPorts = []string{":587", ":25"}

type Service struct {
	domain          string
	dkimSignOptions *dkim.SignOptions
	nextMessageID   uint16
	nextMessageIDMu *sync.Mutex
	rand            *rand.Rand
}

// New returns a new Service to send emails via
// domain is the associated domain name with the host
// dkimSelector is the DKIM selector to use with DKIM signature
// dkimSigner is the private key belongs to the domain and DKIM selector tuple
// check out README if you are not sure what DKIM is
func New(domain string, dkimSelector string, dkimSigner crypto.Signer) *Service {
	serviceRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &Service{
		domain: domain,
		dkimSignOptions: &dkim.SignOptions{
			Domain:   domain,
			Selector: dkimSelector,
			Signer:   dkimSigner,
			Hash:     crypto.SHA256,
		},
		nextMessageID:   uint16(serviceRand.Intn(16)) + 1,
		nextMessageIDMu: &sync.Mutex{},
		rand:            serviceRand,
	}
}

// Send sends the mail to a remote SMTP server
func (s *Service) Send(m *Mail) error {
	s.nextMessageIDMu.Lock()
	msgID := s.nextMessageID
	s.nextMessageID += uint16(s.rand.Intn(16))
	s.nextMessageIDMu.Unlock()
	m.Headers["message-id"] = []byte("<" + strconv.Itoa(int(time.Now().Unix())) + "." + strconv.Itoa(rand.Int()) + "." + strconv.Itoa(int(msgID)) + "@movieofthenight.com>")
	from, err := mail.ParseAddress(string(m.Headers["from"]))
	if err != nil {
		return err
	}
	signer, err := dkim.NewSigner(s.dkimSignOptions)
	if err != nil {
		return err
	}
	rawMail := m.encode()
	_, err = signer.Write(rawMail)
	if err != nil {
		return err
	}
	err = signer.Close()
	if err != nil {
		return err
	}
	var buffer bytes.Buffer
	buffer.WriteString(signer.Signature())
	buffer.Write(rawMail)
	addrs, err := mail.ParseAddressList(string(m.Headers["to"]))
	if err != nil {
		return err
	}
	var firstError error
	for _, to := range addrs {
		addr, err := resolveAddr(to.Address)
		if err != nil {
			return err
		}
		ctx, _ := context.WithTimeout(context.Background(), mxLookUpTimeout)
		mxs, err := net.DefaultResolver.LookupMX(ctx, addr)
		if err != nil || len(mxs) == 0 {
			mxs = []*net.MX{{Host: addr}}
		}
		for _, mx := range mxs {
			for _, port := range smtpPorts {
				err = smtp.SendMail(mx.Host+port, nil, from.Address, []string{to.Address}, &buffer)
				if err == nil {
					return nil
				}
				if firstError == nil {
					firstError = err
				}
			}
		}
	}
	return firstError
}

func resolveAddr(addr string) (string, error) {
	parts := strings.SplitN(addr, "@", 2)
	if len(parts) != 2 {
		return "", errors.New("invalid mail address")
	}
	return parts[1], nil
}
