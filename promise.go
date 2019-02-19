package promise

import (
	"reflect"
	"sync"
)

type Promise interface {
	Then(resolver interface{}, rejector ...interface{}) Promise
	Tap(callback interface{}) Promise
	Spread(callback interface{}) Promise
	Catch(handler interface{}) Promise
	Finally(handler interface{}) Promise

	setState(state callbackState)
	State() callbackState
	setResults([]reflect.Value)
	results() []reflect.Value
	Results() []interface{}
	setError(error)
	Error() (err error)

	addStateCompleteListener(listener func(state callbackState)) Promise
	call(paramValues ...reflect.Value)
	process(parameters ...interface{}) Promise

	this(func (Promise)) Promise
	This(func (Promise)) Promise
}

type PromiseProto struct {
	callback *callback

	callbackError error
	callbackResults []reflect.Value
	promiseState         callbackState
	promiseStateLock     sync.Mutex
	stateChangeListeners []func(state callbackState)

	mux           sync.Mutex
}

func (this *PromiseProto) this(closure func(this Promise)) Promise {
	closure(this)
	return this
}

func (this *PromiseProto) This(closure func(this Promise)) Promise {
	return this.this(closure)
}

func NewPromise(callbackFunc interface{}) Promise {
	return newPromise(callbackFunc).process()
}

func newPromise(callbackFunc interface{}) *PromiseProto {
	if !IsFunc(callbackFunc) || !isValidCallbackSignature(callbackFunc) {
		panic("Invalid callback signature")
	}

	return new(PromiseProto).this(func(this Promise) {
		this.(*PromiseProto).callback = newCallback(callbackFunc)
	}).(*PromiseProto)
}

func (this *PromiseProto) call(paramValues ...reflect.Value) {
	this.callback.call(paramValues...)(func(err error, results ...reflect.Value) {
		if err != nil {
			this.setError(err)
			this.setState(STATE_REJECTED)
			return
		}

		if this.callback.isReturningPromise && len(results) > 0 {
			if promiseValue := results[0]; promiseValue.Type().Implements(PROMISE_TYPE) {
				promise, _ := promiseValue.Interface().(Promise)
				if promise != nil {
					promise.addStateCompleteListener(func(state callbackState) {
						switch state {
						case STATE_FULFILLED:
							this.setResults(promise.results())
							this.setState(state)
						case STATE_REJECTED:
							this.setError(promise.Error())
							this.setState(state)
						}
					})
				} else {
					// TODO : panic?
				}
			}
		} else {
			this.setResults(results)
			this.setState(STATE_FULFILLED)
		}
	})
}

func (this *PromiseProto) process(parameters ...interface{}) Promise {
	go func() {
		var paramValues []reflect.Value
		for _, parameter := range parameters {
			paramValues = append(paramValues, reflect.ValueOf(parameter))
		}

		this.call(paramValues...)
	}()
	return this
}

func (this *PromiseProto) setState(state callbackState) {
	this.promiseStateLock.Lock()
	defer this.promiseStateLock.Unlock()

	if this.State() == STATE_PENDING {
		this.promiseState = state
		this.fireStateChanged(state)
		this.stateChangeListeners = nil
	} else {
		// TODO: print or log something out
	}
}

func (this *PromiseProto) State() callbackState {
	if this.promiseState == "" {
		return STATE_PENDING
	}
	return this.promiseState
}

func (this *PromiseProto) addStateCompleteListener(listener func(state callbackState)) Promise {
	if listener == nil { panic("callback state listener cannot be <nil>") }

	this.promiseStateLock.Lock()
	defer this.promiseStateLock.Unlock()

	switch this.State() {
	case STATE_PENDING:
		this.stateChangeListeners = append(this.stateChangeListeners, listener)
	case STATE_FULFILLED, STATE_REJECTED:
		listener(this.State())
	default:
		panic("Invalid callback state: " + this.State())
	}
	return this
}

