package promise

import "github.com/thoas/go-funk"

func Resolve(values ...interface{}) Promise {
	return NewPromise(func(resolve func(...interface{}), reject func(error)) {
		resolve(values...)
	})
}

func Reject(err error) Promise {
	return NewPromise(func(resolve func(...interface{}), reject func(error)) {
		reject(err)
	})
}

func Map(values interface{}, mapper interface{}) Promise {
	promises := funk.Map(values, mapper).([]Promise)
	return All(promises...)
}

func Race(promises ...Promise) Promise {
	return NewPromise(func(resolve func(Promise), reject func(error)) {
		racing := true
		for _, promise := range promises {
			pending := true
			promise.Then(func(results ...interface{}) {
				if racing && pending {
					racing, pending = false, false
					resolve(promise)
				}
			}, func(err error) {
				if racing && pending {
					racing, pending = false, false
					reject(err)
				}
			})
		}
	})
}

func Any(promises ...Promise) Promise {
	return NewPromise(func(resolve func(...[]interface{}), reject func(...error)) {
		failCountDown := len(promises)
		orderedFailures := make([]error, failCountDown)
		await := make(chan interface{}, failCountDown)

		if len(promises) == 0 { // TODO: Verify behavior
			resolve(nil)
			return
		}

		for index, promise := range promises {
			promise.Then(func(results ...interface{}) {
				if isChanClosed(await) { return }
				close(await)
				resolve(results)
			}, func(err error) {
				if isChanClosed(await) { return }
				await <- func() (int, error) { return index, err}
			})
		}

		for {
			select {
			case funcError, open := <-await:
				if !open { break }
				index, err := funcError.(func() (int, error))()
				orderedFailures[index] = err
				failCountDown--
			}

			if failCountDown == 0 {
				reject(orderedFailures...)
				break
			}
		}
	})
}

func All(promises ...Promise) (result Promise) {
	return NewPromise(func(resolve func(...interface{}), reject func(...error)) {
		promisesCountDown := len(promises)
		promiseResults := make([]interface{}, promisesCountDown)
		await := make(chan interface{}, promisesCountDown)

		for index, promise := range promises {
			if promise == nil {
				if isChanClosed(await) { break }
				await <- func() (int, interface{}) { return index, nil }
				continue
			}

			promise.Then(func (result interface{}) {
				if isChanClosed(await) { return }
				await <- func() (int, interface{}) { return index, result }
			}, func(err error) {
				if isChanClosed(await) { return }
				close(await)
				reject(err)
			})
		}

		for {
			select {
			case funcResult, open := <- await:
				if !open { break }
				index, result := funcResult.(func() (int, interface{}))()
				promiseResults[index] = result
				promisesCountDown--
			}

			if promisesCountDown == 0 {
				resolve(promiseResults)
				break
			}
		}
	})
}