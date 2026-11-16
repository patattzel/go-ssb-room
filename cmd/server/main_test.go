// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWebSocketUpgradeRateLimit(t *testing.T) {
	for _, tc := range []struct {
		name  string
		proxy net.IP
	}{
		{name: "direct"},
		{name: "behind a trusted proxy", proxy: net.ParseIP("192.0.2.1")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler, err := rateLimitHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}), tc.proxy != nil)
			require.NoError(t, err)
			if tc.proxy != nil {
				handler = trustCloudronProxy(tc.proxy, handler)
			}

			request := func(path, clientIP string, attempt int, upgrade bool) int {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				req.RemoteAddr = fmt.Sprintf("%s:%d", clientIP, 10000+attempt)
				spoofed := fmt.Sprintf("203.0.113.%d", attempt)
				if tc.proxy != nil {
					req.RemoteAddr = fmt.Sprintf("%s:%d", tc.proxy, 10000+attempt)
					spoofed += ", " + clientIP
				}
				req.Header.Set("X-Forwarded-For", spoofed)
				if upgrade {
					req.Header.Set("Connection", "Upgrade")
					req.Header.Set("Upgrade", "websocket")
				}
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				return rec.Code
			}

			limited := false
			for attempt := 1; attempt <= 80; attempt++ {
				if request(fmt.Sprintf("/path-%d", attempt), "198.51.100.1", attempt, true) == http.StatusTooManyRequests {
					limited = true
					break
				}
			}
			require.True(t, limited, "changing the path, port, and spoofed header must not bypass the upgrade rate limit")
			require.Equal(t, http.StatusNoContent, request("/path-1", "198.51.100.2", 1, true))
			require.Equal(t, http.StatusNoContent, request("/path-1", "198.51.100.1", 1, false))
		})
	}
}
