package promise

import "reflect"

type typedError interface {
	error

	IsTypeOf(reflect.Type) bool
}

type protoTypedError struct {
	error
}

var typedErrorType = reflect.TypeOf(func(typedError) {}).In(0)

func (e protoTypedError) IsTypeOf(typ reflect.Type) bool {
	var	currentErr typedError = e
	currentErrorType := reflect.TypeOf(currentErr)
	return currentErrorType == typ ||
		(e.error != nil && reflect.TypeOf(e.error).Implements(typedErrorType) && e.error.(typedError).IsTypeOf(typ))
}

func Typed(err error) typedError {
	return protoTypedError{err}
}