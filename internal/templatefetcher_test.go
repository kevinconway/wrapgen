package wrapgen

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	http "net/http"
	"strings"
	"testing"

	gomock "github.com/golang/mock/gomock"
)

func TestHTTPTemplateFetcher(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	rt := NewMockRoundTripper(ctrl)
	ctx := context.Background()
	fetcher := &HTTPTemplateFetcher{
		Client: &http.Client{Transport: rt},
	}

	if _, err := fetcher.FetchTemplate(ctx, "noproto"); err == nil {
		t.Fatal("http tried to use a path without a protocol")
	}

	rt.EXPECT().RoundTrip(gomock.Any()).Return(nil, errors.New("fail"))
	if _, err := fetcher.FetchTemplate(ctx, "https://localhost"); err == nil {
		t.Fatal("http masked a transport error")
	}

	rt.EXPECT().RoundTrip(gomock.Any()).Return(&http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       ioutil.NopCloser(bytes.NewBufferString("internal server error")),
	}, nil)
	if _, err := fetcher.FetchTemplate(ctx, "https://localhost"); err == nil {
		t.Fatal("http masked a status code error")
	}

	rt.EXPECT().RoundTrip(gomock.Any()).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewBufferString("i am template")),
	}, nil)
	tmpl, err := fetcher.FetchTemplate(ctx, "https://localhost")
	if err != nil {
		t.Fatalf("http did not fetch: %v", err)
	}
	if tmpl != "i am template" {
		t.Fatalf("http fetched wrong template: %s", tmpl)
	}
}

func TestFSTemplateFetcher(t *testing.T) {
	fetcher := &FSTemplateFetcher{ReadFn: func(string) ([]byte, error) {
		return nil, errors.New("failure")
	}}
	if _, err := fetcher.FetchTemplate(context.Background(), "template.txt"); err == nil {
		t.Fatal("fs masked a disk error")
	}

	fetcher = &FSTemplateFetcher{ReadFn: func(string) ([]byte, error) {
		return []byte(`i am template`), nil
	}}
	tmpl, err := fetcher.FetchTemplate(context.Background(), "template.txt")
	if err != nil {
		t.Fatalf("fs did not fetch: %v", err)
	}
	if tmpl != "i am template" {
		t.Fatalf("fs fetched wrong template: %s", tmpl)
	}
}

func TestMultiTemplateFetcher(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ferr := errors.New("failure")
	f1 := NewMockTemplateFetcher(ctrl)
	f2 := NewMockTemplateFetcher(ctrl)
	f3 := NewMockTemplateFetcher(ctrl)
	fetcher := MultiTemplateFetcher{f1, f2, f3}

	f1.EXPECT().FetchTemplate(gomock.Any(), gomock.Any()).Return("", ferr)
	f2.EXPECT().FetchTemplate(gomock.Any(), gomock.Any()).Return("", ferr)
	f3.EXPECT().FetchTemplate(gomock.Any(), gomock.Any()).Return("", ferr)
	if _, err := fetcher.FetchTemplate(context.Background(), "template.txt"); err == nil {
		t.Fatal("multi fetcher masked underlying errors")
	}

	f1.EXPECT().FetchTemplate(gomock.Any(), gomock.Any()).Return("", ferr)
	f2.EXPECT().FetchTemplate(gomock.Any(), gomock.Any()).Return("", ferr)
	f3.EXPECT().FetchTemplate(gomock.Any(), gomock.Any()).Return("i am template", nil)
	tmpl, err := fetcher.FetchTemplate(context.Background(), "template.txt")
	if err != nil {
		t.Fatalf("multi fetcher did not succeed: %v", err)
	}
	if tmpl != "i am template" {
		t.Fatalf("multi fetcher fetched wrong template: %s", tmpl)
	}

	if _, err := (MultiTemplateFetcher{}).FetchTemplate(context.Background(), "template.txt"); err == nil {
		t.Fatal("multi fetcher tried to work without any fetcher installed")
	}
}

func TestMultiError(t *testing.T) {
	var err multiError
	_ = err.Error() // ensure no panic when empty

	err = append(err, errors.New("one"), errors.New("two"))
	s := err.Error()
	if !strings.Contains(s, "one") || !strings.Contains(s, "two") {
		t.Fatalf("multiError did not output all contained errors: %s", s)
	}
}
