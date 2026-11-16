// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package network

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	kitlog "go.mindeco.de/log"
	"go.mindeco.de/log/level"
)

// WebsockHandler returns a "middleware" like thing that is able to upgrade a
// websocket request to a muxrpc connection and authenticate using shs.
// It calls the next handler if it fails to upgrade the connection to websocket.
// However, it will error on the request and not call the passed handler
// if the websocket upgrade is successfull.
func (n *node) WebsockHandler(next http.Handler) http.Handler {
	var upgrader = websocket.Upgrader{
		HandshakeTimeout: handshakeTimeout,
		ReadBufferSize:   1024 * 4,
		WriteBufferSize:  1024 * 4,

		CheckOrigin: func(_ *http.Request) bool {
			return true
		},

		// 99% of the traffic will be ciphertext which is impossible to distinguish
		// from randomness and thus also hard to compress
		EnableCompression: false,

		// if upgrading fails, just call the next handler and ignore the error
		Error: func(w http.ResponseWriter, req *http.Request, _ int, _ error) {
			next.ServeHTTP(w, req)
		},
	}

	var wsh websocketHandelr
	wsh.next = next
	wsh.upgrader = &upgrader
	wsh.muxnetwork = n
	wsh.pending = make(chan struct{}, maxPendingHandshakes)

	return wsh
}

type websocketHandelr struct {
	next http.Handler

	muxnetwork *node

	upgrader *websocket.Upgrader
	pending  chan struct{}
}

func (wsh websocketHandelr) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !websocket.IsWebSocketUpgrade(req) {
		wsh.next.ServeHTTP(w, req)
		return
	}

	conn, wsConn := wsh.upgradeAndAuthenticate(w, req)
	if conn != nil {
		defer wsConn.Close()
		wsh.muxnetwork.handleConnection(req.Context(), conn, true)
	}
}

func (wsh websocketHandelr) upgradeAndAuthenticate(w http.ResponseWriter, req *http.Request) (net.Conn, *websocket.Conn) {
	select {
	case wsh.pending <- struct{}{}:
		defer func() { <-wsh.pending }()
	default:
		http.Error(w, "too many pending WebSocket handshakes", http.StatusServiceUnavailable)
		return nil, nil
	}

	remoteAddr := req.RemoteAddr
	if forwarded := req.Header.Get("X-Forwarded-For"); forwarded != "" {
		remoteAddr = strings.TrimSpace(forwarded)
	}

	wsConn, err := wsh.upgrader.Upgrade(w, req, nil)
	if err != nil {
		return nil, nil
	}
	authenticated := false
	defer func() {
		if !authenticated {
			wsConn.Close()
		}
	}()

	errLog := kitlog.With(level.Error(wsh.muxnetwork.log), "remote", remoteAddr)

	wc := NewWebsockConn(wsConn)
	if err := wc.SetDeadline(time.Now().Add(handshakeTimeout)); err != nil {
		return nil, nil
	}
	conn, err := wsh.muxnetwork.secretServer.ConnWrapper()(wc)
	if err != nil {
		errLog.Log("warning", "failed to authenticate", "err", err)
		return nil, nil
	}
	if err := wc.SetDeadline(time.Time{}); err != nil {
		return nil, nil
	}

	authenticated = true
	return conn, wsConn
}

// WebsockConn emulates a normal net.Conn from a websocket connection
type WebsockConn struct {
	r   io.Reader
	wsc *websocket.Conn
}

func NewWebsockConn(wsc *websocket.Conn) *WebsockConn {
	return &WebsockConn{
		wsc: wsc,
	}
}

func (conn *WebsockConn) Read(data []byte) (int, error) {
	for {
		if conn.r == nil {
			if err := conn.renewReader(); err != nil {
				return 0, err
			}
		}

		n, err := conn.r.Read(data)
		if err == io.EOF {
			conn.r = nil
			if n == 0 {
				continue
			}
			err = nil
		}
		if err != nil {
			return n, fmt.Errorf("wsConn: read failed: %w", err)
		}
		return n, nil
	}
}

func (conn *WebsockConn) renewReader() error {
	mt, r, err := conn.wsc.NextReader()
	if err != nil {
		err = fmt.Errorf("wsConn: failed to get reader: %w", err)
		return err
	}

	if mt != websocket.BinaryMessage {
		return fmt.Errorf("wsConn: not binary message: %v", mt)

	}

	conn.r = r
	return nil
}

func (conn *WebsockConn) Write(data []byte) (int, error) {
	writeCloser, err := conn.wsc.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return -1, fmt.Errorf("wsConn: failed to create Reader: %w", err)
	}

	n, err := io.Copy(writeCloser, bytes.NewReader(data))
	if err != nil {
		return -1, fmt.Errorf("wsConn: failed to copy data: %w", err)
	}

	return int(n), writeCloser.Close()
}

func (conn *WebsockConn) Close() error {
	return conn.wsc.Close()
}

func (conn *WebsockConn) LocalAddr() net.Addr  { return conn.wsc.LocalAddr() }
func (conn *WebsockConn) RemoteAddr() net.Addr { return conn.wsc.RemoteAddr() }

func (conn *WebsockConn) SetDeadline(t time.Time) error {
	rErr := conn.wsc.SetReadDeadline(t)
	wErr := conn.wsc.SetWriteDeadline(t)

	var err error

	if rErr != nil {
		err = fmt.Errorf("websock conn: failed to set read deadline: %w", rErr)
	}

	if wErr != nil {
		wErr = fmt.Errorf("websock conn: failed to set read deadline: %w", wErr)
		if err != nil {
			err = fmt.Errorf("both faild: %w and %s", err, wErr)
		} else {
			err = wErr
		}
	}

	return err
}
func (conn *WebsockConn) SetReadDeadline(t time.Time) error {
	return conn.wsc.SetReadDeadline(t)
}
func (conn *WebsockConn) SetWriteDeadline(t time.Time) error {
	return conn.wsc.SetWriteDeadline(t)
}
