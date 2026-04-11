# Reliable File Downloads with HTTP Range Resumption

Imagine you're downloading a 2GB database backup or a large media asset over an unstable mobile connection. You've reached 95%, and then—*click*—the connection drops. A standard retry mechanism might kick in, but it starts the download from 0%. You've just wasted 1.9GB of bandwidth, and the next attempt might fail at 98%.

This is the "Sisyphean Download" problem. To solve it, we need a way to pick up exactly where we left off.

In this article, we'll explore how to implement reliable, resumable file downloads in Go using the HTTP `Range` header and the [Resile](https://github.com/cinar/resile) resilience library.

---

## The Challenge: Unstable Connections & Wasted Bandwidth

Standard HTTP GET requests fetch the entire resource. If the connection is interrupted, the partial data is often discarded, and the client must restart the request. In environments with high latency or intermittent connectivity (like satellite, mobile, or edge computing), this leads to:

1.  **Increased Latency:** Total time to successful download skyrockets.
2.  **Bandwidth Waste:** Multiple failed attempts consume significantly more data than the file size.
3.  **Server Load:** The server spends resources re-sending the same bytes repeatedly.

---

## The Solution: HTTP `Range` Requests

The HTTP/1.1 protocol introduced the `Range` request header. It allows a client to request only a specific portion of a resource.

For example, if you already have the first 1,024 bytes of a file, you can request the rest by sending:
`Range: bytes=1024-`

If the server supports this, it will respond with a `206 Partial Content` status code and only the requested bytes. If it doesn't support ranges, it will typically return `200 OK` and send the entire file from the beginning.

---

## Implementing Resumption with Resile

Resile's `DoErr` (or `Do`) function is perfect for wrapping this logic. By maintaining the download state *outside* the retry closure, we can dynamically adjust each retry attempt to request only the missing data.

### 1. Track the State
We need a variable to keep track of how many bytes we've successfully written to our local file.

```go
var bytesReceived int64
```

### 2. Configure the Retry Policy
We'll use `resile.DoErr` with an exponential backoff to give the network time to recover between attempts.

```go
err := resile.DoErr(ctx, func(ctx context.Context) error {
    req, _ := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)

    // If we have partial data, ask for the rest.
    if bytesReceived > 0 {
        req.Header.Set("Range", fmt.Sprintf("bytes=%d-", bytesReceived))
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err // Retry on network errors
    }
    defer resp.Body.Close()

    // Handle Server Response
    fileFlags := os.O_APPEND | os.O_CREATE | os.O_WRONLY
    if resp.StatusCode == http.StatusOK {
        // Server doesn't support Range or we're starting fresh.
        bytesReceived = 0
        fileFlags = os.O_CREATE | os.O_TRUNC | os.O_WRONLY
    } else if resp.StatusCode != http.StatusPartialContent {
        return fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }

    // Open the local file for writing/appending.
    f, _ := os.OpenFile(localPath, fileFlags, 0644)
    defer f.Close()

    // Stream the body and update our state.
    // io.Copy returns the number of bytes written before any error.
    n, err := io.Copy(f, resp.Body)
    bytesReceived += n 

    return err // If io.Copy fails (e.g., connection drop), Resile retries.
}, 
    resile.WithMaxAttempts(10),
    resile.WithBackoff(resile.NewFullJitter(100*time.Millisecond, 2*time.Second)),
)
```

---

## Why This Works

1.  **State Persistence:** The `bytesReceived` variable lives outside the retry loop. When `io.Copy` fails due to a connection drop, it returns the number of bytes it *did* manage to write. We add this to `bytesReceived`.
2.  **Adaptive Retries:** On the next retry attempt, the closure runs again. It sees that `bytesReceived > 0` and automatically adds the `Range` header to the new request.
3.  **Graceful Fallback:** If the server doesn't support ranges (returns `200 OK`), the code resets `bytesReceived` and starts over, ensuring the download still completes eventually.
4.  **Backoff & Jitter:** Using Resile's `WithBackoff` prevents "thundering herd" issues if multiple clients are trying to resume downloads from the same failing server.

---

## Real-World Considerations

-   **File Integrity:** For mission-critical files, always verify the checksum (SHA-256) after the download completes to ensure no corruption occurred during the multiple resumption steps.
-   **ETags/Last-Modified:** Ideally, you should also track the `ETag` or `Last-Modified` header. If the file on the server changes between retries, resuming will result in a corrupted file. You can use the `If-Range` header to handle this safely.
-   **Resource Cleanup:** Ensure files and response bodies are always closed to prevent resource leaks during multiple retry attempts.

---

## Conclusion

Building resilient systems isn't just about handling errors; it's about handling them *efficiently*. By combining the standard HTTP `Range` protocol with Resile's powerful retry capabilities, you can create a download experience that is both robust and bandwidth-efficient.

Check out the full [Streaming HTTP Resumption Example](https://github.com/cinar/resile/tree/main/examples/http_resume_stream) in the Resile repository to see this pattern in action.
