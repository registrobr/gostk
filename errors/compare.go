// Package errors adds location and log levels to errors.
package errors

// Equal compares the errors messages.
func Equal(first, second error) bool {
	if first == nil || second == nil {
		return first == second
	}

	err1, ok1 := first.(traceableError)
	err2, ok2 := second.(traceableError)

	if ok1 {
		if ok2 {
			return err1.err.Error() == err2.err.Error()
		}

		return err1.err.Error() == second.Error()

	}

	if ok2 {
		return err2.err.Error() == first.Error()
	}

	return first.Error() == second.Error()
}
