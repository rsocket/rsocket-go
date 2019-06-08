package transport

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseUTI(t *testing.T) {
	uri, err := ParseURI("tcp://127.0.0.1:8080")
	assert.NoError(t, err, "bad URI")
	log.Println(uri)
}
