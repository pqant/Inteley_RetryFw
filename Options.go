package Inteley_RetryFw

import (
	"time"
)

const (
	DefaultMaxTries = 3
	DefaultTimeout  = time.Duration(15 * time.Second)
)

func Not(checker func(err error) bool) func(err error) bool {
	return func(err error) bool {
		return !checker(err)
	}
}

type RetryOption func(options *retryOptions)

func Timeout(d time.Duration) RetryOption {
	return func(options *retryOptions) {
		options.Timeout = d
	}
}

func MaxTries(tries int) RetryOption {
	return func(options *retryOptions) {
		options.MaxTries = tries
	}
}

func RetryChecker(checker func(err error) bool) RetryOption {
	return func(options *retryOptions) {
		options.Checker = checker
	}
}

func Sleep(d time.Duration) RetryOption {
	return func(options *retryOptions) {
		options.Sleep = d
	}
}

func AfterRetry(afterRetry func(err error)) RetryOption {
	return func(options *retryOptions) {
		options.AfterRetry = afterRetry
	}
}

func AfterRetryLimit(afterRetryLimit func(err error)) RetryOption {
	return func(options *retryOptions) {
		options.AfterRetryLimit = afterRetryLimit
	}
}

type retryOptions struct {
	Timeout         time.Duration
	MaxTries        int
	Checker         func(err error) bool
	Sleep           time.Duration
	AfterRetry      func(err error)
	AfterRetryLimit func(err error)
}

func newRetryOptions(options ...RetryOption) retryOptions {
	state := retryOptions{
		Timeout:         DefaultTimeout,
		MaxTries:        DefaultMaxTries,
		Checker:         Any,
		AfterRetry:      func(err error) {},
		AfterRetryLimit: func(err error) {},
	}

	for _, option := range options {
		option(&state)
	}

	return state
}
