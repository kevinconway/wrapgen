package wrapgen

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// HTTPTemplateFetcher treats any given path as a URL and attempts
// to download the content via a GET.
type HTTPTemplateFetcher struct {
	Client *http.Client
}

// FetchTemplate attempts to load via GET.
func (f *HTTPTemplateFetcher) FetchTemplate(ctx context.Context, path string) (string, error) {
	if !strings.Contains(path, "://") {
		return "", fmt.Errorf("path %s contains no protocol such as http:// or https://", path)
	}
	req, _ := http.NewRequest(http.MethodGet, path, http.NoBody)
	resp, err := f.Client.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode > 300 || resp.StatusCode < 200 {
		return "", fmt.Errorf("%d fetching template: %v", resp.StatusCode, errors.New(string(b)))
	}
	return string(b), nil
}

// FSTemplateFetcher treats any given path as existing on the file system
// and attempts to open the file.
type FSTemplateFetcher struct {
	ReadFn func(string) ([]byte, error)
}

// FetchTemplate attempts to load from the file system.
func (f *FSTemplateFetcher) FetchTemplate(ctx context.Context, path string) (string, error) {
	b, err := f.ReadFn(path)
	return string(b), err
}

type multiError []error

func (e multiError) Error() string {
	var buf bytes.Buffer
	for _, err := range e {
		_, _ = buf.WriteString(err.Error())
		_, _ = buf.WriteString(" | ")
	}
	return buf.String()
}

// MultiTemplateFetcher takes an ordered set of TemplateFetcher instances an
// attempts to call each one until a response if received or all fail.
type MultiTemplateFetcher []TemplateFetcher

func (f MultiTemplateFetcher) FetchTemplate(ctx context.Context, path string) (string, error) {
	var errs []error
	for _, fetcher := range f {
		r, err := fetcher.FetchTemplate(ctx, path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		return r, nil
	}
	if len(errs) == 0 {
		return "", fmt.Errorf("no fetchers found for path %s", path)
	}
	return "", multiError(errs)
}
