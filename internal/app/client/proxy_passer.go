package client

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/dewadg/freedom/internal/config"
)

type proxyPasser struct {
	cfg                *config.Config
	client             *http.Client
	allowedHTTPHeaders map[string]bool
}

func newProxyPasser(cfg *config.Config) *proxyPasser {
	allowedHTTPHeaders := make(map[string]bool, len(cfg.ProxyPass.AllowedHTTPHeaders))
	for _, key := range cfg.ProxyPass.AllowedHTTPHeaders {
		allowedHTTPHeaders[key] = true
	}

	return &proxyPasser{
		cfg: cfg,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		allowedHTTPHeaders: allowedHTTPHeaders,
	}
}

type proxyPassResponse struct {
	statusCode int
	body       []byte
	headers    http.Header
}

func (p *proxyPasser) Call(ctx context.Context, r *http.Request) (proxyPassResponse, error) {
	var reqBody *bytes.Buffer
	if r.Body != nil {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return proxyPassResponse{}, err
		}

		reqBody = bytes.NewBuffer(body)
	}

	req, err := http.NewRequestWithContext(ctx, r.Method, p.cfg.ProxyPass.Target+r.URL.Path, reqBody)
	if err != nil {
		return proxyPassResponse{}, err
	}
	req = p.populateRequestHeaders(r, req)

	resp, err := p.client.Do(req)
	if err != nil {
		return proxyPassResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return proxyPassResponse{}, err
	}

	headers := resp.Header.Clone()

	return proxyPassResponse{
		statusCode: resp.StatusCode,
		body:       body,
		headers:    headers,
	}, nil
}

func (p *proxyPasser) populateRequestHeaders(sourceReq, destinationReq *http.Request) *http.Request {
	for key, values := range sourceReq.Header {
		key = strings.ToLower(key)
		if _, ok := p.allowedHTTPHeaders[key]; !ok {
			continue
		}

		for _, value := range values {
			destinationReq.Header.Set(key, value)
		}
	}

	return destinationReq
}
