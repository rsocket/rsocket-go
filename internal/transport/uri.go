package transport

import (
	"crypto/tls"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

const (
	schemaUNIX            = "unix"
	schemaTCP             = "tcp"
	schemaWebsocket       = "ws"
	schemaWebsocketSecure = "wss"
)

// URI represents a URI of RSocket transport.
type URI url.URL

var tlsInsecure = &tls.Config{
	InsecureSkipVerify: true,
}

// MakeClientTransport creates a new client-side transport.
func (p *URI) MakeClientTransport(tc *tls.Config) (*Transport, error) {
	switch strings.ToLower(p.Scheme) {
	case schemaTCP:
		return newTCPClientTransport(schemaTCP, p.Host, tc)
	case schemaWebsocket:
		if tc == nil {
			return newWebsocketClientTransport(p.pp().String(), nil)
		}
		var clone = (url.URL)(*p)
		clone.Scheme = "wss"
		return newWebsocketClientTransport(clone.String(), tc)
	case schemaWebsocketSecure:
		if tc == nil {
			tc = tlsInsecure
		}
		return newWebsocketClientTransport(p.pp().String(), tc)
	case schemaUNIX:
		return newTCPClientTransport(schemaUNIX, p.Path, tc)
	default:
		return nil, errors.Errorf("unsupported transport url: %s", p.pp().String())
	}
}

// MakeServerTransport creates a new server-side transport.
func (p *URI) MakeServerTransport(c *tls.Config) (tp ServerTransport, err error) {
	switch strings.ToLower(p.Scheme) {
	case schemaTCP:
		tp = newTCPServerTransport(schemaTCP, p.Host, c)
	case schemaWebsocket:
		tp = newWebsocketServerTransport(p.Host, p.Path, c)
	case schemaWebsocketSecure:
		if c == nil {
			err = errors.Errorf("missing TLS Config for proto %s", schemaWebsocketSecure)
			return
		}
		tp = newWebsocketServerTransport(p.Host, p.Path, c)
	case schemaUNIX:
		tp = newTCPServerTransport(schemaUNIX, p.Path, c)
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
