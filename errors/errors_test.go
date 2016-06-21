package errors_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/registrobr/gostk/errors"
	"github.com/registrobr/gostk/log"
)

func TestNew(t *testing.T) {
	scenarios := []struct {
		description   string
		err           error
		expectedError error
	}{
		{
			description:   "it should encapsulate another error correctly",
			err:           fmt.Errorf("this is a test"),
			expectedError: errors.Errorf("this is a test"),
		},
		{
			description: "it should ignore a nil error",
		},
	}

	for i, scenario := range scenarios {
		err := errors.New(scenario.err)

		if !errors.Equal(scenario.expectedError, err) {
			t.Errorf("scenario %d, “%s”: mismatch results. Expecting: “%v”; found “%v”",
				i, scenario.description, scenario.expectedError, err)
		}
	}
}

func TestTraceableError_Error(t *testing.T) {
	scenarios := []struct {
		description string
		err         error
		expected    *regexp.Regexp
	}{
		{
			description: "it should print an error correctly",
			err:         errors.Errorf("this is a test"),
			expected:    regexp.MustCompile("^gostk/errors/errors_test.go:[0-9]+: this is a test$"),
		},
		{
			description: "it should print a stacktrace error correctly",
			err:         errors.New(errors.Errorf("this is a test")),
			expected:    regexp.MustCompile("^gostk/errors/errors_test.go:[0-9]+ → gostk/errors/errors_test.go:[0-9]+: this is a test$"),
		},
	}

	for i, scenario := range scenarios {
		result := scenario.err.Error()

		if !scenario.expected.MatchString(result) {
			t.Errorf("scenario %d, “%s”: mismatch results. Expecting: “%v”; found “%v”",
				i, scenario.description, scenario.expected.String(), result)
		}
	}
}

func TestTraceableError_Level(t *testing.T) {
	type leveler interface {
		Level() log.Level
	}

	scenarios := []struct {
		description string
		err         leveler
		expected    log.Level
	}{
		{
			description: "it should set an emergency level",
			err:         errors.Emergf("this is a test").(leveler),
			expected:    log.LevelEmergency,
		},
		{
			description: "it should set an alert level",
			err:         errors.Alertf("this is a test").(leveler),
			expected:    log.LevelAlert,
		},
		{
			description: "it should set a critical level",
			err:         errors.Critf("this is a test").(leveler),
			expected:    log.LevelCritical,
		},
		{
			description: "it should set an error level",
			err:         errors.Errorf("this is a test").(leveler),
			expected:    log.LevelError,
		},
	}

	for i, scenario := range scenarios {
		if level := scenario.err.Level(); scenario.expected != scenario.err.Level() {
			t.Errorf("scenario %d, “%s”: mismatch results. Expecting: “%v”; found “%v”",
				i, scenario.description, scenario.expected, level)
		}
	}
}
