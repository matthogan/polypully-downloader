package errors

type ValidationError struct {
	Msg string
	Err error
}

func (e *ValidationError) Error() string {
	return e.Msg
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}
