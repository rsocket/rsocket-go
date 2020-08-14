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

func PrettyHexDump(b []byte) string {
	sb := &strings.Builder{}
	AppendPrettyHexDump(sb, b)
	return sb.String()
}

func AppendPrettyHexDump(dump *strings.Builder, b []byte) {
	if len(b) < 1 {
		return
	}
	dump.WriteString("         +-------------------------------------------------+")
	dump.WriteString(_newLine)
	dump.WriteString("         |  0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f |")
	dump.WriteString(_newLine)
	dump.WriteString("+--------+-------------------------------------------------+----------------+")
	length := len(b)
	startIndex := 0
	fullRows := length >> 4
	remainder := length & 0xF
	for row := 0; row < fullRows; row++ {
		rowStartIndex := row<<4 + startIndex
		appendHexDumpRowPrefix(dump, row, rowStartIndex)
		rowEndIndex := rowStartIndex + 16
		for j := rowStartIndex; j < rowEndIndex; j++ {
			_, _ = fmt.Fprintf(dump, " %02x", b[j])
		}
		dump.WriteString(" |")
		for j := rowStartIndex; j < rowEndIndex; j++ {
			dump.WriteByte(byte2char(b[j]))
		}
		dump.WriteByte('|')
	}
	if remainder != 0 {
		rowStartIndex := fullRows<<4 + startIndex
		appendHexDumpRowPrefix(dump, fullRows, rowStartIndex)
		rowEndIndex := rowStartIndex + remainder
		for j := rowStartIndex; j < rowEndIndex; j++ {
			_, _ = fmt.Fprintf(dump, " %02x", b[j])
		}
		dump.WriteString(_hexPadding[remainder])
		dump.WriteString(" |")
		for j := rowStartIndex; j < rowEndIndex; j++ {
			dump.WriteByte(byte2char(b[j]))
		}
		dump.WriteString(_bytePadding[remainder])
		dump.WriteByte('|')
	}
	dump.WriteString(_newLine)
	dump.WriteString("+--------+-------------------------------------------------+----------------+")
}

func appendHexDumpRowPrefix(dump *strings.Builder, row int, rowStartIndex int) {
	if row < len(_hexDumpRowPrefixes) {
		dump.WriteString(_hexDumpRowPrefixes[row])
		return
	}
	dump.WriteString(_newLine)
	n := rowStartIndex&0xFFFFFFFF | 0x100000000
	dump.WriteString(strconv.FormatInt(int64(n), 16))
	dump.WriteByte('|')
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
