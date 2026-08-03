package httpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const defaultUserAgent = "wallet-sdk/1.0 (+https://github.com/kernelflowlabs/wallet-sdk)"

// HTTPStatusError preserves non-2xx response metadata so callers can make
// decisions based on the status without parsing an error string. Preview and
// URL are intended for diagnostics only and should not be shown to end users.
type HTTPStatusError struct {
	StatusCode        int
	RetryAfterSeconds int
	URL               string
	Preview           string
}

func (e *HTTPStatusError) Error() string {
	if e == nil {
		return "unexpected HTTP status"
	}
	return fmt.Sprintf(
		"unexpected status=%d, url=%s, preview=%q",
		e.StatusCode,
		e.URL,
		e.Preview,
	)
}

type Request struct {
	baseUrl    string
	headers    map[string]string
	headerLock sync.RWMutex
	httpClient *http.Client
	limiter    *rate.Limiter
}

func NewRequest(baseUrl string, headers map[string]string) *Request {
	return &Request{
		baseUrl: baseUrl,
		headers: headers,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				Proxy:               http.ProxyFromEnvironment,
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     60 * time.Second,
				DisableKeepAlives:   false,
			},
		},
		limiter: rate.NewLimiter(rate.Limit(200), 500),
	}
}

func (r *Request) SetBaseUrl(url string) {
	r.baseUrl = url
}

func (r *Request) SetRateLimit(qps float64, burst int) {
	if burst <= 0 {
		burst = 1
	}
	r.limiter = rate.NewLimiter(rate.Limit(qps), burst)
}

func (r *Request) SetHeader(k, v string) {
	r.headerLock.Lock()
	defer r.headerLock.Unlock()
	if r.headers == nil {
		r.headers = make(map[string]string)
	}
	r.headers[k] = v
}

func (r *Request) Get(ctx context.Context, result interface{}, path string, query url.Values) error {
	var queryStr = ""
	if query != nil {
		queryStr = query.Encode()
	}
	uri := strings.Join([]string{r.GetBase(path), queryStr}, "?")
	return r.Execute(ctx, http.MethodGet, uri, nil, result)
}

func (r *Request) GetRaw(ctx context.Context, result *bytes.Buffer, path string, query url.Values) error {
	var queryStr = ""
	if query != nil {
		queryStr = query.Encode()
	}
	uri := strings.Join([]string{r.GetBase(path), queryStr}, "?")
	return r.ExecuteRaw(ctx, http.MethodGet, uri, nil, result)
}

func (r *Request) Post(ctx context.Context, result interface{}, path string, body interface{}) error {
	buf, err := GetJsonBody(body)
	if err != nil {
		return fmt.Errorf("failed to GetBody,err=%v", err)
	}
	uri := r.GetBase(path)
	return r.Execute(ctx, http.MethodPost, uri, buf, result)
}

func (r *Request) PostWithXWWWFormUrlencoded(ctx context.Context, result interface{}, path string, body interface{}) error {
	var buf io.Reader
	if params, ok := body.(url.Values); ok {
		buf = strings.NewReader(params.Encode())
	}
	uri := r.GetBase(path)
	return r.Execute(ctx, http.MethodPost, uri, buf, result)
}

func (r *Request) PostWithOutEncoded(ctx context.Context, result interface{}, path string, body interface{}) error {
	b, ok := body.([]byte)
	if !ok {
		return fmt.Errorf("body must be []byte, got %T", body)
	}
	buf := bytes.NewBuffer(b)
	uri := r.GetBase(path)
	return r.Execute(ctx, http.MethodPost, uri, buf, result)
}

func (r *Request) PostWithPlain(ctx context.Context, result interface{}, path string, body io.Reader) error {
	uri := r.GetBase(path)
	return r.Execute(ctx, http.MethodPost, uri, body, result)
}

func (r *Request) Delete(ctx context.Context, result interface{}, path string) error {
	uri := r.GetBase(path)
	return r.Execute(ctx, http.MethodDelete, uri, nil, result)
}

func (r *Request) Patch(ctx context.Context, result interface{}, path string, body interface{}) error {
	buf, err := GetJsonBody(body)
	if err != nil {
		return fmt.Errorf("failed to GetBody,err=%v", err)
	}
	uri := r.GetBase(path)
	return r.Execute(ctx, http.MethodPatch, uri, buf, result)
}

func (r *Request) GetBase(path string) string {
	if path == "" {
		return r.baseUrl
	}
	return fmt.Sprintf("%s/%s", r.baseUrl, path)
}

func GetJsonBody(body interface{}) (buf io.ReadWriter, err error) {
	if body != nil {
		buf = new(bytes.Buffer)
		err = json.NewEncoder(buf).Encode(body)
	}
	return
}

func (r *Request) Execute(ctx context.Context, method string, url string, body io.Reader, result interface{}) error {
	if err := r.limiter.Wait(ctx); err != nil {
		return &RateLimitError{
			Method: method,
			URL:    url,
			Err:    err,
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}

	r.headerLock.RLock()
	for key, value := range r.headers {
		req.Header.Set(key, value)
	}
	r.headerLock.RUnlock()
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", defaultUserAgent)
	}

	res, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	switch res.StatusCode {
	case http.StatusOK, http.StatusAccepted, http.StatusCreated, http.StatusNoContent:
	default:
		preview := string(b)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return &HTTPStatusError{
			StatusCode:        res.StatusCode,
			RetryAfterSeconds: retryAfterSeconds(res.Header.Get("Retry-After"), time.Now()),
			URL:               url,
			Preview:           preview,
		}
	}

	if result == nil {
		return nil
	}
	if len(bytes.TrimSpace(b)) == 0 {
		if res.StatusCode == http.StatusNoContent {
			return nil
		}
		return fmt.Errorf("empty response body, status=%d, url=%s", res.StatusCode, url)
	}

	if err := json.Unmarshal(b, result); err != nil {
		preview := string(b)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return fmt.Errorf("unmarshal failed, status=%d, url=%s, preview=%q, err=%w",
			res.StatusCode, url, preview, err)
	}

	return nil
}

func retryAfterSeconds(value string, now time.Time) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds > 0 {
			return seconds
		}
		return 0
	}
	retryAt, err := http.ParseTime(value)
	if err != nil || !retryAt.After(now) {
		return 0
	}
	seconds := int(retryAt.Sub(now) / time.Second)
	if seconds < 1 {
		return 1
	}
	return seconds
}

func (r *Request) ExecuteRaw(ctx context.Context, method string, url string, body io.Reader, result *bytes.Buffer) error {
	if err := r.limiter.Wait(ctx); err != nil {
		return &RateLimitError{
			Method: method,
			URL:    url,
			Err:    err,
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}

	r.headerLock.RLock()
	for key, value := range r.headers {
		req.Header.Set(key, value)
	}
	r.headerLock.RUnlock()
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", defaultUserAgent)
	}

	res, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if _, err = io.Copy(result, res.Body); err != nil {
		return err
	}
	switch res.StatusCode {
	case http.StatusOK, http.StatusAccepted, http.StatusCreated, http.StatusNoContent:
		return nil
	default:
		preview := result.String()
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return &HTTPStatusError{
			StatusCode:        res.StatusCode,
			RetryAfterSeconds: retryAfterSeconds(res.Header.Get("Retry-After"), time.Now()),
			URL:               url,
			Preview:           preview,
		}
	}
}
