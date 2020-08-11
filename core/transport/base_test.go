package transport_test

import "github.com/pkg/errors"

var (
	fakeErr      = errors.New("fake-error")
	fakeData     = []byte("fake-data")
	fakeMetadata = []byte("fake-metadata")
)
