package chainrpc

import "fmt"

type RateLimitError struct {
	Method string
	URL    string
	Err    error
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded for %s request to %s: %v", e.Method, e.URL, e.Err)
}
