package database

var ErrNotFound = NewError("not found", nil)

type DBError struct {
	msg  string
	prev error
}

func NewError(msg string, prev error) DBError {
	return DBError{msg: msg, prev: prev}
}

func (e DBError) Error() string {
	return e.msg
}

func (e DBError) Unwrap() error {
	return e.prev
}
