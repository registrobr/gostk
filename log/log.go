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

	"github.com/registrobr/gostk/path"
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

// Dial establishes a connection to a log daemon by connecting to
// address raddr on the specified network.  Each write to the returned
// writer sends a log message with the given facility, severity and
// tag. If network is empty, Dial will connect to the local syslog server.
func Dial(network, raddr, tag string) (err error) {
	remoteLogger, err = syslog.Dial(network, raddr, syslog.LOG_INFO|syslog.LOG_LOCAL0, tag)
	return
}

// Close closes a connection to the syslog daemon.
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
	Emerg(m ...interface{})
	Emergf(m string, a ...interface{})
	Alert(m ...interface{})
	Alertf(m string, a ...interface{})
	Crit(m ...interface{})
	Critf(m string, a ...interface{})
	Error(e error)
	Errorf(m string, a ...interface{})
	Warning(m ...interface{})
	Warningf(m string, a ...interface{})
	Notice(m ...interface{})
	Noticef(m string, a ...interface{})
	Info(m ...interface{})
	Infof(m string, a ...interface{})
	Debug(m ...interface{})
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

func (l logger) Emerg(a ...interface{}) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Emerg
	}
	logWithSourceInfo(f, l.identifier, a...)
}

func (l logger) Emergf(m string, a ...interface{}) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Emerg
	}
	logWithSourceInfof(f, l.identifier, m, a...)
}

func (l logger) Alert(a ...interface{}) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Alert
	}
	logWithSourceInfo(f, l.identifier, a...)
}

func (l logger) Alertf(m string, a ...interface{}) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Alert
	}
	logWithSourceInfof(f, l.identifier, m, a...)
}

func (l logger) Crit(a ...interface{}) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Crit
	}
	logWithSourceInfo(f, l.identifier, a...)
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

func (l logger) Warning(a ...interface{}) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Warning
	}
	logWithSourceInfo(f, l.identifier, a...)
}

func (l logger) Warningf(m string, a ...interface{}) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Warning
	}
	logWithSourceInfof(f, l.identifier, m, a...)
}

func (l logger) Notice(a ...interface{}) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Notice
	}
	logWithSourceInfo(f, l.identifier, a...)
}

func (l logger) Noticef(m string, a ...interface{}) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Notice
	}
	logWithSourceInfof(f, l.identifier, m, a...)
}

func (l logger) Info(a ...interface{}) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Info
	}
	logWithSourceInfo(f, l.identifier, a...)
}

func (l logger) Infof(m string, a ...interface{}) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Info
	}
	logWithSourceInfof(f, l.identifier, m, a...)
}

func (l logger) Debug(a ...interface{}) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Debug
	}
	logWithSourceInfo(f, l.identifier, a...)
}

func (l logger) Debugf(m string, a ...interface{}) {
	var f logFunc
	if remoteLogger != nil {
		f = remoteLogger.Debug
	}
	logWithSourceInfof(f, l.identifier, m, a...)
}

// Emerg log an emergency message
func Emerg(a ...interface{}) {
	l := NewLogger("")
	l.Emerg(a...)
}

// Emergf log an emergency message with arguments
func Emergf(m string, a ...interface{}) {
	l := NewLogger("")
	l.Emergf(m, a...)
}

// Alert log an emergency message
func Alert(a ...interface{}) {
	l := NewLogger("")
	l.Alert(a...)
}

// Alertf log an emergency message with arguments
func Alertf(m string, a ...interface{}) {
	l := NewLogger("")
	l.Alertf(m, a...)
}

// Crit log an emergency message
func Crit(a ...interface{}) {
	l := NewLogger("")
	l.Crit(a...)
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
func Warning(a ...interface{}) {
	l := NewLogger("")
	l.Warning(a...)
}

// Warningf log an emergency message with arguments
func Warningf(m string, a ...interface{}) {
	l := NewLogger("")
	l.Warningf(m, a...)
}

// Notice log an emergency message
func Notice(a ...interface{}) {
	l := NewLogger("")
	l.Notice(a...)
}

// Noticef log an emergency message with arguments
func Noticef(m string, a ...interface{}) {
	l := NewLogger("")
	l.Noticef(m, a...)
}

// Info log an emergency message
func Info(a ...interface{}) {
	l := NewLogger("")
	l.Info(a...)
}

// Infof log an emergency message with arguments
func Infof(m string, a ...interface{}) {
	l := NewLogger("")
	l.Infof(m, a...)
}

// Debug log an emergency message
func Debug(a ...interface{}) {
	l := NewLogger("")
	l.Debug(a...)
}

// Debugf log an emergency message with arguments
func Debugf(m string, a ...interface{}) {
	l := NewLogger("")
	l.Debugf(m, a...)
}

type logFunc func(string) error

func logWithSourceInfo(f logFunc, prefix string, a ...interface{}) {
	// identify the caller from 3 levels above, as this function is never called
	// directly from the place that logged the message
	_, file, line, _ := runtime.Caller(3)
	file = path.RelevantPath(file, pathDeep)
	doLog(f, prefix, fmt.Sprint(a...), file, line)
}

func logWithSourceInfof(f logFunc, prefix, message string, a ...interface{}) {
	// identify the caller from 3 levels above, as this function is never called
	// directly from the place that logged the message
	_, file, line, _ := runtime.Caller(3)
	file = path.RelevantPath(file, pathDeep)
	doLog(f, prefix, fmt.Sprintf(message, a...), file, line)
}

func doLog(f logFunc, prefix, message, file string, line int) {
	// support multiline log message, breaking it in many log entries
	for _, item := range strings.Split(message, "\n") {
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
