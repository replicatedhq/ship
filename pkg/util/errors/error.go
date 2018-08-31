package errors

type FetchFilesError struct {
	Message string
}

func (f FetchFilesError) Error() string {
	return f.Message
}
