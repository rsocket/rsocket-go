package extension

import (
	"errors"
)

const (
	_notWellKnown = "unknown"
	_simpleAuth   = "simple"
	_bearerAuth   = "bearer"
)

const (
	_authenticationSimple wellKnownAuthenticationType = 0x00
	_authenticationBearer wellKnownAuthenticationType = 0x01
)

var errInvalidAuthBytes = errors.New("invalid authentication bytes")

type wellKnownAuthenticationType uint8

func (w wellKnownAuthenticationType) String() string {
	switch w {
	case _authenticationSimple:
		return _simpleAuth
	case _authenticationBearer:
		return _bearerAuth
	default:
		return _notWellKnown
	}
}

// Authentication is a necessary component to any real world application.
// This extension specification provides a standardized mechanism for including both the type of credentials and the credentials in metadata payloads.
// https://github.com/rsocket/rsocket/blob/master/Extensions/Security/WellKnownAuthTypes.md
type Authentication struct {
	typ     string
	payload []byte
}

// Type returns type of Authentication as string.
func (a Authentication) Type() string {
	return a.typ
}

// Payload returns payload in Authentication.
func (a Authentication) Payload() []byte {
	return a.payload
}

// IsWellKnown returns true if Authentication Type is Well-Known.
func (a Authentication) IsWellKnown() (ok bool) {
	_, ok = parseWellKnownAuthenticateType(a.typ)
	return
}

// NewAuthentication creates a new Authentication
func NewAuthentication(authType string, payload []byte) Authentication {
	if len(authType) > 0x7F {
		panic("illegal authType length: exceed 127 bytes")
	}
	return Authentication{
		typ:     authType,
		payload: payload,
	}
}

// Bytes encodes current Authentication to byte slice.
func (a Authentication) Bytes() (raw []byte) {
	if w, ok := parseWellKnownAuthenticateType(a.typ); ok {
		raw = append(raw, uint8(w)|0x80)
	} else {
		raw = append(raw, byte(len(a.typ)))
		raw = append(raw, a.typ...)
	}
	raw = append(raw, a.payload...)
	return
}

// ParseAuthentication parse Authentication from raw bytes.
func ParseAuthentication(raw []byte) (auth Authentication, err error) {
	totals := len(raw)
	if totals < 1 {
		err = errInvalidAuthBytes
		return
	}
	first := raw[0]
	n := 0x7F & first
	if first&0x80 != 0 {
		auth.typ = wellKnownAuthenticationType(n).String()
		auth.payload = raw[1:]
		return
	}
	if totals < int(n+1) {
		err = errInvalidAuthBytes
		return
	}
	auth.typ = string(raw[1 : 1+n])
	auth.payload = raw[n+1:]
	return
}

func parseWellKnownAuthenticateType(typ string) (au wellKnownAuthenticationType, ok bool) {
	switch typ {
	case _simpleAuth:
		ok = true
		au = _authenticationSimple
	case _bearerAuth:
		ok = true
		au = _authenticationBearer
	}
	return
}