func (this *PromiseProto) Then(resolver interface{}, rejector ...interface{}) Promise {
	return new(PromiseProto).this(func(promise Promise) {
		newPromise := promise.(*PromiseProto)
		switch len(rejector) {
		case 1:
			rejector := rejector[0]
			assertFunctionSignature(rejector, _FUNC_IN_ERROR_OUT)
			assertFunctionSignature(resolver, _FUNC_IN_VARIADIC_OBJS_OUT)

			this.addStateCompleteListener(func(state callbackState) {
				switch state {
				case STATE_FULFILLED:
					newPromise.callback = newCallback(resolver)
					newPromise.call(this.results()...)
				case STATE_REJECTED:
					newPromise.callback = newCallback(rejector)
					newPromise.call(reflect.ValueOf(this.Error()))
				}
			})
		case 0:
			assertFunctionSignature(resolver,
				_FUNC_IN_OBJS_RESOLVE_OBJS_REJECT_ERROR_OUT,
				_FUNC_IN_OUT,
				_FUNC_IN_VARIADIC_OBJS_OUT,
				_FUNC_IN_VARIADIC_OBJS_OUT_ERROR,
				_FUNC_IN_VARIADIC_OBJS_OUT_OBJ_ERROR,
				_FUNC_IN_VARIADIC_OBJS_OUT_OBJS_ERROR,
				_FUNC_IN_VARIADIC_OBJS_OUT_PROMISE,
				_FUNC_IN_VARIADIC_OBJS_OUT_PROMISE_ERROR,
			)

			this.addStateCompleteListener(func(state callbackState) {
				switch state {
				case STATE_FULFILLED:
					newPromise.callback = newCallback(resolver)
					newPromise.call(this.results()...)
				case STATE_REJECTED:
					newPromise.callback = this.callback
					newPromise.process(this.Error())
				}
			})
		default:
			panic("Only one rejector can be defined")
		}
	})
}

func (this *PromiseProto) Tap(callback interface{}) Promise {
	assertFunctionSignature(callback,
		_FUNC_IN_OBJ_RESOLVE_REJECT_ERROR_OUT,
		_FUNC_IN_VARIADIC_OBJS_OUT,
		_FUNC_IN_VARIADIC_OBJS_OUT_ERROR,
	)
	return newPromise(callback).this(func(promise Promise) {
		newPromise := promise.(*PromiseProto)
		this.addStateCompleteListener(func(state callbackState) {
			switch state {
			case STATE_FULFILLED:
				newPromise.call(this.results()...)
			case STATE_REJECTED:
				newPromise.setError(this.Error())
				newPromise.setState(state)
			}
		})
		newPromise.addStateCompleteListener(func(state callbackState) {
			if newPromise.Error() == nil {
				newPromise.setError(this.Error())
				newPromise.setResults(this.results())
			}
		})
	})
}

func (this *PromiseProto) Spread(callback interface{}) Promise {
	assertFunctionSignature(callback,
		_FUNC_IN_RESOLVE_OBJS_REJECT_ERROR_VARIADIC_OBJS_OUT,
		_FUNC_IN_VARIADIC_OBJS_OUT,
		_FUNC_IN_VARIADIC_OBJS_OUT_ERROR,
		_FUNC_IN_VARIADIC_OBJS_OUT_OBJ_ERROR,
		_FUNC_IN_VARIADIC_OBJS_OUT_OBJS_ERROR,
		_FUNC_IN_VARIADIC_OBJS_OUT_PROMISE,
		_FUNC_IN_VARIADIC_OBJS_OUT_PROMISE_ERROR,
	)

	return newPromise(callback).this(func(newPromise Promise) {
		this.addStateCompleteListener(func(state callbackState) {
			switch state {
			case STATE_FULFILLED:
				newPromise.call(this.results()...)
			case STATE_REJECTED:
				newPromise.setError(this.Error())
				newPromise.setState(state)
			}
		})
	})
}



