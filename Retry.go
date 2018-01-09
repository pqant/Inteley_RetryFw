package Inteley_RetryFw

import (
	"time"
	"fmt"
	"log"
)

func Retry(attempts int, sleep time.Duration, callback func() error) (err error) {
	for i := 0; ; i++ {
		err = callback()
		if err == nil {
			return
		}
		if i >= (attempts - 1) {
			break
		}
		time.Sleep(sleep)
		log.Println(">>> Retrying after error:", err)
	}
	return fmt.Errorf(">>> After %d attempt(s), last error: %s", attempts, err)
}

type ResultSetForHttpOperation struct {
	Result    chan bool
	Operation func()
}

func RetryDuring(duration time.Duration, sleep time.Duration, callback func() error) (err error) {
	t0 := time.Now()
	i := 0
	for {
		i++
		err = callback()
		if err == nil {
			return
		}
		delta := time.Now().Sub(t0)
		if delta > duration {
			return fmt.Errorf(">>> After %d attempts (during %s), last error: %s", i, delta, err)
		}
		time.Sleep(sleep)
		log.Println(">>> Retrying after error:", err)
	}
}

func Do(op func() error, retryOptions ...RetryOption) error {
	options := newRetryOptions(retryOptions...)

	var timeout <-chan time.Time
	if options.Timeout > 0 {
		timeout = time.After(options.Timeout)
	}

	tryCounter := 0
	for {
		// Check if we reached the timeout
		select {
		case <-timeout:
			return Mask(TimeoutError, Any)
		default:
		}

		// Execute the op
		tryCounter++
		lastError := op()
		options.AfterRetry(lastError)

		if lastError != nil {
			if options.Checker != nil && options.Checker(lastError) {
				// Check max retries
				if tryCounter >= options.MaxTries {
					options.AfterRetryLimit(lastError)
					return WithCausef(lastError, MaxRetriesReachedError, "retry limit reached (%d/%d)", tryCounter, options.MaxTries)
				}

				if options.Sleep > 0 {
					time.Sleep(options.Sleep)
				}
				continue
			}

			return Mask(lastError, Any)
		}
		return nil
	}
}
