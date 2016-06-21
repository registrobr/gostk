// Package errors implements functions to manipulate errors adding the location
// and log levels.
package errors

// EqualMsg compares the errors messages.
func EqualMsg(first, second error) bool {
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
