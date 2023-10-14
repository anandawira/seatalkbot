package helper

import "time"

// RunWithRetry runs the fn until it returns err nil or reaches the maxRetry.
// If maxRetry is set to 0 or lower, it will keep retrying until success.
func RunWithRetry(fn func() error, maxRetry int, interval time.Duration) error {
	var i int

	for {
		i++

		err := fn()
		if err == nil { // success
			return nil
		}

		if maxRetry > 0 && i >= maxRetry {
			return err
		}

		time.Sleep(interval) // wait before retrying
	}
}
