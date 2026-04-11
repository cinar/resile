// Copyright (c) 2026 Onur Cinar.
// The source code is provided under MIT License.
// https://github.com/cinar/resile

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cinar/resile"
)

func main() {
	// 1. Setup Mock Server Data
	const fileSize = 50 * 1024 // 50KB
	const fileName = "downloaded.bin"
	fileContent := make([]byte, fileSize)
	for i := range fileContent {
		fileContent[i] = byte(i % 256)
	}

	// 2. Start a Mock HTTP Server that supports Range requests and simulates failures.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		start := int64(0)
		isPartial := false

		if rangeHeader != "" && strings.HasPrefix(rangeHeader, "bytes=") {
			parts := strings.Split(strings.TrimPrefix(rangeHeader, "bytes="), "-")
			if len(parts) > 0 && parts[0] != "" {
				s, err := strconv.ParseInt(parts[0], 10, 64)
				if err == nil {
					start = s
					isPartial = true
				}
			}
		}

		if start >= fileSize {
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}

		if isPartial {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, fileSize-1, fileSize))
			w.WriteHeader(http.StatusPartialContent)
			fmt.Printf("[Server] Resuming from byte %d\n", start)
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Println("[Server] Starting new download")
		}

		// Simulate a connection drop midway through the remaining data.
		// We'll fail roughly 60% of the time to demonstrate multiple retries.
		remaining := fileSize - start
		failAfter := int64(-1)
		if rand.Float32() < 0.6 && remaining > 1024 {
			failAfter = rand.Int64N(remaining-512) + 512
		}

		written := int64(0)
		chunkSize := int64(1024)
		for written < remaining {
			if failAfter != -1 && written >= failAfter {
				fmt.Printf("[Server] Simulating connection drop after %d bytes\n", written)
				// Hijack the connection to force a close without a proper HTTP trailer/closure.
				if hj, ok := w.(http.Hijacker); ok {
					conn, _, _ := hj.Hijack()
					conn.Close()
					return
				}
				return
			}

			toWrite := chunkSize
			if toWrite > remaining-written {
				toWrite = remaining - written
			}

			n, err := w.Write(fileContent[start+written : start+written+toWrite])
			if err != nil {
				return
			}
			written += int64(n)
		}
	}))
	defer server.Close()

	fmt.Println("--- Streaming HTTP Response Resumption Example ---")

	// 3. Resile execution state.
	var bytesReceived int64
	ctx := context.Background()

	// Ensure the local file is clean.
	os.Remove(fileName)
	defer os.Remove(fileName)

	// 4. Use resile.DoErr to handle retries.
	err := resile.DoErr(ctx, func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
		if err != nil {
			return err
		}

		// Request only the missing bytes.
		if bytesReceived > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", bytesReceived))
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// If server doesn't support 206, it might have returned 200 (full content).
		// In that case, we must restart the local file.
		fileFlags := os.O_APPEND | os.O_CREATE | os.O_WRONLY
		if resp.StatusCode == http.StatusOK {
			if bytesReceived > 0 {
				fmt.Println("[Client] Server returned 200 OK, restarting download...")
				bytesReceived = 0
			}
			fileFlags = os.O_CREATE | os.O_TRUNC | os.O_WRONLY
		} else if resp.StatusCode != http.StatusPartialContent {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		f, err := os.OpenFile(fileName, fileFlags, 0644)
		if err != nil {
			return err
		}
		defer f.Close()

		// Stream content and update state.
		// io.Copy will return an error if the connection drops.
		n, err := io.Copy(f, resp.Body)
		bytesReceived += n
		fmt.Printf("[Client] Progress: %d/%d bytes received\n", bytesReceived, fileSize)

		return err
	},
		resile.WithName("http-resume"),
		resile.WithMaxAttempts(15), // Allow plenty of retries for the simulation.
		resile.WithBackoff(resile.NewFullJitter(100*time.Millisecond, 1*time.Second)),
	)

	// 5. Final validation.
	if err != nil {
		fmt.Printf("Download failed: %v\n", err)
		return
	}

	fmt.Println("Download completed successfully!")

	// Verify file integrity.
	downloaded, err := os.ReadFile(fileName)
	if err != nil {
		fmt.Printf("Error reading downloaded file: %v\n", err)
		return
	}

	if int64(len(downloaded)) != fileSize {
		fmt.Printf("Size mismatch: got %d, want %d\n", len(downloaded), fileSize)
	} else if !bytes.Equal(downloaded, fileContent) {
		fmt.Println("Content mismatch!")
	} else {
		fmt.Println("File integrity verified: content is correct.")
	}
}
