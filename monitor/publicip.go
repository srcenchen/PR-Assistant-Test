package monitor

import (
	"io"
	"net"
	"net/http"
	"op-status/config"
	"op-status/models"
	"strings"
	"sync"
	"time"
)

var (
	lastPublicIPCheck  time.Time
	lastPublicIPResult models.PublicIPMetrics
	publicIPMu         sync.Mutex
)

// checkPublicIP performs the actual HTTP probe to get the public IP.
// It is a pure function: no locking, no caching — callers are responsible for that.
func checkPublicIP() models.PublicIPMetrics {
	now := time.Now()
	var metrics models.PublicIPMetrics
	metrics.LastCheck = now.In(config.LocalLocation)

	startTime := now

	client := http.Client{
		Timeout: config.PublicIPTimeout,
	}

	req, err := http.NewRequest("GET", config.PublicIPCheckURL, nil)
	if err != nil {
		metrics.Connected = false
		metrics.Error = err.Error()
		metrics.ResponseTime = time.Since(startTime).Milliseconds()
		return metrics
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "curl/8.7.1")

	resp, err := client.Do(req)
	if err != nil {
		metrics.Connected = false
		metrics.Error = err.Error()
		metrics.ResponseTime = time.Since(startTime).Milliseconds()
		return metrics
	}
	defer resp.Body.Close()

	metrics.ResponseTime = time.Since(startTime).Milliseconds()

	if resp.StatusCode != http.StatusOK {
		metrics.Connected = false
		metrics.Error = resp.Status
		return metrics
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		metrics.Connected = false
		metrics.Error = err.Error()
		return metrics
	}

	ipStr := strings.TrimSpace(string(body))
	ip := net.ParseIP(ipStr)

	if ip != nil && ip.To4() != nil {
		metrics.Connected = true
		metrics.IPAddress = ipStr
	} else {
		metrics.Connected = false
		metrics.Error = "Only IPv4 is supported, IPv6 detected"
		metrics.IPAddress = ""
	}

	return metrics
}

// storePublicIPResult atomically writes a result into the cache.
func storePublicIPResult(result models.PublicIPMetrics) {
	publicIPMu.Lock()
	lastPublicIPResult = result
	lastPublicIPCheck = time.Now()
	publicIPMu.Unlock()
}

// GetPublicIPMetrics returns public IP information with automatic 1-hour caching.
// When the cache is expired, it launches a background goroutine to probe and waits
// up to PublicIPCheckMaxWait. If the probe finishes within that window the fresh
// result is returned; otherwise the last cached result (if any) is returned immediately,
// and the background goroutine continues running to update the cache for future calls.
func GetPublicIPMetrics() models.PublicIPMetrics {
	publicIPMu.Lock()
	now := time.Now()
	shouldCheck := lastPublicIPCheck.IsZero() || now.Sub(lastPublicIPCheck) >= 1*time.Hour

	if !shouldCheck {
		cached := lastPublicIPResult
		publicIPMu.Unlock()
		return cached
	}

	// Snapshot the previous result before unlocking, in case we need to fall back.
	haveCached := false
	prevResult := lastPublicIPResult
	if !lastPublicIPCheck.IsZero() {
		haveCached = true
	}
	publicIPMu.Unlock()

	// Launch async probe.
	resultCh := make(chan models.PublicIPMetrics, 1)
	go func() {
		result := checkPublicIP()
		storePublicIPResult(result)
		resultCh <- result
	}()

	// Wait up to PublicIPCheckMaxWait for the fresh result.
	select {
	case result := <-resultCh:
		return result
	case <-time.After(config.PublicIPCheckMaxWait):
		// Timeout: return last cached data if available.
		if haveCached {
			return prevResult
		}
		// No cached data at all — return a disconnected result.
		return models.PublicIPMetrics{
			LastCheck:    now.In(config.LocalLocation),
			Connected:    false,
			Error:        "public IP check timeout (network may be down)",
			ResponseTime: config.PublicIPCheckMaxWait.Milliseconds(),
		}
	}
}

// RefreshPublicIP forces an immediate synchronous refresh of public IP information,
// bypassing the cache. Since the caller explicitly requests a refresh, this blocks
// until the HTTP probe completes (up to PublicIPTimeout).
func RefreshPublicIP() models.PublicIPMetrics {
	result := checkPublicIP()
	storePublicIPResult(result)
	return result
}
