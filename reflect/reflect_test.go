package reflect_test

import (
	"testing"

	"fmt"

	"github.com/registrobr/gostk/reflect"
)

func TestIsDefined(t *testing.T) {
	scenarios := []struct {
		description string
		value       interface{}
		expected    bool
	}{
		{
			description: "it should detect nil",
			expected:    false,
		},
		{
			description: "it should detect an undefined pointer",
			value: func() interface{} {
				var value *int
				return value
			}(),
			expected: false,
		},
		{
			description: "it should detect an undefined slice",
			value: func() interface{} {
				var value []int
				return value
			}(),
			expected: false,
		},
		{
			description: "it should detect a type that refers to an undefined slice",
			value: func() interface{} {
				type t []int
				var value t
				return value
			}(),
			expected: false,
		},
		{
			description: "it should detect a pointer of a type that refers to an undefined slice",
			value: func() interface{} {
				type t []int
				var value1 t
				var value2 = &value1
				return value2
			}(),
			expected: false,
		},
		{
			description: "it should detect a defined struct",
			value:       struct{ value int }{},
			expected:    true,
		},
		{
			description: "it should detect a pointer to a defined struct",
			value:       &struct{ value int }{},
			expected:    true,
		},
		{
			description: "it should detect a pointer of a type that refers to a defined slice",
			value: func() interface{} {
				type t []int
				value1 := t{1, 2, 3}
				var value2 = &value1
				return value2
			}(),
			expected: true,
		},
	}

	for i, scenario := range scenarios {
		if result := reflect.IsDefined(scenario.value); result != scenario.expected {
			t.Errorf("scenario %d, “%s”: mismatch result. Expecting: “%t”; found “%t”",
				i, scenario.description, scenario.expected, result)
		}
	}
}

// ExampleIsDefined show the common case where there's a need to inspect deeper
// if a value is nil or not.
func ExampleIsDefined() {
	type mytype []int
	var value1 mytype
	var value2 = &value1

	fmt.Println(reflect.IsDefined(value2))
	// Output: false
}
