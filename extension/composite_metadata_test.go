package extension

import (
	"fmt"
	"github.com/rsocket/rsocket-go/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDecodeCompositeMetadata(t *testing.T) {
	const mod = 3
	bf := common.BorrowByteBuffer()
	defer common.ReturnByteBuffer(bf)
	for i := 0; i < 10; i++ {
		var cm CompositeMetadata
		if i%mod == 0 {
			cm = NewCompositeMetadata("application/notWell", []byte(fmt.Sprintf("notWell_%d", i)))
		} else {
			cm = NewCompositeMetadata("text/plain", []byte(fmt.Sprintf("text_%d", i)))
		}
		bs, err := cm.encode()
		if err != nil {
			assert.Error(t, err, "encode composite metadata failed")
		}
		_, _ = bf.Write(bs)
	}
	bs := bf.Bytes()
	cms, err := DecodeCompositeMetadata(bs)
	if err != nil {
		assert.Error(t, err, "decode composite metadata failed")
	}

	for k, v := range cms {
		if k%mod == 0 {
			assert.Equal(t, "application/notWell", v.MIME(), "bad MIME")
			assert.Equal(t, fmt.Sprintf("notWell_%d", k), string(v.Payload()), "bad payload")
		} else {
			assert.Equal(t, "text/plain", v.MIME(), "bad MIME")
			assert.Equal(t, fmt.Sprintf("text_%d", k), string(v.Payload()), "bad payload")
		}
	}

}
