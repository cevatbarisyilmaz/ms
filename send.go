// Package ms provides a simple email service to send emails to remote SMTP servers
package ms

import (
	"bytes"
	"context"
	"crypto"
	"github.com/cevatbarisyilmaz/ms/smtp"
	"github.com/emersion/go-msgauth/dkim"
	"github.com/pkg/errors"
	"math/rand"
	"net"
	"net/mail"
	"strconv"
	"strings"
	"sync"
	"time"
)

const timeout = time.Second * 8

// Service is used to send mails
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
	m.Headers["Message-ID"] = []byte("<" + strconv.Itoa(int(time.Now().Unix())) + "." + strconv.Itoa(rand.Int()) + "." + strconv.Itoa(int(msgID)) + "@movieofthenight.com>")
	from, err := mail.ParseAddress(string(m.Headers["From"]))
	if err != nil {
		return errors.Wrap(err, "parsing form header failed")
	}
	var to []string
	addrs, err := mail.ParseAddressList(string(m.Headers["To"]))
	if err == nil {
		for _, addr := range addrs {
			to = append(to, addr.Address)
		}
	}
	addrs, err = mail.ParseAddressList(string(m.Headers["Cc"]))
	if err == nil {
		for _, addr := range addrs {
			to = append(to, addr.Address)
		}
	}
	addrs, err = mail.ParseAddressList(string(m.Headers["Bcc"]))
	if err == nil {
		for _, addr := range addrs {
			to = append(to, addr.Address)
		}
	}
	delete(m.Headers, "Bcc")
	if len(to) == 0 {
		return errors.New("either To, Cc, or Bcc must be supplied")
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
	var firstError error
	for _, receipent := range to {
		addr, err := resolveAddr(receipent)
		if err != nil {
			return err
		}
		ctx, _ := context.WithTimeout(context.Background(), timeout)
		mxs, err := net.DefaultResolver.LookupMX(ctx, addr)
		if err != nil || len(mxs) == 0 {
			mxs = []*net.MX{{Host: addr}}
		}
		for _, mx := range mxs {
			err = smtp.SendMail(mx.Host+":smtp", nil, from.Address, []string{receipent}, &buffer, s.domain)
			if err == nil {
				return nil
			}
			if firstError == nil {
				firstError = err
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
