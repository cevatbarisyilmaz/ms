package ms

import (
	"bytes"
)

type Mail struct {
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
