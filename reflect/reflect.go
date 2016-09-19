// Package reflect adds some useful features to the standard reflect package.
package reflect

import "reflect"

// IsDefined checks if the value is different from nil looking further to its
// contents.
func IsDefined(value interface{}) bool {
	if value == nil {
		return false
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if canIsNil(v.Elem()) {
			v = v.Elem()
		}
	}

	if canIsNil(v) {
		return !v.IsNil()
	}

	return true
}

func canIsNil(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:
		return true
	}

	return false
}
