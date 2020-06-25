package main

import (
	"crypto/tls"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/core/transport"
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

// IsWebsocket returns true if current uri is websocket.
func (p *URI) IsWebsocket() bool {
	switch strings.ToLower(p.Scheme) {
	case schemaWebsocket, schemaWebsocketSecure:
		return true
	default:
		return false
	}
}

// MakeClientTransport creates a new client-side transport.
func (p *URI) MakeClientTransport(tc *tls.Config, headers map[string][]string) (*transport.Transport, error) {
	switch strings.ToLower(p.Scheme) {
	case schemaTCP:
		return transport.NewTcpClientTransport(schemaTCP, p.Host, tc)
	case schemaWebsocket:
		if tc == nil {
			return transport.NewWebsocketClientTransport(p.pp().String(), nil, headers)
		}
		var clone = (url.URL)(*p)
		clone.Scheme = "wss"
		return transport.NewWebsocketClientTransport(clone.String(), tc, headers)
	case schemaWebsocketSecure:
		if tc == nil {
			tc = tlsInsecure
		}
		return transport.NewWebsocketClientTransport(p.pp().String(), tc, headers)
	case schemaUNIX:
		return transport.NewTcpClientTransport(schemaUNIX, p.Path, tc)
	default:
		return nil, errors.Errorf("unsupported transport url: %s", p.pp().String())
	}
}

// MakeServerTransport creates a new server-side transport.
func (p *URI) MakeServerTransport(c *tls.Config) (tp transport.ServerTransport, err error) {
	switch strings.ToLower(p.Scheme) {
	case schemaTCP:
		tp = transport.NewTcpServerTransport(schemaTCP, p.Host, c)
	case schemaWebsocket:
		tp = transport.NewWebsocketServerTransport(p.Host, p.Path, c)
	case schemaWebsocketSecure:
		if c == nil {
			err = errors.Errorf("missing TLS Config for proto %s", schemaWebsocketSecure)
			return
		}
		tp = transport.NewWebsocketServerTransport(p.Host, p.Path, c)
	case schemaUNIX:
		tp = transport.NewTcpServerTransport(schemaUNIX, p.Path, c)
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
func ParseURI(rawUrl string) (*URI, error) {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return nil, errors.Wrapf(err, "parse url failed: %s", rawUrl)
	}
	return (*URI)(u), nil
}
