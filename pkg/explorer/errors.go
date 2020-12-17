package explorer

import "fmt"

// ErrStatusIsNotOk is returned if the request status is not 200
type ErrStatusIsNotOk struct {
	httpCode int
}

var _ error = (*ErrStatusIsNotOk)(nil)

func (e ErrStatusIsNotOk) Error() string {
	return "http status code != OK: " + fmt.Sprint(e.httpCode)
}
