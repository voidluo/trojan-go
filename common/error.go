package common

import (
	"fmt"
)

type Error struct {
	info string
	err  error
}

func (e *Error) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s: %v", e.info, e.err)
	}
	return e.info
}

func (e *Error) Unwrap() error {
	return e.err
}

func (e *Error) Base(err error) *Error {
	e.err = err
	return e
}

func NewError(info string) *Error {
	return &Error{
		info: info,
	}
}

func Must(err error) {
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}

func Must2(_ any, err error) {
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}
