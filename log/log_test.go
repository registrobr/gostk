package log

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"reflect"
	"strings"
	"testing"
)

func TestDial(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	go func(l net.Listener) {
		if _, err := l.Accept(); err != nil {
			return
		}
	}(l)

	scenarios := []struct {
		description   string
		network       string
		raddr         string
		tag           string
		expectedError error
	}{
		{
			description: "it should connect to the remote syslog server correctly",
			network:     "tcp",
			raddr:       l.Addr().String(),
			tag:         "test",
		},
		{
			description:   "it should detect an error connecting to the remote syslog server",
			network:       "tcp",
			raddr:         "localhost:0",
			tag:           "test",
			expectedError: fmt.Errorf("dial tcp 127.0.0.1:0: getsockopt: connection refused"),
		},
	}

	for i, scenario := range scenarios {
		err := Dial(scenario.network, scenario.raddr, scenario.tag)

		if ((err == nil || scenario.expectedError == nil) && err != scenario.expectedError) ||
			(err != nil && scenario.expectedError != nil && err.Error() != scenario.expectedError.Error()) {
			t.Errorf("scenario %d, “%s”: mismatch errors. Expecting: “%v”; found “%v”",
				i, scenario.description, scenario.expectedError, err)
		}
	}
}

func TestClose(t *testing.T) {
	scenarios := []struct {
		description   string
		remoteLogger  syslogWriter
		expectedError error
	}{
		{
			description: "it should ignore if there's no connection to syslog",
		},
		{
			description: "it should close the connection of the syslog correctly",
			remoteLogger: mockSyslogWriter{
				mockClose: func() error {
					return nil
				},
			},
		},
		{
			description: "it should detect an error while closing the connection of the syslog",
			remoteLogger: mockSyslogWriter{
				mockClose: func() error {
					return fmt.Errorf("generic error")
				},
			},
			expectedError: fmt.Errorf("generic error"),
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	for i, scenario := range scenarios {
		remoteLogger = scenario.remoteLogger

		if err := Close(); !reflect.DeepEqual(err, scenario.expectedError) {
			t.Errorf("scenario %d, “%s”: mismatch errors. Expecting: “%v”; found “%v”",
				i, scenario.description, scenario.expectedError, err)
		}
	}
}

func TestLogger_Emerg(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		remoteLogger        syslogWriter
		identifier          string
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message 1\n\nthis is the message 2",
			remoteLogger: mockSyslogWriter{
				mockEmerg: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedIdentifier := "[test] "
						if !strings.HasPrefix(msg, expectedIdentifier) {
							t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
						}

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
			identifier: "test",
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockEmerg: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			identifier:          "test",
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message 1\n\nthis is the message 2",
			identifier:          "test",
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		logger := NewLogger(scenario.identifier)
		logger.Emerg(scenario.msg)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestLogger_Emergf(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		arguments           []interface{}
		remoteLogger        syslogWriter
		identifier          string
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message %d\n\nthis is the message %d",
			arguments:   []interface{}{1, 2},
			remoteLogger: mockSyslogWriter{
				mockEmerg: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedIdentifier := "[test] "
						if !strings.HasPrefix(msg, expectedIdentifier) {
							t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
						}

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
			identifier: "test",
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockEmerg: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			identifier:          "test",
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message %d\n\nthis is the message %d",
			arguments:           []interface{}{1, 2},
			identifier:          "test",
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		logger := NewLogger(scenario.identifier)
		logger.Emergf(scenario.msg, scenario.arguments...)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestLogger_Alert(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		remoteLogger        syslogWriter
		identifier          string
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message 1\n\nthis is the message 2",
			remoteLogger: mockSyslogWriter{
				mockAlert: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedIdentifier := "[test] "
						if !strings.HasPrefix(msg, expectedIdentifier) {
							t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
						}

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
			identifier: "test",
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockAlert: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			identifier:          "test",
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message 1\n\nthis is the message 2",
			identifier:          "test",
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		logger := NewLogger(scenario.identifier)
		logger.Alert(scenario.msg)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestLogger_Alertf(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		arguments           []interface{}
		remoteLogger        syslogWriter
		identifier          string
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message %d\n\nthis is the message %d",
			arguments:   []interface{}{1, 2},
			remoteLogger: mockSyslogWriter{
				mockAlert: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedIdentifier := "[test] "
						if !strings.HasPrefix(msg, expectedIdentifier) {
							t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
						}

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
			identifier: "test",
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockAlert: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			identifier:          "test",
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message %d\n\nthis is the message %d",
			arguments:           []interface{}{1, 2},
			identifier:          "test",
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		logger := NewLogger(scenario.identifier)
		logger.Alertf(scenario.msg, scenario.arguments...)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestLogger_Crit(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		remoteLogger        syslogWriter
		identifier          string
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message 1\n\nthis is the message 2",
			remoteLogger: mockSyslogWriter{
				mockCrit: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedIdentifier := "[test] "
						if !strings.HasPrefix(msg, expectedIdentifier) {
							t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
						}

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
			identifier: "test",
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockCrit: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			identifier:          "test",
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message 1\n\nthis is the message 2",
			identifier:          "test",
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		logger := NewLogger(scenario.identifier)
		logger.Crit(scenario.msg)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestLogger_Critf(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		arguments           []interface{}
		remoteLogger        syslogWriter
		identifier          string
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message %d\n\nthis is the message %d",
			arguments:   []interface{}{1, 2},
			remoteLogger: mockSyslogWriter{
				mockCrit: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedIdentifier := "[test] "
						if !strings.HasPrefix(msg, expectedIdentifier) {
							t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
						}

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
			identifier: "test",
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockCrit: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			identifier:          "test",
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message %d\n\nthis is the message %d",
			arguments:           []interface{}{1, 2},
			identifier:          "test",
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		logger := NewLogger(scenario.identifier)
		logger.Critf(scenario.msg, scenario.arguments...)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestLogger_Error(t *testing.T) {
	scenarios := []struct {
		description         string
		err                 error
		remoteLogger        syslogWriter
		identifier          string
		expectedlocalBuffer string
	}{
		{
			description: "it should ignore a undefined error",
		},
		{
			description: "it should log correctly a no level error message",
			err:         fmt.Errorf("generic error message"),
			remoteLogger: mockSyslogWriter{
				mockErr: func(msg string) error {
					expectedIdentifier := "[test] "
					if !strings.HasPrefix(msg, expectedIdentifier) {
						t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
					}

					expectedMsg := "generic error message"
					if !strings.HasSuffix(msg, expectedMsg) {
						t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
					}
					return nil
				},
			},
			identifier: "test",
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			err:         fmt.Errorf("generic error message"),
			remoteLogger: mockSyslogWriter{
				mockErr: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			identifier:          "test",
			expectedlocalBuffer: "generic error message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			err:                 fmt.Errorf("generic error message"),
			identifier:          "test",
			expectedlocalBuffer: "generic error message",
		},
		{
			description: "it should log correctly an emergency message",
			err: levelError{
				msg:   "error message",
				level: LevelEmergency,
			},
			remoteLogger: mockSyslogWriter{
				mockEmerg: func(msg string) error {
					expectedIdentifier := "[test] "
					if !strings.HasPrefix(msg, expectedIdentifier) {
						t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
					}

					expectedMsg := "error message"
					if !strings.HasSuffix(msg, expectedMsg) {
						t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
					}
					return nil
				},
			},
			identifier: "test",
		},
		{
			description: "it should log correctly an alert message",
			err: levelError{
				msg:   "error message",
				level: LevelAlert,
			},
			remoteLogger: mockSyslogWriter{
				mockAlert: func(msg string) error {
					expectedIdentifier := "[test] "
					if !strings.HasPrefix(msg, expectedIdentifier) {
						t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
					}

					expectedMsg := "error message"
					if !strings.HasSuffix(msg, expectedMsg) {
						t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
					}
					return nil
				},
			},
			identifier: "test",
		},
		{
			description: "it should log correctly a critical message",
			err: levelError{
				msg:   "error message",
				level: LevelCritical,
			},
			remoteLogger: mockSyslogWriter{
				mockCrit: func(msg string) error {
					expectedIdentifier := "[test] "
					if !strings.HasPrefix(msg, expectedIdentifier) {
						t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
					}

					expectedMsg := "error message"
					if !strings.HasSuffix(msg, expectedMsg) {
						t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
					}
					return nil
				},
			},
			identifier: "test",
		},
		{
			description: "it should log correctly an error message",
			err: levelError{
				msg:   "error message",
				level: LevelError,
			},
			remoteLogger: mockSyslogWriter{
				mockErr: func(msg string) error {
					expectedIdentifier := "[test] "
					if !strings.HasPrefix(msg, expectedIdentifier) {
						t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
					}

					expectedMsg := "error message"
					if !strings.HasSuffix(msg, expectedMsg) {
						t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
					}
					return nil
				},
			},
			identifier: "test",
		},
		{
			description: "it should log correctly a warning message",
			err: levelError{
				msg:   "error message",
				level: LevelWarning,
			},
			remoteLogger: mockSyslogWriter{
				mockWarning: func(msg string) error {
					expectedIdentifier := "[test] "
					if !strings.HasPrefix(msg, expectedIdentifier) {
						t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
					}

					expectedMsg := "error message"
					if !strings.HasSuffix(msg, expectedMsg) {
						t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
					}
					return nil
				},
			},
			identifier: "test",
		},
		{
			description: "it should log correctly a notice message",
			err: levelError{
				msg:   "error message",
				level: LevelNotice,
			},
			remoteLogger: mockSyslogWriter{
				mockNotice: func(msg string) error {
					expectedIdentifier := "[test] "
					if !strings.HasPrefix(msg, expectedIdentifier) {
						t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
					}

					expectedMsg := "error message"
					if !strings.HasSuffix(msg, expectedMsg) {
						t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
					}
					return nil
				},
			},
			identifier: "test",
		},
		{
			description: "it should log correctly an info message",
			err: levelError{
				msg:   "error message",
				level: LevelInfo,
			},
			remoteLogger: mockSyslogWriter{
				mockInfo: func(msg string) error {
					expectedIdentifier := "[test] "
					if !strings.HasPrefix(msg, expectedIdentifier) {
						t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
					}

					expectedMsg := "error message"
					if !strings.HasSuffix(msg, expectedMsg) {
						t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
					}
					return nil
				},
			},
			identifier: "test",
		},
		{
			description: "it should log correctly a debug message",
			err: levelError{
				msg:   "error message",
				level: LevelDebug,
			},
			remoteLogger: mockSyslogWriter{
				mockDebug: func(msg string) error {
					expectedIdentifier := "[test] "
					if !strings.HasPrefix(msg, expectedIdentifier) {
						t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
					}

					expectedMsg := "error message"
					if !strings.HasSuffix(msg, expectedMsg) {
						t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
					}
					return nil
				},
			},
			identifier: "test",
		},
		{
			description: "it should log correctly an unknown level message",
			err: levelError{
				msg:   "error message",
				level: Level(-1),
			},
			remoteLogger: mockSyslogWriter{
				mockErr: func(msg string) error {
					expectedIdentifier := "[test] "
					if !strings.HasPrefix(msg, expectedIdentifier) {
						t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
					}

					expectedMsg := "error message"
					if !strings.HasSuffix(msg, expectedMsg) {
						t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
					}
					return nil
				},
				mockWarning: func(msg string) error {
					expectedMsg := "Wrong error level: -1"
					if !strings.HasSuffix(msg, expectedMsg) {
						t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
					}
					return nil
				},
			},
			identifier: "test",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		logger := NewLogger(scenario.identifier)
		logger.Error(scenario.err)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestLogger_Errorf(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		arguments           []interface{}
		remoteLogger        syslogWriter
		identifier          string
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message %d\n\nthis is the message %d",
			arguments:   []interface{}{1, 2},
			remoteLogger: mockSyslogWriter{
				mockErr: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedIdentifier := "[test] "
						if !strings.HasPrefix(msg, expectedIdentifier) {
							t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
						}

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
			identifier: "test",
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockErr: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			identifier:          "test",
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message %d\n\nthis is the message %d",
			arguments:           []interface{}{1, 2},
			identifier:          "test",
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		logger := NewLogger(scenario.identifier)
		logger.Errorf(scenario.msg, scenario.arguments...)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestLogger_Warning(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		remoteLogger        syslogWriter
		identifier          string
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message 1\n\nthis is the message 2",
			remoteLogger: mockSyslogWriter{
				mockWarning: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedIdentifier := "[test] "
						if !strings.HasPrefix(msg, expectedIdentifier) {
							t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
						}

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
			identifier: "test",
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockWarning: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			identifier:          "test",
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message 1\n\nthis is the message 2",
			identifier:          "test",
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		logger := NewLogger(scenario.identifier)
		logger.Warning(scenario.msg)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestLogger_Warningf(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		arguments           []interface{}
		remoteLogger        syslogWriter
		identifier          string
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message %d\n\nthis is the message %d",
			arguments:   []interface{}{1, 2},
			remoteLogger: mockSyslogWriter{
				mockWarning: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedIdentifier := "[test] "
						if !strings.HasPrefix(msg, expectedIdentifier) {
							t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
						}

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
			identifier: "test",
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockWarning: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			identifier:          "test",
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message %d\n\nthis is the message %d",
			arguments:           []interface{}{1, 2},
			identifier:          "test",
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		logger := NewLogger(scenario.identifier)
		logger.Warningf(scenario.msg, scenario.arguments...)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestLogger_Notice(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		remoteLogger        syslogWriter
		identifier          string
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message 1\n\nthis is the message 2",
			remoteLogger: mockSyslogWriter{
				mockNotice: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedIdentifier := "[test] "
						if !strings.HasPrefix(msg, expectedIdentifier) {
							t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
						}

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
			identifier: "test",
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockNotice: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			identifier:          "test",
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message 1\n\nthis is the message 2",
			identifier:          "test",
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		logger := NewLogger(scenario.identifier)
		logger.Notice(scenario.msg)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestLogger_Noticef(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		arguments           []interface{}
		remoteLogger        syslogWriter
		identifier          string
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message %d\n\nthis is the message %d",
			arguments:   []interface{}{1, 2},
			remoteLogger: mockSyslogWriter{
				mockNotice: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedIdentifier := "[test] "
						if !strings.HasPrefix(msg, expectedIdentifier) {
							t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
						}

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
			identifier: "test",
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockNotice: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			identifier:          "test",
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message %d\n\nthis is the message %d",
			arguments:           []interface{}{1, 2},
			identifier:          "test",
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		logger := NewLogger(scenario.identifier)
		logger.Noticef(scenario.msg, scenario.arguments...)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestLogger_Info(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		remoteLogger        syslogWriter
		identifier          string
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message 1\n\nthis is the message 2",
			remoteLogger: mockSyslogWriter{
				mockInfo: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedIdentifier := "[test] "
						if !strings.HasPrefix(msg, expectedIdentifier) {
							t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
						}

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
			identifier: "test",
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockInfo: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			identifier:          "test",
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message 1\n\nthis is the message 2",
			identifier:          "test",
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		logger := NewLogger(scenario.identifier)
		logger.Info(scenario.msg)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestLogger_Infof(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		arguments           []interface{}
		remoteLogger        syslogWriter
		identifier          string
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message %d\n\nthis is the message %d",
			arguments:   []interface{}{1, 2},
			remoteLogger: mockSyslogWriter{
				mockInfo: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedIdentifier := "[test] "
						if !strings.HasPrefix(msg, expectedIdentifier) {
							t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
						}

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
			identifier: "test",
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockInfo: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			identifier:          "test",
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message %d\n\nthis is the message %d",
			arguments:           []interface{}{1, 2},
			identifier:          "test",
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		logger := NewLogger(scenario.identifier)
		logger.Infof(scenario.msg, scenario.arguments...)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestLogger_Debug(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		remoteLogger        syslogWriter
		identifier          string
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message 1\n\nthis is the message 2",
			remoteLogger: mockSyslogWriter{
				mockDebug: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedIdentifier := "[test] "
						if !strings.HasPrefix(msg, expectedIdentifier) {
							t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
						}

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
			identifier: "test",
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockDebug: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			identifier:          "test",
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message 1\n\nthis is the message 2",
			identifier:          "test",
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		logger := NewLogger(scenario.identifier)
		logger.Debug(scenario.msg)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestLogger_Debugf(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		arguments           []interface{}
		remoteLogger        syslogWriter
		identifier          string
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message %d\n\nthis is the message %d",
			arguments:   []interface{}{1, 2},
			remoteLogger: mockSyslogWriter{
				mockDebug: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedIdentifier := "[test] "
						if !strings.HasPrefix(msg, expectedIdentifier) {
							t.Errorf("mismatch identifier. Expecting “%s”; found “%s”", expectedIdentifier, msg)
						}

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
			identifier: "test",
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockDebug: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			identifier:          "test",
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message %d\n\nthis is the message %d",
			arguments:           []interface{}{1, 2},
			identifier:          "test",
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		logger := NewLogger(scenario.identifier)
		logger.Debugf(scenario.msg, scenario.arguments...)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestEmerg(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		remoteLogger        syslogWriter
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message 1\n\nthis is the message 2",
			remoteLogger: mockSyslogWriter{
				mockEmerg: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockEmerg: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message 1\n\nthis is the message 2",
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		Emerg(scenario.msg)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestEmergf(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		arguments           []interface{}
		remoteLogger        syslogWriter
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message %d\n\nthis is the message %d",
			arguments:   []interface{}{1, 2},
			remoteLogger: mockSyslogWriter{
				mockEmerg: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockEmerg: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message %d\n\nthis is the message %d",
			arguments:           []interface{}{1, 2},
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		Emergf(scenario.msg, scenario.arguments...)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestAlert(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		remoteLogger        syslogWriter
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message 1\n\nthis is the message 2",
			remoteLogger: mockSyslogWriter{
				mockAlert: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockAlert: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message 1\n\nthis is the message 2",
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		Alert(scenario.msg)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestAlertf(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		arguments           []interface{}
		remoteLogger        syslogWriter
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message %d\n\nthis is the message %d",
			arguments:   []interface{}{1, 2},
			remoteLogger: mockSyslogWriter{
				mockAlert: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockAlert: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message %d\n\nthis is the message %d",
			arguments:           []interface{}{1, 2},
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		Alertf(scenario.msg, scenario.arguments...)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestCrit(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		remoteLogger        syslogWriter
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message 1\n\nthis is the message 2",
			remoteLogger: mockSyslogWriter{
				mockCrit: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockCrit: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message 1\n\nthis is the message 2",
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		Crit(scenario.msg)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestCritf(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		arguments           []interface{}
		remoteLogger        syslogWriter
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message %d\n\nthis is the message %d",
			arguments:   []interface{}{1, 2},
			remoteLogger: mockSyslogWriter{
				mockCrit: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockCrit: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message %d\n\nthis is the message %d",
			arguments:           []interface{}{1, 2},
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		Critf(scenario.msg, scenario.arguments...)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestError(t *testing.T) {
	// TODO(rafaeljusto)
	scenarios := []struct {
		description         string
		err                 error
		remoteLogger        syslogWriter
		expectedlocalBuffer string
	}{
		{
			description: "it should ignore a undefined error",
		},
		{
			description: "it should log correctly a no level error message",
			err:         fmt.Errorf("generic error message"),
			remoteLogger: mockSyslogWriter{
				mockErr: func(msg string) error {
					expectedMsg := "generic error message"
					if !strings.HasSuffix(msg, expectedMsg) {
						t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
					}
					return nil
				},
			},
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			err:         fmt.Errorf("generic error message"),
			remoteLogger: mockSyslogWriter{
				mockErr: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			expectedlocalBuffer: "generic error message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			err:                 fmt.Errorf("generic error message"),
			expectedlocalBuffer: "generic error message",
		},
		{
			description: "it should log correctly an emergency message",
			err: levelError{
				msg:   "error message",
				level: LevelEmergency,
			},
			remoteLogger: mockSyslogWriter{
				mockEmerg: func(msg string) error {
					expectedMsg := "error message"
					if !strings.HasSuffix(msg, expectedMsg) {
						t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
					}
					return nil
				},
			},
		},
		{
			description: "it should log correctly an alert message",
			err: levelError{
				msg:   "error message",
				level: LevelAlert,
			},
			remoteLogger: mockSyslogWriter{
				mockAlert: func(msg string) error {
					expectedMsg := "error message"
					if !strings.HasSuffix(msg, expectedMsg) {
						t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
					}
					return nil
				},
			},
		},
		{
			description: "it should log correctly a critical message",
			err: levelError{
				msg:   "error message",
				level: LevelCritical,
			},
			remoteLogger: mockSyslogWriter{
				mockCrit: func(msg string) error {
					expectedMsg := "error message"
					if !strings.HasSuffix(msg, expectedMsg) {
						t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
					}
					return nil
				},
			},
		},
		{
			description: "it should log correctly an error message",
			err: levelError{
				msg:   "error message",
				level: LevelError,
			},
			remoteLogger: mockSyslogWriter{
				mockErr: func(msg string) error {
					expectedMsg := "error message"
					if !strings.HasSuffix(msg, expectedMsg) {
						t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
					}
					return nil
				},
			},
		},
		{
			description: "it should log correctly a warning message",
			err: levelError{
				msg:   "error message",
				level: LevelWarning,
			},
			remoteLogger: mockSyslogWriter{
				mockWarning: func(msg string) error {
					expectedMsg := "error message"
					if !strings.HasSuffix(msg, expectedMsg) {
						t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
					}
					return nil
				},
			},
		},
		{
			description: "it should log correctly a notice message",
			err: levelError{
				msg:   "error message",
				level: LevelNotice,
			},
			remoteLogger: mockSyslogWriter{
				mockNotice: func(msg string) error {
					expectedMsg := "error message"
					if !strings.HasSuffix(msg, expectedMsg) {
						t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
					}
					return nil
				},
			},
		},
		{
			description: "it should log correctly an info message",
			err: levelError{
				msg:   "error message",
				level: LevelInfo,
			},
			remoteLogger: mockSyslogWriter{
				mockInfo: func(msg string) error {
					expectedMsg := "error message"
					if !strings.HasSuffix(msg, expectedMsg) {
						t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
					}
					return nil
				},
			},
		},
		{
			description: "it should log correctly a debug message",
			err: levelError{
				msg:   "error message",
				level: LevelDebug,
			},
			remoteLogger: mockSyslogWriter{
				mockDebug: func(msg string) error {
					expectedMsg := "error message"
					if !strings.HasSuffix(msg, expectedMsg) {
						t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
					}
					return nil
				},
			},
		},
		{
			description: "it should log correctly an unknown level message",
			err: levelError{
				msg:   "error message",
				level: Level(-1),
			},
			remoteLogger: mockSyslogWriter{
				mockErr: func(msg string) error {
					expectedMsg := "error message"
					if !strings.HasSuffix(msg, expectedMsg) {
						t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
					}
					return nil
				},
				mockWarning: func(msg string) error {
					expectedMsg := "Wrong error level: -1"
					if !strings.HasSuffix(msg, expectedMsg) {
						t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
					}
					return nil
				},
			},
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		Error(scenario.err)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestErrorf(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		arguments           []interface{}
		remoteLogger        syslogWriter
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message %d\n\nthis is the message %d",
			arguments:   []interface{}{1, 2},
			remoteLogger: mockSyslogWriter{
				mockErr: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockErr: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message %d\n\nthis is the message %d",
			arguments:           []interface{}{1, 2},
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		Errorf(scenario.msg, scenario.arguments...)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestWarning(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		remoteLogger        syslogWriter
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message 1\n\nthis is the message 2",
			remoteLogger: mockSyslogWriter{
				mockWarning: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockWarning: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message 1\n\nthis is the message 2",
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		Warning(scenario.msg)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestWarningf(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		arguments           []interface{}
		remoteLogger        syslogWriter
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message %d\n\nthis is the message %d",
			arguments:   []interface{}{1, 2},
			remoteLogger: mockSyslogWriter{
				mockWarning: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockWarning: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message %d\n\nthis is the message %d",
			arguments:           []interface{}{1, 2},
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		Warningf(scenario.msg, scenario.arguments...)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestNotice(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		remoteLogger        syslogWriter
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message 1\n\nthis is the message 2",
			remoteLogger: mockSyslogWriter{
				mockNotice: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockNotice: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message 1\n\nthis is the message 2",
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		Notice(scenario.msg)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestNoticef(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		arguments           []interface{}
		remoteLogger        syslogWriter
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message %d\n\nthis is the message %d",
			arguments:   []interface{}{1, 2},
			remoteLogger: mockSyslogWriter{
				mockNotice: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockNotice: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message %d\n\nthis is the message %d",
			arguments:           []interface{}{1, 2},
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		Noticef(scenario.msg, scenario.arguments...)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestInfo(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		remoteLogger        syslogWriter
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message 1\n\nthis is the message 2",
			remoteLogger: mockSyslogWriter{
				mockInfo: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockInfo: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message 1\n\nthis is the message 2",
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		Info(scenario.msg)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestInfof(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		arguments           []interface{}
		remoteLogger        syslogWriter
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message %d\n\nthis is the message %d",
			arguments:   []interface{}{1, 2},
			remoteLogger: mockSyslogWriter{
				mockInfo: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockInfo: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message %d\n\nthis is the message %d",
			arguments:           []interface{}{1, 2},
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		Infof(scenario.msg, scenario.arguments...)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestDebug(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		remoteLogger        syslogWriter
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message 1\n\nthis is the message 2",
			remoteLogger: mockSyslogWriter{
				mockDebug: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockDebug: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message 1\n\nthis is the message 2",
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		Debug(scenario.msg)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

func TestDebugf(t *testing.T) {
	scenarios := []struct {
		description         string
		msg                 string
		arguments           []interface{}
		remoteLogger        syslogWriter
		expectedlocalBuffer string
	}{
		{
			description: "it should log correctly a multiline message",
			msg:         "this is the message %d\n\nthis is the message %d",
			arguments:   []interface{}{1, 2},
			remoteLogger: mockSyslogWriter{
				mockDebug: func() func(msg string) error {
					i := 0
					return func(msg string) error {
						i++

						expectedMsg := fmt.Sprintf("this is the message %d", i)
						if !strings.HasSuffix(msg, expectedMsg) {
							t.Errorf("mismatch message. Expecting “%s”; found “%s”", expectedMsg, msg)
						}
						return nil
					}
				}(),
			},
		},
		{
			description: "it should fallback to a local buffer when there's an error writing to syslog",
			msg:         "this is a message",
			remoteLogger: mockSyslogWriter{
				mockDebug: func(msg string) error {
					return fmt.Errorf("error detected")
				},
			},
			expectedlocalBuffer: "this is a message",
		},
		{
			description:         "it should use a local buffer when there's no syslog connection",
			msg:                 "this is the message %d\n\nthis is the message %d",
			arguments:           []interface{}{1, 2},
			expectedlocalBuffer: "this is the message 2",
		},
	}

	originalRemoteLogger := remoteLogger
	defer func() {
		remoteLogger = originalRemoteLogger
	}()

	var localBuffer bytes.Buffer
	localLogger = log.New(&localBuffer, "", log.Lshortfile)

	for i, scenario := range scenarios {
		localBuffer.Reset()
		remoteLogger = scenario.remoteLogger

		Debugf(scenario.msg, scenario.arguments...)

		localMessage := strings.TrimSpace(localBuffer.String())
		if scenario.expectedlocalBuffer != "" && !strings.HasSuffix(localMessage, scenario.expectedlocalBuffer) {
			t.Errorf("scenario %d, “%s”: mismatch message. Expecting “%s”; found “%s”",
				i, scenario.description, scenario.expectedlocalBuffer, localMessage,
			)
		}
	}
}

type mockSyslogWriter struct {
	mockClose   func() error
	mockEmerg   func(msg string) (err error)
	mockAlert   func(msg string) (err error)
	mockCrit    func(msg string) (err error)
	mockErr     func(msg string) (err error)
	mockWarning func(msg string) (err error)
	mockNotice  func(msg string) (err error)
	mockInfo    func(msg string) (err error)
	mockDebug   func(msg string) (err error)
}

func (m mockSyslogWriter) Close() error {
	return m.mockClose()
}

func (m mockSyslogWriter) Emerg(msg string) (err error) {
	return m.mockEmerg(msg)
}

func (m mockSyslogWriter) Alert(msg string) (err error) {
	return m.mockAlert(msg)
}

func (m mockSyslogWriter) Crit(msg string) (err error) {
	return m.mockCrit(msg)
}

func (m mockSyslogWriter) Err(msg string) (err error) {
	return m.mockErr(msg)
}

func (m mockSyslogWriter) Warning(msg string) (err error) {
	return m.mockWarning(msg)
}

func (m mockSyslogWriter) Notice(msg string) (err error) {
	return m.mockNotice(msg)
}

func (m mockSyslogWriter) Info(msg string) (err error) {
	return m.mockInfo(msg)
}

func (m mockSyslogWriter) Debug(msg string) (err error) {
	return m.mockDebug(msg)
}

type levelError struct {
	msg   string
	level Level
}

func (l levelError) Error() string {
	return l.msg
}

func (l levelError) Level() Level {
	return l.level
}
