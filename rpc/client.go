package chainrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

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

func (r *Request) SetHeader(k, v string) {
	if r.headers == nil {
		r.headers = make(map[string]string)
	}
	r.headerLock.Lock()
	defer r.headerLock.Unlock()
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

	res, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	// Accept 200 OK and 202 Accepted (Aptos submit tx), plus a couple common "success" codes.
	switch res.StatusCode {
	case http.StatusOK, http.StatusAccepted, http.StatusCreated, http.StatusNoContent:
		// ok
	default:
		preview := string(b)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return fmt.Errorf("unexpected status=%d, url=%s, preview=%q",
			res.StatusCode, url, preview)
	}

	// Some success codes may have empty body (e.g., 204).
	if result == nil || len(bytes.TrimSpace(b)) == 0 {
		return nil
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

	res, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	_, err = io.Copy(result, res.Body)
	return err
}
