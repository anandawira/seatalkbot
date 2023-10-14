package helper

import (
	"io"
)

// ParseBodyJSON reads the body and then Unmarshal it into T.
func ParseBodyJSON[T any](body io.Reader) (responseBody T, err error) {
	rawBody, err := io.ReadAll(body)
	if err != nil {
		return *new(T), err
	}

	responseBody, err = UnmarshalJSON[T](rawBody)
	if err != nil {
		return *new(T), err
	}

	return responseBody, nil
}
