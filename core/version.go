package core

import (
	"encoding/binary"
	"io"
	"strconv"
	"strings"
)

// DefaultVersion is default protocol version.
var DefaultVersion Version = [2]uint16{1, 0}

// Version define the version of protocol.
// It includes major and minor version.
type Version [2]uint16

// Bytes returns raw bytes of current version.
func (p Version) Bytes() []byte {
	bs := make([]byte, 4)
	binary.BigEndian.PutUint16(bs, p[0])
	binary.BigEndian.PutUint16(bs[2:], p[1])
	return bs
}

// Major returns major version.
func (p Version) Major() uint16 {
	return p[0]
}

// Minor returns minor version.
func (p Version) Minor() uint16 {
	return p[1]
}

// Equals returns true if versions are same.
func (p Version) Equals(version Version) bool {
	return p.Major() == version.Major() && p.Minor() == version.Minor()
}

// GreaterThan returns true if current version is greater than target.
func (p Version) GreaterThan(version Version) bool {
	if p[0] == version[0] {
		return p[1] > version[1]
	}
	return p[0] > version[0]
}

// LessThan returns true if current version is less than target.
func (p Version) LessThan(version Version) bool {
	if p[0] == version[0] {
		return p[1] < version[1]
	}
	return p[0] < version[0]
}

// WriteTo write raw version bytes to a writer.
func (p Version) WriteTo(w io.Writer) (n int64, err error) {
	err = binary.Write(w, binary.BigEndian, p[0])
	if err != nil {
		return
	}
	err = binary.Write(w, binary.BigEndian, p[1])
	if err != nil {
		return
	}
	n = 4
	return
}

func (p Version) String() string {
	b := strings.Builder{}
	b.WriteString(strconv.Itoa(int(p[0])))
	b.WriteByte('.')
	b.WriteString(strconv.Itoa(int(p[1])))
	return b.String()
}

// NewVersion creates a new Version from major and minor.
func NewVersion(major, minor uint16) Version {
	return Version{
		major, minor,
	}
}
