package promise

import (
	"fmt"
	"github.com/thoas/go-funk"
	"reflect"
)

/*

		func() {},
		func() (error) { return nil },
		func() (interface{}, error) { return nil, nil },
		func() ([]interface{}, error) { return nil, nil },
		func() (*Promise) { return nil },
		func() (*Promise, error) { return nil, nil },

		func(resolve func(...interface{}), reject func(error)) {},
		func(resolve func(...interface{}), reject func(error), values ...interface{}) {},
		func(value interface{}, resolve func(), reject func(error)) {},
		func(value interface{}, resolve func(...interface{}), reject func(error)) {},
		func(error, resolve func(...interface{}), reject func(error)) {},

		func(values ...interface{}) {},
		func(values ...interface{}) (error) { return nil },
		func(values ...interface{}) (interface{}, error) { return nil, nil },
		func(values ...interface{}) ([]interface{}, error) { return nil, nil },
		func(values ...interface{}) (*Promise) { return nil },
		func(values ...interface{}) (*Promise, error) { return nil, nil },

		func(error) {},
		func(error) (error) { return nil },
		func(error) (interface{}, error) { return nil, nil },
		func(error) ([]interface{}, error) { return nil, nil },
		func(error) (*Promise) { return nil },
		func(error) (*Promise, error) { return nil, nil },
 */

type funcSignature struct {
	name string
	fun interface{}
}

func signature(function interface{}) funcSignature {
	return funcSignature{
		fmt.Sprint(function),
		function,
	}
}

var (
	_FUNC_IN_OUT = signature(func() {})
	_FUNC_IN_OUT_ERROR = signature(func() (error) { return nil })
	_FUNC_IN_OUT_OBJ_ERROR = signature(func() (interface{}, error) { return nil, nil })
	_FUNC_IN_OUT_OBJS_ERROR = signature(func() ([]interface{}, error) { return nil, nil })
	_FUNC_IN_OUT_PROMISE = signature(func() (*Promise) { return nil })
	_FUNC_IN_OUT_PROMISE_ERROR = signature(func() (*Promise, error) { return nil, nil })

	_FUNC_IN_RESOLVE_OBJS_REJECT_ERROR_OUT = signature(func(resolve func(...interface{}), reject func(error)) {})
	_FUNC_IN_RESOLVE_OBJS_REJECT_ERROR_VARIADIC_OBJS_OUT = signature(func(resolve func(...interface{}), reject func(error), values ...interface{}) {})
	_FUNC_IN_OBJ_RESOLVE_REJECT_ERROR_OUT = signature(func(value interface{}, resolve func(), reject func(error)) {})
	_FUNC_IN_OBJS_RESOLVE_OBJS_REJECT_ERROR_OUT = signature(func(value interface{}, resolve func(...interface{}), reject func(error)) {})
	_FUNC_IN_ERROR_RESOLVE_OBJS_REJECT_ERROR_OUT = signature(func(error, resolve func(...interface{}), reject func(error)) {})

	_FUNC_IN_VARIADIC_OBJS_OUT = signature(func(values ...interface{}) {})
	_FUNC_IN_VARIADIC_OBJS_OUT_ERROR = signature(func(values ...interface{}) (error) { return nil })
	_FUNC_IN_VARIADIC_OBJS_OUT_OBJ_ERROR = signature(func(values ...interface{}) (interface{}, error) { return nil, nil })
	_FUNC_IN_VARIADIC_OBJS_OUT_OBJS_ERROR = signature(func(values ...interface{}) ([]interface{}, error) { return nil, nil })
	_FUNC_IN_VARIADIC_OBJS_OUT_PROMISE = signature(func(values ...interface{}) (*Promise) { return nil })
	_FUNC_IN_VARIADIC_OBJS_OUT_PROMISE_ERROR = signature(func(values ...interface{}) (*Promise, error) { return nil, nil })

	_FUNC_IN_ERROR_OUT = signature(func(error) {})
	_FUNC_IN_ERROR_OUT_ERROR = signature(func(error) (error) { return nil })
	_FUNC_IN_ERROR_OUT_OBJ_ERROR = signature(func(error) (interface{}, error) { return nil, nil })
	_FUNC_IN_ERROR_OUT_OBJS_ERROR = signature(func(error) ([]interface{}, error) { return nil, nil })
	_FUNC_IN_ERROR_OUT_PROMISE = signature(func(error) (*Promise) { return nil })
	_FUNC_IN_ERROR_OUT_PROMISE_ERROR = signature(func(error) (*Promise, error) { return nil, nil })
)