func (this *PromiseProto) setError(err error) {
	this.callbackError = err
}

func (this *PromiseProto) Error() (err error) {
	if this.callback == nil {
		return nil
	}
	return this.callbackError
}

func (this *PromiseProto) setResults(results []reflect.Value) {
	this.callbackResults = results
}

func (this *PromiseProto) results() (results []reflect.Value) {
	if this.callback != nil && this.callbackResults != nil {
		results = this.callbackResults
	}
	return
}

func (this *PromiseProto) Results() (results []interface{}) {
	if this.callback != nil && this.results() != nil {
		for _, result := range this.results() {
			results = append(results, result.Interface())
		}
	}
	return
}

func (this *PromiseProto) Catch(handler interface{}) Promise {
	var errorType reflect.Type

	assertFunctionSignature(handler,
		_FUNC_IN_ERROR_RESOLVE_OBJS_REJECT_ERROR_OUT,	// func(error, resolve func(...interface{}), reject func(error))
		_FUNC_IN_ERROR_OUT, 							// func(error)
		_FUNC_IN_ERROR_OUT_ERROR, 						// func(error) (error)
		_FUNC_IN_ERROR_OUT_OBJ_ERROR, 					// func(error) (interface{}, error)
		_FUNC_IN_ERROR_OUT_OBJS_ERROR, 					// func(error) ([]interface{}, error)
		_FUNC_IN_ERROR_OUT_PROMISE, 					// func(error) (*Promise)
		_FUNC_IN_ERROR_OUT_PROMISE_ERROR, 				// func(error) (*Promise, error)
	)

	handlerType := reflect.TypeOf(handler)
	errorType = handlerType.In(0)

	return newPromise(handler).this(func(newPromise Promise) {
		this.addStateCompleteListener(func(state callbackState) {
			switch state {
			case STATE_REJECTED:
				if typedError, isTyped := this.Error().(typedError);
					(isTyped && typedError.IsTypeOf(errorType)) ||
						func(typ reflect.Type) bool {
							return typ == errorType || typ.Implements(ERROR_TYPE)
						}(reflect.TypeOf(this.Error())) {
					newPromise.process(this.Error())
				} else {
					newPromise.setError(this.Error())
					newPromise.setState(state)
				}
			case STATE_FULFILLED:
				newPromise.setResults(this.results())
				newPromise.setState(state)
			}
		})
	})
}

func (this *PromiseProto) Finally(handler interface{}) Promise {
	assertFunctionSignature(handler,
		_FUNC_IN_OUT,
		_FUNC_IN_OUT_ERROR,
		_FUNC_IN_OUT_OBJ_ERROR,
		_FUNC_IN_OUT_OBJS_ERROR,
		_FUNC_IN_OUT_PROMISE,
		_FUNC_IN_OUT_PROMISE_ERROR,
	)

	return newPromise(handler).this(func(newPromise Promise) {
		this.addStateCompleteListener(func(state callbackState) {
			switch state {
			case STATE_REJECTED, STATE_FULFILLED:
				newPromise.addStateCompleteListener(func(state callbackState) {
					if state != STATE_REJECTED {
						newPromise.setError(this.Error())
						newPromise.setResults(this.results())
					}
				}).process()
			}
		})
	})
}


func (this *PromiseProto) fireStateChanged(state callbackState) {
	stateChangeListeners := this.stateChangeListeners
	for _, stateChangeListener := range stateChangeListeners {
		stateChangeListener(state)
	}
}

func IsFunc(target interface{}) (isFunc bool) {
	defer func() { if r := recover(); r != nil { isFunc = false } }()
	return reflect.TypeOf(target).Kind() == reflect.Func
}

func isChanOpen(ch chan interface{}) bool {
	select {
	case result, open := <- ch:
		if !open { return false }
		ch <- result
	default:
	}
	return true
}

func isChanClosed(ch chan interface{}) bool {
	return !isChanOpen(ch)
}

