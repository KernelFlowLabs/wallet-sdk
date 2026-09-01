package httpc

import (
	"fmt"
	"math"
	"net/http"
	"time"
)

type Option func(*requestOptions) error

type requestOptions struct {
	timeout          time.Duration
	timeoutSet       bool
	maxResponseBytes int64
	rateLimitEnabled bool
	rateLimitQPS     float64
	rateLimitBurst   int
	httpClient       *http.Client
}

func WithTimeout(timeout time.Duration) Option {
	return func(options *requestOptions) error {
		if timeout <= 0 {
			return fmt.Errorf("timeout must be positive")
		}
		options.timeout = timeout
		options.timeoutSet = true
		return nil
	}
}

func WithMaxResponseBytes(maxBytes int64) Option {
	return func(options *requestOptions) error {
		if maxBytes <= 0 {
			return fmt.Errorf("max response bytes must be positive")
		}
		options.maxResponseBytes = maxBytes
		return nil
	}
}

func WithRateLimit(qps float64, burst int) Option {
	return func(options *requestOptions) error {
		if qps <= 0 || math.IsNaN(qps) || math.IsInf(qps, 0) {
			return fmt.Errorf("rate limit qps must be positive and finite")
		}
		if burst <= 0 {
			return fmt.Errorf("rate limit burst must be positive")
		}
		options.rateLimitQPS = qps
		options.rateLimitBurst = burst
		options.rateLimitEnabled = true
		return nil
	}
}

func WithoutRateLimit() Option {
	return func(options *requestOptions) error {
		options.rateLimitEnabled = false
		return nil
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(options *requestOptions) error {
		if client == nil {
			return fmt.Errorf("http client must not be nil")
		}
		options.httpClient = client
		return nil
	}
}
