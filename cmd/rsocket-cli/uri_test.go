package main

import (
	"log"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUTI(t *testing.T) {
	uri, err := ParseURI("tcp://127.0.0.1:8080")
	assert.NoError(t, err, "bad URI")
	log.Println(uri)
}

func TestName(t *testing.T) {
	//u, err := url.Parse("unix:///tmp/rsocket.sock")
	u, err := url.Parse("tcp://127.0.0.1:8080")
	require.NoError(t, err, "bad parse")
	log.Println("schema:", u.Scheme)
	log.Println("host:", u.Host)
	log.Println("path:", u.Path)
}
