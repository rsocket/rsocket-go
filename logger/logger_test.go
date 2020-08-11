package logger_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/rsocket/rsocket-go/logger"
	"github.com/stretchr/testify/assert"
)

var (
	fakeFormat = "fake format: %v"
	fakeArgs   = []interface{}{"fake args"}
)

func TestSetLogger(t *testing.T) {
	logger.SetLevel(logger.LevelDebug)

	call := func() {
		logger.Debugf(fakeFormat, fakeArgs...)
		logger.Infof(fakeFormat, fakeArgs...)
		logger.Warnf(fakeFormat, fakeArgs...)
		logger.Errorf(fakeFormat, fakeArgs...)
	}

	call()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	l := NewMockLogger(ctrl)
	l.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
	l.EXPECT().Infof(gomock.Any(), gomock.Any()).Times(2)
	l.EXPECT().Warnf(gomock.Any(), gomock.Any()).Times(2)
	l.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(2)

	logger.SetLogger(l)
	assert.Equal(t, logger.LevelDebug, logger.GetLevel(), "wrong logger level")
	assert.True(t, logger.IsDebugEnabled(), "should be enabled")

	call()

	logger.SetLevel(logger.LevelInfo)
	call()

	logger.SetLevel(logger.LevelDebug)
	logger.SetLogger(nil)
	call()
}
