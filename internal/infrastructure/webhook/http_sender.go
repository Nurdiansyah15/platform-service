// Package webhook berisi pengirim HTTP untuk webhook.
package webhook

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"platform-service/internal/app/port"
)

// HTTPSender implements port.WebhookSender.
type HTTPSender struct {
	client *http.Client
}

func NewHTTPSender(timeout time.Duration) *HTTPSender {
	return &HTTPSender{client: &http.Client{
		Timeout: timeout,
		// Jangan ikuti redirect: penerima wajib membalas di URL yang terdaftar.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

func (s *HTTPSender) Send(ctx context.Context, req port.WebhookRequest) (int, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, req.URL, bytes.NewReader(req.Body))
	if err != nil {
		return 0, fmt.Errorf("webhook: buat request: %w", err)
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("webhook: kirim: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	return resp.StatusCode, nil
}
