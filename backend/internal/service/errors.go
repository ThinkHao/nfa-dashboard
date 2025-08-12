package service

import "fmt"

// BadRequestError represents a client error that should be returned as HTTP 400
type BadRequestError struct{ Msg string }

func (e BadRequestError) Error() string { return e.Msg }

func NewBadRequest(msg string) error { return BadRequestError{Msg: msg} }

func NewBadRequestf(format string, a ...any) error { return BadRequestError{Msg: fmt.Sprintf(format, a...)} }

func IsBadRequest(err error) bool {
    if err == nil { return false }
    _, ok := err.(BadRequestError)
    return ok
}
