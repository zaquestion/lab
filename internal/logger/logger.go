package logger

import (
	"errors"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// Logger levels available
const (
	LOG_NONE = iota
	LOG_ERROR
	LOG_INFO
	LOG_DEBUG
)

// Logger represents lab's internal logger structure, which has a different
// log.Logger for each level, allowing different destination (file or fd) in
// different levels and also different prefixes.
type Logger struct {
	level       int
	errorLogger *log.Logger
	warnLogger  *log.Logger
	infoLogger  *log.Logger
	debugLogger *log.Logger
}

// Internal instance that is used by anyone getting it through GetInstance()
var internalLogger *Logger

// A way to avoid multiple initialization of the same logger
var once sync.Once

// GetInstance returns the default lab internal logger
func GetInstance() *Logger {
	once.Do(func() {
		internalLogger = &Logger{
			// Set INFO as default level. The user can change it
			level: LOG_INFO,
			// Setting Lmsgprefix preffix make the prefix to be printed before
			// the actual message, but after the LstdFlags (date and time)
			errorLogger: log.New(os.Stderr, "ERROR: ", log.LstdFlags|log.Lmsgprefix),
			warnLogger:  log.New(os.Stdout, "WARNING: ", log.LstdFlags|log.Lmsgprefix),
			infoLogger:  log.New(os.Stdout, "", log.LstdFlags|log.Lmsgprefix),
			debugLogger: log.New(os.Stdout, "DEBUG: ", log.LstdFlags|log.Lmsgprefix),
		}
	})
	return internalLogger
}

// SetLogLevel set the level of the internal logger.
// Allowed values are LOG_{ERROR,INFO,DEBUG,NONE}.
func (l *Logger) SetLogLevel(level int) error {
	if !(level >= LOG_NONE && level <= LOG_DEBUG) {
		return errors.New("invalid log level")
	}
	l.level = level
	return nil
}

// LogLevel return de current log level of the internal logger
func (l *Logger) LogLevel() int {
	return l.level
}

// SetStdDest sets what's the desired stdout and stderr for the internal
// log. It can be any io.Writer value.
func (l *Logger) SetStdDest(stdout io.Writer, stderr io.Writer) {
	l.errorLogger.SetOutput(stderr)
	l.warnLogger.SetOutput(stdout)
	l.infoLogger.SetOutput(stdout)
	l.debugLogger.SetOutput(stdout)
}

// printKeysAndValues prints the keys and valus, as pairs, passed to those
// functions in the way expected by go-retryablehttp LeveledLogger interface
func printKeysAndValues(l *log.Logger, keysAndValues ...interface{}) {
	for i := 0; i < len(keysAndValues)/2; i += 2 {
		l.Printf("\t%s = %s\n", keysAndValues[i], keysAndValues[i+1])
	}
}

// addFileLinePrefix prepend the file name and line number to the message being
// printed.
func addFileLinePrefix(msg string) string {
	var file string

	// Using runtime.Caller() with calldepth == 2 is enough for getting the
	// logger function callers
	_, filePath, line, ok := runtime.Caller(2)
	if ok {
		fileParts := strings.Split(filePath, "/")
		file = fileParts[len(fileParts)-1]
	} else {
		// Not sure if there's a better name or line number for an unknown caller
		file = "???"
		line = 0
	}

	prefix := []string{file, ":", strconv.Itoa(line), ":"}
	// When called from Error, Warn, Info or Debug(), the Print() used
	// doesn't know about this additional prefix we're adding, so we
	// need to add the space between it and the msg ourselves.
	if len(strings.TrimSpace(msg)) > 0 {
		prefix = append(prefix, " ")
	}

	prefixedMsg := append(prefix, msg)
	return strings.Join(prefixedMsg, "")
}

// Fatal prints the values and exit the program with os.Exit()
func (l *Logger) Fatal(values ...interface{}) {
	values = append([]interface{}{addFileLinePrefix(" ")}, values...)
	l.errorLogger.Fatal(values...)
}

// Fatalf prints formated strings and exit the program with os.Exit()
func (l *Logger) Fatalf(format string, values ...interface{}) {
	values = append([]interface{}{addFileLinePrefix("")}, values...)
	l.errorLogger.Fatalf("%s "+format, values...)
}

// Fatalln prints the values in a new line and exit the program with os.Exit()
func (l *Logger) Fatalln(values ...interface{}) {
	values = append([]interface{}{addFileLinePrefix(" ")}, values...)
	l.errorLogger.Fatalln(values...)
}

// Error prints error messages (prefixed with "ERROR:").
// These parameters match the retryablehttp.LeveledLogger, which we want to
// satisfy for silencing their debug messages being printed in the stdout.
// Error message are always printed, regardless the log level.
func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	if l.level >= LOG_ERROR {
		l.errorLogger.Print(addFileLinePrefix(msg))
		printKeysAndValues(l.errorLogger, keysAndValues...)
	}
}

