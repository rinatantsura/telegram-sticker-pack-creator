package errors

import "fmt"

type Error struct {
	CustomerMessage string
	ErrInternal     error
}

func (e Error) Error() string {
	return fmt.Sprintf("Interanl Error: %s, Customer Message: %s", e.ErrInternal, e.CustomerMessage)
}

func (e Error) Wrap(err error) error {
	e.ErrInternal = err
	return e
}
