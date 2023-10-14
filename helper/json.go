package helper

import "encoding/json"

// UnmarshalJSON create a new object T, unmarshal the data into the object.
func UnmarshalJSON[T any](data []byte) (T, error) {
	var v T

	if err := json.Unmarshal(data, &v); err != nil {
		return v, err
	}

	return v, nil
}
