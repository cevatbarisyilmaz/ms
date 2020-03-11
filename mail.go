package ms

import (
	"bytes"
)

// Mail holds mail headers and the body to send
type Mail struct {
	// From, To, Cc and Bcc headers should start with a capital letter followed by lower cases
	// Such as "From", "To" etc.
	Headers map[string][]byte
	Body    []byte
}

func (m *Mail) encode() []byte {
	var buffer bytes.Buffer
	for key, value := range m.Headers {
		buffer.WriteString(key)
		buffer.WriteString(": ")
		buffer.Write(value)
		buffer.WriteString("\r\n")
	}
	buffer.WriteString("\r\n")
	buffer.Write(m.Body)
	buffer.WriteString("\r\n")
	return buffer.Bytes()
}
