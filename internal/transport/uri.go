package transport

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

const (
	schemaUNIX      = "unix"
	schemaTCP       = "tcp"
	schemaWebsocket = "ws"
)

// URI represents a URI of RSocket transport.
type URI url.URL

// MakeClientTransport creates a new client-side transport.
func (p *URI) MakeClientTransport() (*Transport, error) {
	switch strings.ToLower(p.Scheme) {
	case schemaTCP:
		return newTCPClientTransport(schemaTCP, p.Host)
	case schemaWebsocket:
		return newWebsocketClientTransport(p.pp().String())
	case schemaUNIX:
		return newTCPClientTransport(schemaUNIX, p.Path)
	default:
		return nil, errors.Errorf("unsupported transport url: %s", p.pp().String())
	}
}

// MakeServerTransport creates a new server-side transport.
func (p *URI) MakeServerTransport() (tp ServerTransport, err error) {
	switch strings.ToLower(p.Scheme) {
	case schemaTCP:
		tp = newTCPServerTransport(schemaTCP, p.Host)
	case schemaWebsocket:
		tp = newWebsocketServerTransport(p.Host, p.Path)
	case schemaUNIX:
		tp = newTCPServerTransport(schemaUNIX, p.Path)
	default:
		err = errors.Errorf("unsupported transport url: %s", p.pp().String())
	}
	return
}

func (p *URI) String() string {
	return p.pp().String()
}

func (p *URI) pp() *url.URL {
	return (*url.URL)(p)
}

// ParseURI parse URI string and returns a URI.
func ParseURI(rawurl string) (*URI, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, errors.Wrapf(err, "parse url failed: %s", rawurl)
	}
	return (*URI)(u), nil
}
