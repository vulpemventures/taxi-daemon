package dbbadger

// ErrIsNotDirectory is when we try the datadir badger path
// if the path references a file and not a directory, this error is returned
type ErrIsNotDirectory struct {
	Path string
}

var _ error = (*ErrIsNotDirectory)(nil)

func (e ErrIsNotDirectory) Error() string {
	return e.Path + " is not a directory"
}
