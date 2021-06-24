package logger

import (
	"bytes"
	"io"
	"os"
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
	require.Equal(t, LogLevelInfo, log.LogLevel())

	// Set invalid log level
	err := log.SetLogLevel(100)
	require.Error(t, err)
	require.EqualError(t, err, "invalid log level")
	err = log.SetLogLevel(-1)
	require.Error(t, err)
	require.EqualError(t, err, "invalid log level")

	// Set a different and valid log level
	err = log.SetLogLevel(LogLevelDebug)
	require.NoError(t, err)
	require.Equal(t, LogLevelDebug, log.LogLevel())
}

func Test_addFileLinePrefix(t *testing.T) {
	msg := addFileLinePrefix("test")
	regex := regexp.MustCompile("testing.go:[0-9]+: test")
	require.Regexp(t, regex, msg)
}

func TestLogFunctions(t *testing.T) {
	type logFunc func(string, ...interface{})
	type logFuncf func(string, ...interface{})
	type logFuncln func(...interface{})

	log := GetInstance()
	log.SetLogLevel(LogLevelDebug)

	tests := []struct {
		name   string
		prefix string
		fn     logFunc
		fnf    logFuncf
		fnln   logFuncln
	}{
		{
			name:   "error",
			prefix: "ERROR:",
			fn:     log.Error,
			fnf:    log.Errorf,
			fnln:   log.Errorln,
		},
		{
			name:   "warn",
			prefix: "WARNING:",
			fn:     log.Warn,
			fnf:    log.Warnf,
			fnln:   log.Warnln,
		},
		{
			name:   "info",
			prefix: "",
			fn:     log.Info,
			fnf:    log.Infof,
			fnln:   log.Infoln,
		},
		{
			name:   "debug",
			prefix: "DEBUG:",
			fn:     log.Debug,
			fnf:    log.Debugf,
			fnln:   log.Debugln,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Redirect system stdout to our own so we can check log output
			// to stdout
			oldStdout := os.Stdout
			r, w, err := os.Pipe()
			if err != nil {
				t.Errorf("failed to redirect stdout: %s", err)
			}
			os.Stdout = w
			log.SetStdDest(w, w)
			outChan := make(chan string)

			test.fn("test")
			test.fnf("test\n")
			test.fnln("test")

			go func() {
				var buf bytes.Buffer
				io.Copy(&buf, r)
				outChan <- buf.String()
			}()

			w.Close()
			os.Stdout = oldStdout
			out := <-outChan

			regex := regexp.MustCompile(test.prefix + " logger_test.go:[0-9]+: test")
			require.Regexp(t, regex, out)
		})
	}

}
