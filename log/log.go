// Package log connects to a local or remote syslog server with fallback to
// stderr output.
package log

import (
	"fmt"
	"log"
	"log/syslog"
	"os"
	"runtime"
	"strings"
)

// pathDeep defines the number of folders that are visible when logging a
// message with the logging location.
const pathDeep = 3

// Syslog level message, defined in RFC 5424, section 6.2.1
const (
	// LevelEmergency sets a high priority level of problem advising that system
	// is unusable.
	LevelEmergency Level = 0

	// LevelAlert sets a high priority level of problem advising to correct
	// immediately.
	LevelAlert Level = 1

	// LevelCritical sets a medium priority level of problem indicating a failure
	// in a primary system.
	LevelCritical Level = 2

	// LevelError sets a medium priority level of problem indicating a non-urgent
	// failure.
	LevelError Level = 3

	// LevelWarning sets a low priority level indicating that an error will occur
	// if action is not taken.
	LevelWarning Level = 4

	// LevelNotice sets a low priority level indicating events that are unusual,
	// but not error conditions.
	LevelNotice Level = 5

	// LevelInfo sets a very low priority level indicating normal operational
	// messages that require no action.
	LevelInfo Level = 6

	// LevelDebug sets a very low priority level indicating information useful to
	// developers for debugging the application.
	LevelDebug Level = 7
)

// Level defines the severity of an error. For example, if a custom error is
// created as bellow:
//
//    import "github.com/registrobr/gostk/log"
//
//    type ErrDatabaseFailure struct {
//    }
//
//    func (e ErrDatabaseFailure) Error() string {
//      return "database failure!"
//    }
//
//    func (e ErrDatabaseFailure) Level() log.Level {
//      return log.LevelEmergency
//    }
//
//  When used with the Logger type will be written in the syslog in the
//  corresponding log level.
type Level int

type leveler interface {
	Level() Level
}

// syslogWriter is useful to mock a low level syslog writer for unit tests.
type syslogWriter interface {
	Close() error
	Emerg(m string) (err error)
	Alert(m string) (err error)
	Crit(m string) (err error)
	Err(m string) (err error)
	Warning(m string) (err error)
	Notice(m string) (err error)
	Info(m string) (err error)
	Debug(m string) (err error)
}

var (
	remoteLogger syslogWriter
	localLogger  *log.Logger
)

func init() {
	localLogger = log.New(os.Stderr, "", log.LstdFlags)
}

// Connect initializes a connection with a remote syslog server using TCP
// (minimize package loss). All messages sent to this syslog server will be
// tagged with the name parameter.
func Connect(name, hostAndPort string) (err error) {
	remoteLogger, err = syslog.Dial("tcp", hostAndPort, syslog.LOG_INFO|syslog.LOG_LOCAL0, name)
	return
}

// ConnectLocal initializes a connection with a local syslog server.
func ConnectLocal(name string) (err error) {
	remoteLogger, err = syslog.New(syslog.LOG_INFO|syslog.LOG_LOCAL0, name)
	return
}

// Close disconnects the connection from the syslog server.
func Close() error {
	if remoteLogger == nil {
		return nil
	}

	err := remoteLogger.Close()
	if err == nil {
		remoteLogger = nil
	}
	return err
}

// Logger allows logging messages in all different level types. As it is an
// interface it can be replaced by mocks for test purposes.
type Logger interface {
	Emerg(m string)
	Emergf(m string, a ...interface{})
	Alert(m string)
	Alertf(m string, a ...interface{})
	Crit(m string)
	Critf(m string, a ...interface{})
	Error(e error)
	Errorf(m string, a ...interface{})
	Warning(m string)
	Warningf(m string, a ...interface{})
	Notice(m string)
	Noticef(m string, a ...interface{})
	Info(m string)
	Infof(m string, a ...interface{})
	Debug(m string)
	Debugf(m string, a ...interface{})
}

type logger struct {
	identifier string
}

// NewLogger returns a internal instance of the Logger type tagging an
// identifier to every message logged. This identifier is useful to group many
// messages to one related transaction id.
var NewLogger = func(id string) Logger {
	return logger{"[" + id + "] "}
}

func (l logger) Emerg(m string) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Emerg
	}
	logWithSourceInfo(f, l.identifier, m)
}

func (l logger) Emergf(m string, a ...interface{}) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Emerg
	}
	logWithSourceInfof(f, l.identifier, m, a...)
}

func (l logger) Alert(m string) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Alert
	}
	logWithSourceInfo(f, l.identifier, m)
}

func (l logger) Alertf(m string, a ...interface{}) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Alert
	}
	logWithSourceInfof(f, l.identifier, m, a...)
}

func (l logger) Crit(m string) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Crit
	}
	logWithSourceInfo(f, l.identifier, m)
}

func (l logger) Critf(m string, a ...interface{}) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Crit
	}
	logWithSourceInfof(f, l.identifier, m, a...)
}