// Errorf prints formated error message (prefixed with "ERROR:").
// Error message are always printed, regardless the log level.
func (l *Logger) Errorf(format string, values ...interface{}) {
	if l.level >= LOG_ERROR {
		values = append([]interface{}{addFileLinePrefix("")}, values...)
		l.errorLogger.Printf("%s "+format, values...)
	}
}

// Errorln prints error values in a new line (prefixed with "ERROR:").
// Error message are always printed, regardless the log level.
func (l *Logger) Errorln(values ...interface{}) {
	if l.level >= LOG_ERROR {
		values = append([]interface{}{addFileLinePrefix("")}, values...)
		l.errorLogger.Println(values...)
	}
}

// Warn prints warning messages (prefixed with "WARNING:").
// These parameters match the retryablehttp.LeveledLogger, which we want to
// satisfy for silencing their debug messages being printed in the stdout.
// Warning messages require at least LOG_INFO level.
func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	if l.level >= LOG_INFO {
		l.warnLogger.Print(addFileLinePrefix(msg))
		printKeysAndValues(l.warnLogger, keysAndValues...)
	}
}

// Warnf prints formated warning message (prefixed with "WARNING:").
// Warning messages require at least LOG_INFO level.
func (l *Logger) Warnf(format string, values ...interface{}) {
	if l.level >= LOG_INFO {
		values = append([]interface{}{addFileLinePrefix("")}, values...)
		l.warnLogger.Printf("%s "+format, values...)
	}
}

// Warnln prints warning values in a new line (prefixed with "WARNING:").
// Warning messages require at least LOG_INFO level.
func (l *Logger) Warnln(values ...interface{}) {
	if l.level >= LOG_INFO {
		values = append([]interface{}{addFileLinePrefix("")}, values...)
		l.warnLogger.Println(values...)
	}
}

// Info prints informational messages (prefixed with "INFO:").
// These parameters match the retryablehttp.LeveledLogger, which we want to
// satisfy for silencing their debug messages being printed in the stdout.
// Info messages require at least LOG_INFO level.
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	if l.level >= LOG_INFO {
		l.infoLogger.Print(addFileLinePrefix(msg))
		printKeysAndValues(l.infoLogger, keysAndValues...)
	}
}

// Infof prints formated informational message (prefixed with "INFO:").
// Info messages require at least LOG_INFO level.
func (l *Logger) Infof(format string, values ...interface{}) {
	if l.level >= LOG_INFO {
		values = append([]interface{}{addFileLinePrefix("")}, values...)
		l.infoLogger.Printf("%s "+format, values...)
	}
}

// Infoln prints info values in a new line (prefixed with "INFO:").
// Info messages require at least LOG_INFO level.
func (l *Logger) Infoln(values ...interface{}) {
	if l.level >= LOG_INFO {
		values = append([]interface{}{addFileLinePrefix("")}, values...)
		l.infoLogger.Println(values...)
	}
}

// Debug prints warning messages (prefixed with "DEBUG:").
// These parameters match the retryablehttp.LeveledLogger, which we want to
// satisfy for silencing thier debug messages being printed in the stdout.
// Debug messages require at least LOG_DEBUG level.
func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	if l.level >= LOG_DEBUG {
		l.debugLogger.Print(addFileLinePrefix(msg))
		printKeysAndValues(l.debugLogger, keysAndValues...)
	}
}

// Debugf prints formated debug message (prefixed with "DEBUG:").
// Debug messages require at least LOG_DEBUG level.
func (l *Logger) Debugf(format string, values ...interface{}) {
	if l.level >= LOG_DEBUG {
		values = append([]interface{}{addFileLinePrefix("")}, values...)
		l.debugLogger.Printf("%s "+format, values...)
	}
}

// Debugln prints debug values in a new line (prefixed with "DEBUG:").
// Debug messages require at least LOG_DEBUG level.
func (l *Logger) Debugln(values ...interface{}) {
	if l.level >= LOG_DEBUG {
		values = append([]interface{}{addFileLinePrefix("")}, values...)
		l.debugLogger.Println(values...)
	}
}
