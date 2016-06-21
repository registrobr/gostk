package errors_test

import (
	"fmt"
	"testing"

	"github.com/registrobr/gostk/errors"
)

func TestEqual(t *testing.T) {
	scenarios := []struct {
		description string
		err1        error
		err2        error
		expected    bool
	}{
		{
			description: "it should compare correctly 2 equal traceable errors",
			err1:        errors.Errorf("this is a test"),
			err2:        errors.Errorf("this is a test"),
			expected:    true,
		},
		{
			description: "it should detect when 2 traceable errors are different",
			err1:        errors.Errorf("this is a test 1"),
			err2:        errors.Errorf("this is a test 2"),
			expected:    false,
		},
		{
			description: "it should compare correctly when there's only 1 traceable error (1)",
			err1:        errors.Errorf("this is a test"),
			err2:        fmt.Errorf("this is a test"),
			expected:    true,
		},
		{
			description: "it should compare correctly when there's only 1 traceable error (2)",
			err1:        fmt.Errorf("this is a test"),
			err2:        errors.Errorf("this is a test"),
			expected:    true,
		},
		{
			description: "it should detect when errors are different (1)",
			err1:        errors.Errorf("this is a test 1"),
			err2:        fmt.Errorf("this is a test 2"),
			expected:    false,
		},
		{
			description: "it should detect when errors are different (2)",
			err1:        fmt.Errorf("this is a test 1"),
			err2:        errors.Errorf("this is a test 2"),
			expected:    false,
		},
		{
			description: "it should compare correctly when there's no traceable error (1)",
			err1:        fmt.Errorf("this is a test"),
			err2:        fmt.Errorf("this is a test"),
			expected:    true,
		},
		{
			description: "it should compare correctly when there's no traceable error (2)",
			expected:    true,
		},
		{
			description: "it should detect when errors are different (3)",
			err1:        fmt.Errorf("this is a test"),
			err2:        nil,
			expected:    false,
		},
		{
			description: "it should detect when errors are different (4)",
			err1:        nil,
			err2:        fmt.Errorf("this is a test"),
			expected:    false,
		},
	}

	for i, scenario := range scenarios {
		if errors.Equal(scenario.err1, scenario.err2) != scenario.expected {
			t.Errorf("scenario %d, “%s”: mismatch results. Expecting: “%v”",
				i, scenario.description, scenario.expected)
		}
	}
}
