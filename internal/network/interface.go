// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package network

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ssbc/go-muxrpc/v2"
	"github.com/ssbc/go-netwrap"
	"github.com/ssbc/go-secretstream"
	refs "github.com/ssbc/go-ssb-refs"
)

// ServerEndpointDetails encapsulates the endpoint information.
// Like domain name of the room, it's ssb/secret-handshake public key and the HTTP and MUXRPC TCP ports.
type ServerEndpointDetails struct {
	RoomID refs.FeedRef

	ListenAddressMUXRPC string // defaults to ":8008"
	PublicMUXRPCPort    uint   // 0 uses the listener port
	PublicWebSocket     string // optional ws(s) URL in place of public TCP

	// Domain sets the DNS name for all the HTTP(S) URLs.
	Domain    string
	PortHTTPS uint // 0 assumes default (443)

	// UseSubdomainForAliases controls wether urls for alias resolving
	// are generated as https://$alias.$domain instead of https://$domain/alias/$alias
	UseSubdomainForAliases bool

	// Development instructs url building to happen with http and include the http port
	Development bool
}

func (sed ServerEndpointDetails) URLForAlias(a string) string {
	var u url.URL

	if sed.Development {
		u.Path = "/alias/" + a
		u.Scheme = "http"
		u.Host = fmt.Sprintf("localhost:%d", sed.PortHTTPS)
		return u.String()
	}

	u.Scheme = "https"

	if sed.UseSubdomainForAliases {
		u.Host = a + "." + sed.Domain
	} else {
		u.Host = sed.Domain
		u.Path = "/alias/" + a
	}

	return u.String()
}

// MultiserverAddress returns the public transport and SHS key used by clients.
func (sed ServerEndpointDetails) MultiserverAddress() string {
	transport := sed.PublicWebSocket
	if transport == "" {
		port := sed.PublicMUXRPCPort
		if port == 0 {
			port = DefaultPort
			if sed.ListenAddressMUXRPC != "" {
				_, listenPort, err := net.SplitHostPort(sed.ListenAddressMUXRPC)
				if err != nil {
					panic(err)
				}
				resolvedPort, err := net.LookupPort("tcp", listenPort)
				if err != nil {
					panic(err)
				}
				port = uint(resolvedPort)
			}
		}
		transport = "net:" + net.JoinHostPort(sed.Domain, strconv.Itoa(int(port)))
	}
	roomPubKey := base64.StdEncoding.EncodeToString(sed.RoomID.PubKey())
	return transport + "~shs:" + roomPubKey
}

// EndpointStat gives some information about a connected peer
type EndpointStat struct {
	ID       *refs.FeedRef
	Addr     net.Addr
	Since    time.Duration
	Endpoint muxrpc.Endpoint
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o mocked/endpoints.go . Endpoints

// Endpoints returns the connected endpoint for the passed feed,
// or false if there is none.
type Endpoints interface {
	GetEndpointFor(refs.FeedRef) (muxrpc.Endpoint, bool)
}

// Network supplies all network related functionalitiy
type Network interface {
	Connect(ctx context.Context, addr net.Addr) error
	Serve(context.Context, ...muxrpc.HandlerWrapper) error
	GetListenAddr() net.Addr

	GetAllEndpoints() []EndpointStat
	Endpoints

	GetConnTracker() ConnTracker

	// WebsockHandler returns a "middleware" like thing that is able to upgrade a
	// websocket request to a muxrpc connection and authenticate using shs.
	// It calls the next handler if it fails to upgrade the connection to websocket.
	// However, it will error on the request and not call the passed handler
	// if the websocket upgrade is successfull.
	WebsockHandler(next http.Handler) http.Handler

	io.Closer
}

// ConnTracker decides if connections should be established and keeps track of them
type ConnTracker interface {
	// Active returns true and since when a peer connection is active
	Active(net.Addr) (bool, time.Duration)

	// OnAccept receives a new connection as an argument.
	// If it decides to accept it, it returns true and a context that will be canceled once it should shut down
	// If it decides to deny it, it returns false (and a nil context)
	OnAccept(context.Context, net.Conn) (bool, context.Context)

	// OnClose notifies the tracker that a connection was closed
	OnClose(conn net.Conn) time.Duration

	// Count returns the number of open connections
	Count() uint

	// CloseAll closes all tracked connections
	CloseAll()
}

// GetFeedRefFromAddr uses netwrap to get the secretstream address and then uses ParseFeedRef
func GetFeedRefFromAddr(addr net.Addr) (refs.FeedRef, error) {
	addr = netwrap.GetAddr(addr, secretstream.NetworkString)
	if addr == nil {
		return refs.FeedRef{}, errors.New("no shs-bs address found")
	}
	ssAddr := addr.(secretstream.Addr)
	return refs.ParseFeedRef(ssAddr.String())
}
