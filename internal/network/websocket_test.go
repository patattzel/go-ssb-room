// SPDX-License-Identifier: MIT

package network

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"go.mindeco.de/log"

	"github.com/ssbc/go-ssb-room/v2/internal/maybemod/keys"
)

func TestWebSocketPendingHandshakeLimit(t *testing.T) {
	keyPair, err := keys.NewKeyPair(nil)
	require.NoError(t, err)
	network, err := New(Options{
		Logger:  log.NewNopLogger(),
		KeyPair: keyPair,
		AppKey:  make([]byte, 32),
	})
	require.NoError(t, err)

	server := httptest.NewServer(network.WebsockHandler(http.NotFoundHandler()))
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	dialer := websocket.Dialer{HandshakeTimeout: time.Second}

	var pending []*websocket.Conn
	defer func() {
		for _, conn := range pending {
			conn.Close()
		}
	}()
	for i := 0; i < maxPendingHandshakes; i++ {
		conn, _, err := dialer.Dial(wsURL, nil)
		require.NoError(t, err)
		pending = append(pending, conn)
	}
	conn, resp, err := dialer.Dial(wsURL, nil)
	if conn != nil {
		conn.Close()
	}
	require.ErrorIs(t, err, websocket.ErrBadHandshake)
	defer resp.Body.Close()
	require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

	require.NoError(t, pending[0].Close())
	require.Eventually(t, func() bool {
		conn, resp, err := dialer.Dial(wsURL, nil)
		if err != nil {
			if resp != nil {
				resp.Body.Close()
			}
			return false
		}
		conn.Close()
		return true
	}, time.Second, 10*time.Millisecond)
}
