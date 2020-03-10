package ms

import (
	"bytes"
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

type Service struct {
	domain          string
	dkimSignOptions *dkim.SignOptions
	nextMessageID   uint16
	nextMessageIDMu *sync.Mutex
	rand            *rand.Rand
}

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
	for _, to := range addrs {
		addr, err := resolveAddr(to.Address)
		if err != nil {
			return err
		}
		mxs, err := net.LookupMX(addr)
		if err != nil || len(mxs) == 0 {
			mxs = []*net.MX{{Host: addr}}
		}
		for _, mx := range mxs {
			return smtp.SendMail(mx.Host+":smtp", nil, from.Address, []string{to.Address}, &buffer)
		}
	}
	return nil
}

func resolveAddr(addr string) (string, error) {
	parts := strings.SplitN(addr, "@", 2)
	if len(parts) != 2 {
		return "", errors.New("invalid mail address")
	}
	return parts[1], nil
}