// Error converts an Go error into an error message. The responsibility of
// knowing the file and line where the error occurred is from the Error()
// function of the specific error.
func (l logger) Error(e error) {
	if e == nil {
		return
	}

	msg := l.identifier + e.Error()
	if remoteLogger == nil {
		localLogger.Println(msg)
		return
	}

	var err error

	if levelError, ok := e.(leveler); ok {
		switch levelError.Level() {
		case LevelEmergency:
			err = remoteLogger.Emerg(msg)
		case LevelAlert:
			err = remoteLogger.Alert(msg)
		case LevelCritical:
			err = remoteLogger.Crit(msg)
		case LevelError:
			err = remoteLogger.Err(msg)
		case LevelWarning:
			err = remoteLogger.Warning(msg)
		case LevelNotice:
			err = remoteLogger.Notice(msg)
		case LevelInfo:
			err = remoteLogger.Info(msg)
		case LevelDebug:
			err = remoteLogger.Debug(msg)
		default:
			l.Warningf("Wrong error level: %d", levelError.Level())
			err = remoteLogger.Err(msg)
		}
	} else {
		err = remoteLogger.Err(msg)
	}

	if err != nil {
		localLogger.Println("Error writing to syslog. Details:", err)
		localLogger.Println(msg)
	}
}

func (l logger) Errorf(m string, a ...interface{}) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Err
	}
	logWithSourceInfof(f, l.identifier, m, a...)
}

func (l logger) Warning(m string) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Warning
	}
	logWithSourceInfo(f, l.identifier, m)
}

func (l logger) Warningf(m string, a ...interface{}) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Warning
	}
	logWithSourceInfof(f, l.identifier, m, a...)
}

func (l logger) Notice(m string) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Notice
	}
	logWithSourceInfo(f, l.identifier, m)
}

func (l logger) Noticef(m string, a ...interface{}) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Notice
	}
	logWithSourceInfof(f, l.identifier, m, a...)
}

func (l logger) Info(m string) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Info
	}
	logWithSourceInfo(f, l.identifier, m)
}

func (l logger) Infof(m string, a ...interface{}) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Info
	}
	logWithSourceInfof(f, l.identifier, m, a...)
}

func (l logger) Debug(m string) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Debug
	}
	logWithSourceInfo(f, l.identifier, m)
}

func (l logger) Debugf(m string, a ...interface{}) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Debug
	}
	logWithSourceInfof(f, l.identifier, m, a...)
}

// Emerg log an emergency message
func Emerg(m string) {
	l := NewLogger("")
	l.Emerg(m)
}

// Emergf log an emergency message with arguments
func Emergf(m string, a ...interface{}) {
	l := NewLogger("")
	l.Emergf(m, a...)
}

// Alert log an emergency message
func Alert(m string) {
	l := NewLogger("")
	l.Alert(m)
}

// Alertf log an emergency message with arguments
func Alertf(m string, a ...interface{}) {
	l := NewLogger("")
	l.Alertf(m, a...)
}

// Crit log an emergency message
func Crit(m string) {
	l := NewLogger("")
	l.Crit(m)
}

// Critf log an emergency message with arguments
func Critf(m string, a ...interface{}) {
	l := NewLogger("")
	l.Critf(m, a...)
}

// Error log an emergency message
func Error(err error) {
	l := NewLogger("")
	l.Error(err)
}

// Errorf log an emergency message with arguments
func Errorf(m string, a ...interface{}) {
	l := NewLogger("")
	l.Errorf(m, a...)
}

// Warning log an emergency message
func Warning(m string) {
	l := NewLogger("")
	l.Warning(m)
}

// Warningf log an emergency message with arguments
func Warningf(m string, a ...interface{}) {
	l := NewLogger("")
	l.Warningf(m, a...)
}

// Notice log an emergency message
func Notice(m string) {
	l := NewLogger("")
	l.Notice(m)
}

// Noticef log an emergency message with arguments
func Noticef(m string, a ...interface{}) {
	l := NewLogger("")
	l.Noticef(m, a...)
}

// Info log an emergency message
func Info(m string) {
	l := NewLogger("")
	l.Info(m)
}

// Infof log an emergency message with arguments
func Infof(m string, a ...interface{}) {
	l := NewLogger("")
	l.Infof(m, a...)
}

// Debug log an emergency message
func Debug(m string) {
	l := NewLogger("")
	l.Debug(m)
}

// Debugf log an emergency message with arguments
func Debugf(m string, a ...interface{}) {
	l := NewLogger("")
	l.Debugf(m, a...)
}

type logFunc func(string) error

func logWithSourceInfo(f logFunc, prefix, message string) {
	_, file, line, _ := runtime.Caller(2)
	file = relevantPath(file, pathDeep)
	lines := strings.Split(message, "\n")

	for _, item := range lines {
		if item == "" {
			continue
		}

		msg := fmt.Sprintf("%s%s:%d: %s", prefix, file, line, item)

		if f == nil {
			localLogger.Println(msg)

		} else if err := f(msg); err != nil {
			localLogger.Println("Error writing to syslog. Details:", err)
			localLogger.Println(msg)
		}
	}
}

func logWithSourceInfof(f logFunc, prefix, message string, a ...interface{}) {
	_, file, line, _ := runtime.Caller(2)
	file = relevantPath(file, pathDeep)
	message = fmt.Sprintf(message, a...)
	lines := strings.Split(message, "\n")

	for _, item := range lines {
		if item == "" {
			continue
		}

		msg := fmt.Sprintf("%s%s:%d: %s", prefix, file, line, item)

		if f == nil {
			localLogger.Println(msg)

		} else if err := f(msg); err != nil {
			localLogger.Println("Error writing to syslog. Details:", err)
			localLogger.Println(msg)
		}
	}
}

func relevantPath(path string, n int) string {
	tokens := strings.Split(path, "/")
	total := len(tokens)

	if n >= total {
		return path
	}

	var result string
	for i := total - n; i < total; i++ {
		result += tokens[i] + "/"
	}
	return strings.TrimSuffix(result, "/")
}
