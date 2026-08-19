// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/cinar/resile"
)

// HTTPError represents a non-2xx HTTP response.
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP request failed with status %d: %s", e.StatusCode, e.Body)
}

// shouldRetry restricts retries to transient server errors and transport errors.
// It explicitly aborts on 400 Bad Request, 401 Unauthorized, and other client errors.
func shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		switch httpErr.StatusCode {
		case http.StatusInternalServerError, http.StatusBadGateway, http.StatusGatewayTimeout:
			return true
		default:
			return false
		}
	}

	// Network-level failures (reset connections, DNS, TLS) are safe to retry
	// because the server never processed the mutation.
	return true
}

func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}

	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func main() {
	var mu sync.Mutex
	attempts := make(map[string]int)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		key := r.Header.Get("Idempotency-Key")

		// Terminal business-logic error: invalid payload. Never retried.
		if strings.Contains(string(body), "terminal") {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "invalid order: quantity must be positive")
			return
		}

		mu.Lock()
		attempts[key]++
		n := attempts[key]
		mu.Unlock()

		switch n {
		case 1:
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "order service temporarily unavailable")
		case 2:
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprintln(w, "upstream gateway unavailable")
		case 3:
			w.WriteHeader(http.StatusGatewayTimeout)
			fmt.Fprintln(w, "upstream gateway timed out")
		default:
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"status":"accepted","idempotency_key":"%s"}`, key)
		}
	}))
	defer server.Close()

	client := &http.Client{Timeout: 2 * time.Second}
	ctx := context.Background()

	fmt.Println("--- Safe Mutation with Idempotent POST Requests ---")
	fmt.Println("Transient 5xx responses are retried with the SAME Idempotency-Key.")
	fmt.Println("Terminal 400 responses are pushed back to the caller and abort.")

	// Transient scenario: 500, 502, 504, then success.
	if err := executeSafePOST(ctx, client, server.URL, `{"order":"transient"}`, 4); err != nil {
		fmt.Printf("Unexpected failure in transient scenario: %v\n", err)
		return
	}

	// Terminal scenario: 400 Bad Request, no retry.
	if err := executeSafePOST(ctx, client, server.URL, `{"order":"terminal"}`, 4); err != nil {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusBadRequest {
			fmt.Println("\nTerminal 400 Bad Request was not retried (as expected).")
			return
		}
		fmt.Printf("Unexpected failure in terminal scenario: %v\n", err)
		return
	}

	fmt.Println("\nUnexpected: terminal scenario succeeded instead of returning 400.")
}

func executeSafePOST(ctx context.Context, client *http.Client, url string, payload string, maxAttempts int) error {
	// Generate the idempotency key once, before the resile.Do retry loop begins.
	key, err := newUUID()
	if err != nil {
		return err
	}

	fmt.Printf("\nNew mutation request key: %s\n", key)
	attempt := 0

	_, err = resile.Do(ctx, func(ctx context.Context) (string, error) {
		attempt++
		fmt.Printf("  [Attempt %d] POST /orders with Idempotency-Key: %s\n", attempt, key)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(payload))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", key)

		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", &HTTPError{StatusCode: resp.StatusCode, Body: string(respBody)}
		}

		fmt.Printf("  [Attempt %d] Mutation accepted: %s\n", attempt, string(respBody))
		return string(respBody), nil
	},
		resile.WithName("safe-mutation-post"),
		resile.WithMaxAttempts(maxAttempts),
		resile.WithBaseDelay(50*time.Millisecond),
		resile.WithRetryIfFunc(shouldRetry),
	)

	return err
}
