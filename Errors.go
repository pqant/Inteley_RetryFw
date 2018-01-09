package Inteley_RetryFw

import (
	"fmt"
	"runtime"
	"bytes"
	"log"
)

const debug = false

var (
	TimeoutError           = New("Operation aborted. Timeout occured")
	MaxRetriesReachedError = New("Operation aborted. Too many errors.")
)

func IsTimeout(err error) bool {
	return Cause(err) == TimeoutError
}

func IsMaxRetriesReached(err error) bool {
	return Cause(err) == MaxRetriesReachedError
}

type Location struct {
	File string
	Line int
}

func (loc Location) String() string {
	return fmt.Sprintf("%s:%d", loc.File, loc.Line)
}

func (loc Location) IsSet() bool {
	return loc.File != ""
}

type Err struct {
	Message_ string

	Cause_ error

	Underlying_ error

	Location_ Location
}

func (e *Err) Location() Location {
	return e.Location_
}

func (e *Err) Underlying() error {
	return e.Underlying_
}

func (e *Err) Cause() error {
	return e.Cause_
}

func (e *Err) Message() string {
	return e.Message_
}

func (e *Err) Error() string {
	switch {
	case e.Message_ == "" && e.Underlying == nil:
		return "<no error>"
	case e.Message_ == "":
		return e.Underlying_.Error()
	case e.Underlying_ == nil:
		return e.Message_
	}
	return fmt.Sprintf("%s: %v", e.Message_, e.Underlying_)
}

func (e *Err) GoString() string {
	return Details(e)
}

type Causer interface {
	Cause() error
}

type Wrapper interface {
	Message() string
	Underlying() error
}

type Locationer interface {
	Location() Location
}

func Details(err error) string {
	if err == nil {
		return "[]"
	}
	var s []byte
	s = append(s, '[')
	for {
		s = append(s, '{')
		if err, ok := err.(Locationer); ok {
			loc := err.Location()
			if loc.IsSet() {
				s = append(s, loc.String()...)
				s = append(s, ": "...)
			}
		}
		if cerr, ok := err.(Wrapper); ok {
			s = append(s, cerr.Message()...)
			err = cerr.Underlying()
		} else {
			s = append(s, err.Error()...)
			err = nil
		}
		if debug {
			if err, ok := err.(Causer); ok {
				if cause := err.Cause(); cause != nil {
					s = append(s, fmt.Sprintf("=%T", cause)...)
					s = append(s, Details(cause)...)
				}
			}
		}
		s = append(s, '}')
		if err == nil {
			break
		}
		s = append(s, ' ')
	}
	s = append(s, ']')
	return string(s)
}

func (e *Err) SetLocation(callDepth int) {
	_, file, line, _ := runtime.Caller(callDepth + 1)
	e.Location_ = Location{file, line}
}

func setLocation(err error, callDepth int) {
	if e, _ := err.(*Err); e != nil {
		e.SetLocation(callDepth + 1)
	}
}

func New(s string) error {
	err := &Err{Message_: s}
	err.SetLocation(1)
	return err
}

func Newf(f string, a ...interface{}) error {
	err := &Err{Message_: fmt.Sprintf(f, a...)}
	err.SetLocation(1)
	return err
}

func match(err error, pass ...func(error) bool) bool {
	for _, f := range pass {
		if f(err) {
			return true
		}
	}
	return false
}

func Is(err error) func(error) bool {
	return func(err1 error) bool {
		return err == err1
	}
}

func Any(error) bool {
	return true
}

func NoteMask(underlying error, msg string, pass ...func(error) bool) error {
	newErr := &Err{
		Underlying_: underlying,
		Message_:    msg,
	}
	if len(pass) > 0 {
		if cause := Cause(underlying); match(cause, pass...) {
			newErr.Cause_ = cause
		}
	}
	if debug {
		if newd, oldd := newErr.Cause_, Cause(underlying); newd != oldd {
			log.Printf("Mask cause %[1]T(%[1]v)->%[2]T(%[2]v)", oldd, newd)
			log.Printf("call stack: %s", callers(0, 20))
			log.Printf("len(allow) == %d", len(pass))
			log.Printf("old error %#v", underlying)
			log.Printf("new error %#v", newErr)
		}
	}
	return newErr
}

func Mask(underlying error, pass ...func(error) bool) error {
	if underlying == nil {
		return nil
	}
	err := NoteMask(underlying, "", pass...)
	setLocation(err, 1)
	return err
}

func Notef(underlying error, f string, a ...interface{}) error {
	err := NoteMask(underlying, fmt.Sprintf(f, a...))
	setLocation(err, 1)
	return err
}

func MaskFunc(allow ...func(error) bool) func(error, ...func(error) bool) error {
	return func(err error, allow1 ...func(error) bool) error {
		var allowEither []func(error) bool
		if len(allow1) > 0 {
			// This is more efficient than using a function literal,
			// because the compiler knows that it doesn't escape.
			allowEither = make([]func(error) bool, len(allow)+len(allow1))
			copy(allowEither, allow)
			copy(allowEither[len(allow):], allow1)
		} else {
			allowEither = allow
		}
		err = Mask(err, allowEither...)
		setLocation(err, 1)
		return err
	}
}

func WithCausef(underlying, cause error, f string, a ...interface{}) error {
	err := &Err{
		Underlying_: underlying,
		Cause_:      cause,
		Message_:    fmt.Sprintf(f, a...),
	}
	err.SetLocation(1)
	return err
}

func Cause(err error) error {
	var diag error
	if err, ok := err.(Causer); ok {
		diag = err.Cause()
	}
	if diag != nil {
		return diag
	}
	return err
}

func callers(n, max int) []byte {
	var b bytes.Buffer
	prev := false
	for i := 0; i < max; i++ {
		_, file, line, ok := runtime.Caller(n + 1)
		if !ok {
			return b.Bytes()
		}
		if prev {
			fmt.Fprintf(&b, " ")
		}
		fmt.Fprintf(&b, "%s:%d", file, line)
		n++
		prev = true
	}
	return b.Bytes()
}
