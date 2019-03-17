package extension

import "fmt"

func ParseRoutingTags(bs []byte) (tags []string, err error) {
	totals := len(bs)
	cursor := 0
	for {
		if cursor >= totals {
			break
		}
		tagLen := int(bs[cursor])
		end := cursor + 1 + tagLen
		if end > totals {
			err = fmt.Errorf("bad routing tags: illegal tag len %d", tagLen)
			return
		}
		tags = append(tags, string(bs[cursor+1:end]))
		cursor += tagLen + 1
	}
	return
}