var RESOLVER_TYPE = reflect.TypeOf(func(...interface{}) {})
var REJECTOR_TYPE = reflect.TypeOf(func(error) {})
var ERROR_TYPE = getFunctionNthParamType(func(error) {}, 1)
var PROMISE_TYPE = getFunctionNthParamType(func(Promise) {}, 1)
var withType = func(verifier func(reflect.Type) bool) (func(interface{}) bool) {
	return func(function interface{}) bool {
		return verifier(reflect.TypeOf(function))
	}
}
var signatureVerifiers = map[string]func(interface{}) bool {
	// func()
	_FUNC_IN_OUT.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() == 0 && funcType.NumOut() == 0
	}),
	// func() (error)
	_FUNC_IN_OUT_ERROR.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() == 0 && funcType.NumOut() == 1 && funcType.Out(0).Implements(ERROR_TYPE)
	}),
	// func() (interface{}, error)
	_FUNC_IN_OUT_OBJ_ERROR.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() == 0 && funcType.NumOut() == 2 &&
			!funcType.Out(0).Implements(ERROR_TYPE) &&
			!funcType.Out(0).Implements(PROMISE_TYPE) &&
			funcType.Out(1).Implements(ERROR_TYPE)
	}),
	// func() ([]interface{}, error)
	_FUNC_IN_OUT_OBJS_ERROR.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() == 0 && funcType.NumOut() == 2 &&
			funcType.Out(0).Kind() == reflect.Slice &&
			funcType.Out(1).Implements(ERROR_TYPE)
	}),
	// func() (*Promise)
	_FUNC_IN_OUT_PROMISE.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() == 0 && funcType.NumOut() == 1 &&
			funcType.Out(0).Implements(PROMISE_TYPE)
	}),
	// func() (*Promise, error)
	_FUNC_IN_OUT_PROMISE_ERROR.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() == 0 && funcType.NumOut() == 2 &&
			funcType.Out(0).Implements(PROMISE_TYPE) &&
			funcType.Out(1).Implements(ERROR_TYPE)
	}),

	// func(resolve func(...interface{}), reject func(error))
	_FUNC_IN_RESOLVE_OBJS_REJECT_ERROR_OUT.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() == 2 && funcType.NumOut() == 0 &&
			funcType.In(0).Kind() == reflect.Func &&
			funcType.In(0).NumIn() > 0 &&
			funcType.In(1).Kind() == reflect.Func &&
			funcType.In(1).NumIn() == 1 &&
			funcType.In(1).In(0).Implements(ERROR_TYPE)
	}),
	// func(resolve func(...interface{}), reject func(error), values ...interface{})
	_FUNC_IN_RESOLVE_OBJS_REJECT_ERROR_VARIADIC_OBJS_OUT.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() == 3 && funcType.NumOut() == 0 &&
			func (valueType reflect.Type) bool {
				return valueType.Kind() != reflect.Slice &&
					!valueType.Implements(ERROR_TYPE)
			}(funcType.In(0)) && func (resolveType reflect.Type) bool {
			return resolveType.Kind() == reflect.Func &&
				resolveType.NumIn() > 0
		}(funcType.In(1)) && func (rejectType reflect.Type) bool {
			return rejectType.Kind() == reflect.Func &&
				rejectType.NumIn() == 1 &&
				rejectType.Implements(ERROR_TYPE)
		}(funcType.In(2))
	}),
	// func(value interface{}, resolve func(), reject func(error))
	_FUNC_IN_OBJ_RESOLVE_REJECT_ERROR_OUT.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() == 3 && funcType.NumOut() == 0 &&
			funcType.In(0).Kind() == reflect.Slice &&
			func (resolveType reflect.Type) bool {
				return resolveType.Kind() == reflect.Func &&
					resolveType.NumIn() == 0
			}(funcType.In(1)) && func (rejectType reflect.Type) bool {
			return rejectType.Kind() == reflect.Func &&
				rejectType.NumIn() == 1 &&
				rejectType.Implements(ERROR_TYPE)
		}(funcType.In(2))
	}),
	// func(value interface{}, resolve func(...interface{}), reject func(error))
	_FUNC_IN_OBJS_RESOLVE_OBJS_REJECT_ERROR_OUT.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() == 3 && funcType.NumOut() == 0 &&
			funcType.In(0).Kind() == reflect.Slice &&
			func (resolveType reflect.Type) bool {
				return resolveType.Kind() == reflect.Func &&
					resolveType.NumIn() > 0
			}(funcType.In(1)) && func (rejectType reflect.Type) bool {
			return rejectType.Kind() == reflect.Func &&
				rejectType.NumIn() == 1 &&
				rejectType.Implements(ERROR_TYPE)
		}(funcType.In(2))
	}),
	// func(error, resolve func(...interface{}), reject func(error))
	_FUNC_IN_ERROR_RESOLVE_OBJS_REJECT_ERROR_OUT.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() == 3 && funcType.NumOut() == 0 &&
			funcType.In(0).Implements(ERROR_TYPE) &&
			func (resolveType reflect.Type) bool {
				return resolveType.Kind() == reflect.Func &&
					resolveType.NumIn() > 0
			}(funcType.In(1)) && func (rejectType reflect.Type) bool {
			return rejectType.Kind() == reflect.Func &&
				rejectType.NumIn() == 1 &&
				rejectType.Implements(ERROR_TYPE)
		}(funcType.In(2))
	}),

	// func(values ...interface{})
	_FUNC_IN_VARIADIC_OBJS_OUT.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() > 0 && funcType.NumOut() == 0 &&
			!funcType.In(0).Implements(ERROR_TYPE)
	}),
	// func(values ...interface{}) (error)
	_FUNC_IN_VARIADIC_OBJS_OUT_ERROR.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() > 0 && funcType.NumOut() == 1 &&
			!funcType.In(0).Implements(ERROR_TYPE) &&
			funcType.Out(1).Implements(ERROR_TYPE)
	}),
	// func(values ...interface{}) (interface{}, error)
	_FUNC_IN_VARIADIC_OBJS_OUT_OBJ_ERROR.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() > 0 && funcType.NumOut() == 2 &&
			!funcType.In(0).Implements(ERROR_TYPE) &&
			funcType.Out(0).Kind() != reflect.Slice &&
			!funcType.Out(0).Implements(PROMISE_TYPE) &&
			funcType.Out(1).Implements(ERROR_TYPE)
	}),
	// func(values ...interface{}) ([]interface{}, error)
	_FUNC_IN_VARIADIC_OBJS_OUT_OBJS_ERROR.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() > 0 && funcType.NumOut() == 2 &&
			!funcType.In(0).Implements(ERROR_TYPE) &&
			funcType.Out(0).Kind() == reflect.Slice &&
			funcType.Out(1).Implements(ERROR_TYPE)
	}),
	// func(values ...interface{}) (*Promise)
	_FUNC_IN_VARIADIC_OBJS_OUT_PROMISE.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() > 0 && funcType.NumOut() == 1 &&
			!funcType.In(0).Implements(ERROR_TYPE) &&
			funcType.Out(0).Implements(PROMISE_TYPE)
	}),
	// func(values ...interface{}) (*Promise, error)
	_FUNC_IN_VARIADIC_OBJS_OUT_PROMISE_ERROR.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() > 0 && funcType.NumOut() == 2 &&
			!funcType.In(0).Implements(ERROR_TYPE) &&
			funcType.Out(0).Implements(PROMISE_TYPE) &&
			funcType.Out(1).Implements(ERROR_TYPE)
	}),

	// func(error)
	_FUNC_IN_ERROR_OUT.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() == 1 && funcType.NumOut() == 0 &&
			funcType.In(0).Implements(ERROR_TYPE)
	}),
	// func(error) (error)
	_FUNC_IN_ERROR_OUT_ERROR.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() == 1 && funcType.NumOut() == 1 &&
			funcType.In(0).Implements(ERROR_TYPE) &&
			funcType.Out(0).Implements(ERROR_TYPE)
	}),
	// func(error) (interface{}, error)
	_FUNC_IN_ERROR_OUT_OBJ_ERROR.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() == 1 && funcType.NumOut() == 2 &&
			funcType.In(0).Implements(ERROR_TYPE) &&
			funcType.Out(0).Kind() != reflect.Slice &&
			!funcType.Out(0).Implements(PROMISE_TYPE) &&
			funcType.Out(1).Implements(ERROR_TYPE)
	}),
	// func(error) ([]interface{}, error)
	_FUNC_IN_ERROR_OUT_OBJS_ERROR.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() == 1 && funcType.NumOut() == 2 &&
			funcType.In(0).Implements(ERROR_TYPE) &&
			funcType.Out(0).Kind() == reflect.Slice &&
			funcType.Out(1).Implements(ERROR_TYPE)
	}),
	// func(error) (*Promise)
	_FUNC_IN_ERROR_OUT_PROMISE.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() == 1 && funcType.NumOut() == 1 &&
			funcType.In(0).Implements(ERROR_TYPE) &&
			funcType.Out(0).Implements(PROMISE_TYPE)
	}),
	// func(error) (*Promise, error)
	_FUNC_IN_ERROR_OUT_PROMISE_ERROR.name: withType(func(funcType reflect.Type) bool {
		return funcType.NumIn() == 1 && funcType.NumOut() == 2 &&
			funcType.In(0).Implements(ERROR_TYPE) &&
			funcType.Out(0).Implements(PROMISE_TYPE) &&
			funcType.Out(1).Implements(ERROR_TYPE)
	}),
}

func assertFunctionSignature(function interface{}, signatures ...funcSignature) {
	if !IsFunc(function) { panic("") } // TODO:

	for _, signature := range signatures {
		isSignatureValid := signatureVerifiers[signature.name]
		if IsFunc(function) && isSignatureValid(function) {
			return
		}
	}
	panic("") // TODO:
}

func getFunctionNthParamType(function interface{}, nth int) reflect.Type {
	if !IsFunc(function) { panic("Parameter is not a function definition") }
	funcType := reflect.TypeOf(function)
	if funcType.NumIn() < nth { panic("Function in the parameter doesn't have a single parameter in the signature") }
	return funcType.In(nth - 1)
}

func isValidCallbackSignature(callback interface{}) bool {
	return IsFunc(callback) && funk.Find(funk.Values(signatureVerifiers), func(verifier func(function interface{}) bool) bool {
		return verifier(callback)
	}) != nil
}