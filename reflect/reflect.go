// Package reflect adds some useful features to the standard reflect package.
package reflect

import "reflect"

// IsDefined checks if the value is different from nil looking further to its
// contents.
func IsDefined(value interface{}) bool {
	if value == nil {
		return false
	}

	return isDefined(reflect.ValueOf(value))
}

func isDefined(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Ptr, reflect.Interface:
		if canIsNil(value.Elem()) {
			return isDefined(value.Elem())
		}
	}

	if canIsNil(value) {
		return !value.IsNil()
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
