package promise

import (
	"fmt"
	"github.com/thoas/go-funk"
	"reflect"
)

type callbackState string

const (
	// When the final value is not available yet. This is the only state that may transition to one of the other two states.
	STATE_PENDING = callbackState("PENDING")

	// When and if the final value becomes available. A fulfillment value becomes permanently associated with the promise.
	// This may be any value, including undefined.
	STATE_FULFILLED = callbackState("FULFILLED")

	// If an error prevented the final value from being determined.  A rejection reason becomes permanently associated with
	// the promise. This may be any value, including undefined, though it is generally an Error object, like in exception handling.
	STATE_REJECTED = callbackState("REJECTED")
)

type callback struct {
	callback interface{}
	callbackInParamTypes []reflect.Type
	callbackOutParamTypes []reflect.Type
	isResolveRejectPresent struct{
		bool
		resolveIndex int
		rejectIndex int
	}

	isSignatureValidated bool
	isReturningPromise bool
	isReturningError bool
}

func newCallback(function interface{}) *callback {
	if !IsFunc(function) { panic("callback is not a function") }
	if !isValidCallbackSignature(function) { panic("callback signature is invalid") }

	var inParamTypes, outParamTypes []reflect.Type
	var isResolveRejectPresent, isReturningPromise, isReturningError bool
	var resolveIndex, rejectIndex int
	funcType := reflect.TypeOf(function)

	inParamTypes = extractParameterTypes(funcType.In, funcType.NumIn())

	if funcType.NumOut() == 0 {
		if funcType.NumIn() >= 2 {
			var found bool
			if resolveIndex, found = findResolveParameterIndex(funcType); found {
				if rejectIndex, found = findRejectParameterIndex(funcType); found {
					resolveFunc := funcType.In(resolveIndex)
					rejectFunc := funcType.In(rejectIndex)
					isResolveRejectPresent = true

					inParamTypes = append(inParamTypes[:resolveIndex], inParamTypes[rejectIndex+1:]...)
					outParamTypes = extractParameterTypes(resolveFunc.In, resolveFunc.NumIn())
					outParamTypes = append(outParamTypes, extractParameterTypes(rejectFunc.In, rejectFunc.NumIn())...)
				}
			}

			if !found {

			}
		}
	} else {
		outParamTypes = extractParameterTypes(funcType.Out, funcType.NumOut())
		outParamTypesLen := len(outParamTypes)
		if outParamTypesLen > 0 && outParamTypes[0].Implements(PROMISE_TYPE) {
			isReturningPromise = true
		}
		if outParamTypesLen > 0 && outParamTypes[outParamTypesLen - 1].Implements(ERROR_TYPE) {
			isReturningError = true
		}
	}

	return &callback{
		callback: function,
		callbackInParamTypes: inParamTypes,
		callbackOutParamTypes: outParamTypes,
		isResolveRejectPresent: struct {
			bool
			resolveIndex int
			rejectIndex  int
		}{isResolveRejectPresent, resolveIndex, rejectIndex},

		isReturningPromise: isReturningPromise,
		isReturningError: isReturningError,
	}
}

func (callback *callback) call(params ...reflect.Value) (func(func(error, ...reflect.Value))) {
	return func(completed func(error, ...reflect.Value)) {
		var (
			results []reflect.Value
			err error
		)

		if callback.isResolveRejectPresent.bool {
			await := make(chan interface{}, 1)

			resolve := reflect.ValueOf(func(values ...interface{}) {
				if isChanClosed(await) {
					// TODO: print or log something out
					return
				}
				close(await)

				for _, value := range values {
					results = append(results, reflect.ValueOf(value))
				}

				completed(nil, results...)
			})
			reject := reflect.ValueOf(func(err error) {
				if isChanClosed(await) {
					// TODO: print or log something out
					return
				}
				close(await)

				completed(err)
			})

			params = insertIntoSlice(params, resolve, callback.isResolveRejectPresent.resolveIndex).([]reflect.Value)
			params = insertIntoSlice(params, reject, callback.isResolveRejectPresent.rejectIndex).([]reflect.Value)

			go reflect.ValueOf(callback.callback).Call(params)
		} else {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Failed to execute: %s\nWith params: %s\n", callback.callback, fmt.Sprint(funk.Map(params, func(value reflect.Value) []interface{} {
						return []interface{}{value.Type(), value.Interface()}
					}).([][]interface{})))
					panic(r)
				}
			}()
			if params != nil {
				params = params[:len(callback.callbackInParamTypes)]
			}
			results = reflect.ValueOf(callback.callback).Call(params)
			if callback.isReturningError {
				results, err = extractError(results)
			}

			completed(err, results...)
		}
	}
}

func findResolveParameterIndex(funcType reflect.Type) (index int, found bool) {
	index, found = findRejectParameterIndex(funcType); index--
	found = found && index >= 0 && funcType.In(index).Kind() == reflect.Func &&
		(funcType.In(index).NumIn() == 0 || !funcType.In(index).In(0).Implements(ERROR_TYPE))
	return
}

func findRejectParameterIndex(funcType reflect.Type) (int, bool) {
	for i := 0; i < funcType.NumIn(); i++ {
		if funcType.In(i) == REJECTOR_TYPE {
			return i, true
		}
	}
	return -1, false
}

func extractError(results []reflect.Value) ([]reflect.Value, error) {
	resultLen := len(results)
	if resultLen > 0 {
		err, _ := results[resultLen - 1].Interface().(error)
		return results[:resultLen - 1], err
	}
	return results, nil
}

func insertIntoSlice(slice interface{}, value interface{}, index int) (interface{}) {
	sliceValue := reflect.ValueOf(slice)
	sliceType := reflect.SliceOf(reflect.TypeOf(slice).Elem())
	newSlice := reflect.MakeSlice(sliceType, 0, 0)
	for i := 0; i <= sliceValue.Len(); i++ {
		if i == index { newSlice = reflect.Append(newSlice, reflect.ValueOf(value)) }
		if i != sliceValue.Len() { newSlice = reflect.Append(newSlice, sliceValue.Index(i)) }
	}
	return newSlice.Interface()
}

func extractParameterTypes(paramType func(int) (reflect.Type), count int) (parameterTypes []reflect.Type) {
	for i := 0; i < count; i++ {
		parameterTypes = append(parameterTypes, paramType(i))
	}
	return
}