package promise

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
	"time"
)

func TestNewPromise(t *testing.T) {
	// Prepare
	done := make(chan struct{})
	// Test
	NewPromise(func() {
		close(done)
	})
	// Verify
	select {
	case <-done:
		// All done!
	case <-time.After(500 * time.Millisecond):
		t.Error()
	}
}

func TestNewPromiseThen(t *testing.T) {
	// Prepare
	done := make(chan struct{})
	// Test
	NewPromise(func() {
	}).Then(func() {
		close(done)
	})
	// Verify
	for {
		select {
		case _, open := <-done:
			if !open { return }
			// All done!
		case <-time.After(500 * time.Millisecond):
			t.Error()
			break
		}
	}
}

func TestNewPromiseResolveThen(t *testing.T) {
	// Prepare
	done := make(chan interface{})
	// Test
	NewPromise(func() (Promise) {
		return Resolve("resolved")
	}).Then(func(str string) {
		done <- str
	})
	// Verify
	for {
		select {
		case res := <- done:
			assert.Equal(t, "resolved", res)
			return
		case <-time.After(500 * time.Millisecond):
			t.Error()
			return
		}
	}
}

func TestNewPromiseThenString(t *testing.T) {
	// Prepare
	done := make(chan string)
	// Test
	NewPromise(func() (string, error) {
		return "resolved", nil
	}).Then(func(str string) {
		done <- str
	})
	// Verify
	for {
		select {
		case res := <-done:
			assert.Equal(t, "resolved", res)
			return
		case <-time.After(500 * time.Millisecond):
			t.Error()
			return
		}
	}
}

func TestNewPromiseRejectPromiseThenString(t *testing.T) {
	// Prepare
	done := make(chan interface{})
	// Test
	NewPromise(func() (Promise) {
		return Reject(errors.New("tadaa"))
	}).Then(func(str string) {
		done <- str
	}).Then(func(str string) {
		done <- str
	}, func(err error) {
		done <- err
	})
	// Verify
	for {
		select {
		case val := <- done:
			if !reflect.TypeOf(val).Implements(ERROR_TYPE) {
				t.Error()
			}
			return
		case <-time.After(500 * time.Millisecond):
			t.Error()
			return
		}
	}
}

func TestNewPromiseRejectPromiseThenStringFinally(t *testing.T) {
	// Prepare
	var doneInOrder []string
	done := make(chan interface{})
	// Test
	NewPromise(func() (Promise) {
		return Reject(errors.New("tadaa"))
	}).Then(func(str string) {
		done <- str
	}).Then(func(str string) {
		done <- str
	}, func(err error) {
		doneInOrder = append(doneInOrder,"thenHandleError")
	}).Finally(func() {
		doneInOrder = append(doneInOrder, "handleFinally")
		done <- nil
	})
	// Verify
	for {
		select {
		case res := <- done:
			if !assert.Nil(t, res) || !reflect.DeepEqual(doneInOrder, []string{"thenHandleError", "handleFinally"}) {
				t.Error()
			}
			return
		case <-time.After(500 * time.Millisecond):
			t.Error()
			return
		}
	}
}

func TestNewPromiseResolvePromiseThenString(t *testing.T) {
	// Prepare
	done := make(chan error)
	// Test
	NewPromise(func() (error) {
		return errors.New("rejected")
	}).Catch(func(err error) {
		done <- err
	})
	// Verify
	for {
		select {
		case err := <- done:
			assert.Error(t, err)
			return
		case <-time.After(500 * time.Millisecond):
			t.Error()
			return
		}
	}
}

func TestNewPromiseRejectCatchThenString(t *testing.T) {
	// Prepare
	done := make(chan string)
	// Test
	Reject(errors.New("resolved")).
		Catch(func(err error) (string, error) {
			return err.Error(), nil
		}).
		Finally(func() (string, error) {
			return "asd", nil
		}).
		Then(func(str string) {
			done <- str
		})
	// Verify
	for {
		select {
		case str := <- done:
			assert.Equal(t, "resolved", str)
			return
		case <-time.After(500 * time.Millisecond):
			t.Error()
			return
		}
	}
}