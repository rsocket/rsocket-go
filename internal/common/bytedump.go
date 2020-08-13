package common

import (
	"fmt"
	"strconv"
	"strings"
)

var (
	_newLine            = "\n"
	_hexPadding         [16]string
	_bytePadding        [16]string
	_hexDumpRowPrefixes [4096]string
)

func init() {
	b := &strings.Builder{}
	for i := 0; i < len(_hexPadding); i++ {
		padding := len(_hexPadding) - i
		for j := 0; j < padding; j++ {
			b.WriteString("   ")
		}
		_hexPadding[i] = b.String()
		b.Reset()
	}
	for i := 0; i < len(_bytePadding); i++ {
		padding := len(_bytePadding) - i
		for j := 0; j < padding; j++ {
			b.WriteByte(' ')
		}
		_bytePadding[i] = b.String()
		b.Reset()
	}
	for i := 0; i < len(_hexDumpRowPrefixes); i++ {
		b.WriteString(_newLine)
		n := i<<4&0xFFFFFFFF | 0x100000000
		b.WriteByte('|')
		b.WriteString(leftPad(strconv.FormatInt(int64(n), 16), "0", 8))
		b.WriteByte('|')
		_hexDumpRowPrefixes[i] = b.String()
		b.Reset()
	}
}

func PrettyHexDump(b []byte) (s string, err error) {
	sb := &strings.Builder{}
	err = AppendPrettyHexDump(sb, b)
	if err != nil {
		return
	}
	s = sb.String()
	return
}

func AppendPrettyHexDump(dump *strings.Builder, b []byte) (err error) {
	if len(b) < 1 {
		return
	}
	_, err = dump.WriteString("         +-------------------------------------------------+")
	if err != nil {
		return
	}
	_, err = dump.WriteString(_newLine)
	if err != nil {
		return
	}
	_, err = dump.WriteString("         |  0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f |")
	if err != nil {
		return
	}
	_, err = dump.WriteString(_newLine)
	if err != nil {
		return
	}
	_, err = dump.WriteString("+--------+-------------------------------------------------+----------------+")
	if err != nil {
		return
	}
	length := len(b)
	startIndex := 0
	fullRows := length >> 4
	remainder := length & 0xF
	for row := 0; row < fullRows; row++ {
		rowStartIndex := row<<4 + startIndex
		err = appendHexDumpRowPrefix(dump, row, rowStartIndex)
		if err != nil {
			return
		}
		rowEndIndex := rowStartIndex + 16
		for j := rowStartIndex; j < rowEndIndex; j++ {
			_, err = fmt.Fprintf(dump, " %02x", b[j])
			if err != nil {
				return
			}
		}
		_, err = dump.WriteString(" |")
		if err != nil {
			return
		}
		for j := rowStartIndex; j < rowEndIndex; j++ {
			err = dump.WriteByte(byte2char(b[j]))
			if err != nil {
				return
			}
		}
		err = dump.WriteByte('|')
		if err != nil {
			return
		}
	}
	if remainder != 0 {
		rowStartIndex := fullRows<<4 + startIndex
		err = appendHexDumpRowPrefix(dump, fullRows, rowStartIndex)
		if err != nil {
			return
		}
		rowEndIndex := rowStartIndex + remainder
		for j := rowStartIndex; j < rowEndIndex; j++ {
			_, err = fmt.Fprintf(dump, " %02x", b[j])
			if err != nil {
				return
			}
		}
		_, err = dump.WriteString(_hexPadding[remainder])
		if err != nil {
			return
		}
		_, err = dump.WriteString(" |")
		if err != nil {
			return
		}
		for j := rowStartIndex; j < rowEndIndex; j++ {
			err = dump.WriteByte(byte2char(b[j]))
			if err != nil {
				return
			}
		}
		_, err = dump.WriteString(_bytePadding[remainder])
		if err != nil {
			return
		}
		err = dump.WriteByte('|')
		if err != nil {
			return
		}
	}
	_, err = dump.WriteString(_newLine)
	if err != nil {
		return
	}
	_, err = dump.WriteString("+--------+-------------------------------------------------+----------------+")
	return
}

func appendHexDumpRowPrefix(dump *strings.Builder, row int, rowStartIndex int) (err error) {
	if row < len(_hexDumpRowPrefixes) {
		_, err = dump.WriteString(_hexDumpRowPrefixes[row])
		return
	}
	_, err = dump.WriteString(_newLine)
	if err != nil {
		return
	}
	n := rowStartIndex&0xFFFFFFFF | 0x100000000
	_, err = dump.WriteString(strconv.FormatInt(int64(n), 16))
	if err != nil {
		return
	}
	err = dump.WriteByte('|')
	return
}

func byte2char(b byte) byte {
	if b <= 0x1f || b >= 0x7f {
		return '.'
	}
	return b
}

func leftPad(s string, padStr string, length int) string {
	padCountInt := 1 + ((length - len(padStr)) / len(padStr))
	retStr := strings.Repeat(padStr, padCountInt) + s
	return retStr[(len(retStr) - length):]
}
