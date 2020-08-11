package socket_test

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/internal/socket"
	"github.com/rsocket/rsocket-go/logger"
)

var (
	fakeErr = errors.New("fake error")

	fakeMetadata = []byte("fake-metadata")
	fakeData     = []byte("fake-data")
	fakeMimeType = []byte("fake-mime-type")

	fakeSetup = &socket.SetupInfo{
		Version:           core.DefaultVersion,
		MetadataMimeType:  fakeMimeType,
		DataMimeType:      fakeMimeType,
		Metadata:          fakeMetadata,
		Data:              fakeData,
		KeepaliveLifetime: 90 * time.Second,
		KeepaliveInterval: 30 * time.Second,
	}
)

func InitTransport(t *testing.T) (*gomock.Controller, *MockConn, *transport.Transport) {
	ctrl := gomock.NewController(t)
	conn := NewMockConn(ctrl)
	tp := transport.NewTransport(conn)
	return ctrl, conn, tp
}

func init() {
	logger.SetLevel(logger.LevelError)
}
