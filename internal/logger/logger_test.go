package logger

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetInstance(t *testing.T) {
	firstInstance := GetInstance()
	secondInstance := GetInstance()
	require.Equal(t, firstInstance, secondInstance)
}

func TestLogLevel(t *testing.T) {
	log := GetInstance()
	// Check default log level
	require.Equal(t, LOG_INFO, log.LogLevel())

	// Set invalid log level
	err := log.SetLogLevel(100)
	require.Error(t, err)
	require.EqualError(t, err, "invalid log level")
	err = log.SetLogLevel(-1)
	require.Error(t, err)
	require.EqualError(t, err, "invalid log level")

	// Set a different and valid log level
	err = log.SetLogLevel(LOG_DEBUG)
	require.NoError(t, err)
	require.Equal(t, LOG_DEBUG, log.LogLevel())
}

func Test_addFileLinePrefix(t *testing.T) {
	msg := addFileLinePrefix("test")
	regex := regexp.MustCompile("testing.go:[0-9]+: test")
	require.Regexp(t, regex, msg)
}
