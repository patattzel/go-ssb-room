// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package network

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/ssbc/go-muxrpc/v2"
	"github.com/ssbc/go-netwrap"
	"github.com/ssbc/go-secretstream"
	"go.mindeco.de/log"
	"go.mindeco.de/log/level"

	refs "github.com/ssbc/go-ssb-refs"
	"github.com/ssbc/go-ssb-room/v2/internal/maybemod/keys"
)

// DefaultPort is the default listening port for ScuttleButt.
const DefaultPort = 8008

const (
	handshakeTimeout     = 10 * time.Second
	maxPendingHandshakes = 64
)

type Options struct {
	Logger log.Logger

	Dialer     netwrap.Dialer
	ListenAddr net.Addr

	KeyPair     *keys.KeyPair
	AppKey      []byte
	MakeHandler func(net.Conn) (muxrpc.Handler, error)

	ConnTracker ConnTracker

	// PreSecureWrappers are applied before the shs+boxstream wrapping takes place
	// usefull for accessing the sycall.Conn to apply control options on the socket
	BefreCryptoWrappers []netwrap.ConnWrapper

	// AfterSecureWrappers are applied afterwards, usefull to debug muxrpc content
	AfterSecureWrappers []netwrap.ConnWrapper
}

type node struct {
	opts Options

	log log.Logger

	listening chan struct{}

	listenerLock sync.Mutex
	lisClose     sync.Once
	lis          net.Listener

	dialer       netwrap.Dialer
	secretServer *secretstream.Server
	secretClient *secretstream.Client
	connTracker  ConnTracker

	beforeCryptoConnWrappers []netwrap.ConnWrapper
	afterSecureConnWrappers  []netwrap.ConnWrapper

	remotesLock sync.Mutex
	remotes     map[string]muxrpc.Endpoint

	// "ssb-ws"
	httpLis     net.Listener
	httpHandler http.Handler
}

func New(opts Options) (Network, error) {
	n := &node{
		opts:    opts,
		remotes: make(map[string]muxrpc.Endpoint),
	}

	if opts.ConnTracker == nil {
		opts.ConnTracker = NewLastWinsTracker()
	}
	n.connTracker = opts.ConnTracker

	var err error

	if opts.Dialer != nil {
		n.dialer = opts.Dialer
	} else {
		n.dialer = netwrap.Dial
	}

	n.secretClient, err = secretstream.NewClient(opts.KeyPair.Pair, opts.AppKey)
	if err != nil {
		return nil, fmt.Errorf("error creating secretstream.Client: %w", err)
	}

	n.secretServer, err = secretstream.NewServer(opts.KeyPair.Pair, opts.AppKey)
	if err != nil {
		return nil, fmt.Errorf("error creating secretstream.Server: %w", err)
	}

	n.beforeCryptoConnWrappers = opts.BefreCryptoWrappers
	n.afterSecureConnWrappers = opts.AfterSecureWrappers

	n.listening = make(chan struct{})

	n.log = opts.Logger

	return n, nil
}

func (n *node) HandleHTTP(h http.Handler) {
	n.httpHandler = h
}

func (n *node) GetConnTracker() ConnTracker {
	return n.connTracker
}

// GetEndpointFor returns a muxrpc endpoint to call the remote identified by the passed feed ref
// retruns false if there is no such connection
// TODO: merge with conntracker
func (n *node) GetEndpointFor(ref refs.FeedRef) (muxrpc.Endpoint, bool) {
	n.remotesLock.Lock()
	defer n.remotesLock.Unlock()

	edp, has := n.remotes[ref.String()]
	return edp, has
}

// TODO: merge with conntracker
func (n *node) GetAllEndpoints() []EndpointStat {
	n.remotesLock.Lock()
	defer n.remotesLock.Unlock()

	var stats []EndpointStat

	for ref, edp := range n.remotes {
		id, _ := refs.ParseFeedRef(ref)
		remote := edp.Remote()
		ok, durr := n.connTracker.Active(remote)
		if !ok {
			continue
		}
		stats = append(stats, EndpointStat{
			ID:       &id,
			Addr:     remote,
			Since:    durr,
			Endpoint: edp,
		})
	}
	return stats
}

// TODO: merge with conntracker
func (n *node) addRemote(edp muxrpc.Endpoint) {
	n.remotesLock.Lock()
	defer n.remotesLock.Unlock()
	r, err := GetFeedRefFromAddr(edp.Remote())
	if err != nil {
		panic(err)
	}
	// ref := r.String()
	// if oldEdp, has := n.remotes[ref]; has {
	// n.log.Log("remotes", "previous active", "ref", ref)
	// c := client.FromEndpoint(oldEdp)
	// _, err := c.Whoami()
	// if err == nil {
	// 	// old one still works
	// 	return
	// }
	// }
	// replace with new
	n.remotes[r.String()] = edp
}

// TODO: merge with conntracker
func (n *node) removeRemote(edp muxrpc.Endpoint) {
	n.remotesLock.Lock()
	defer n.remotesLock.Unlock()
	r, err := GetFeedRefFromAddr(edp.Remote())
	if err != nil {
		panic(err)
	}
	delete(n.remotes, r.String())
}

func (n *node) handleConnection(ctx context.Context, origConn net.Conn, isServer bool, hws ...muxrpc.HandlerWrapper) {
	// TODO: overhaul events and logging levels
	conn, err := n.applyConnWrappers(origConn)
	if err != nil {
		origConn.Close()
		level.Error(n.log).Log("msg", "node/Serve: failed to wrap connection", "err", err)
		return
	}

	// TODO: obfuscate the remote address by nulling bytes in it before printing ip and pubkey in full
	remoteRef, err := GetFeedRefFromAddr(conn.RemoteAddr())
	if err != nil {
		conn.Close()
		level.Error(n.log).Log("conn", "not shs authorized", "err", err)
		return
	}
	rLogger := log.With(n.log, "peer", remoteRef.ShortSigil())

	ok, ctx := n.connTracker.OnAccept(ctx, conn)
	if !ok {
		err := conn.Close()
		level.Debug(rLogger).Log("conn", "ignored", "err", err)
		return
	}

	defer func() {
		n.connTracker.OnClose(conn)
		conn.Close()
		origConn.Close()
	}()

	h, err := n.opts.MakeHandler(conn)
	if err != nil {
		level.Debug(rLogger).Log("conn", "mkHandler", "err", err)
		return
	}

	for _, hw := range hws {
		h = hw(h)
	}

	pkr := muxrpc.NewPacker(conn)
	filtered := level.NewFilter(n.log, level.AllowInfo())
	edp := muxrpc.Handle(pkr, h,
		muxrpc.WithContext(ctx),
		muxrpc.WithLogger(filtered),
		// _isServer_ defines _are we a server_.
		// the muxrpc option asks are we _talking_ to a server > inverted
		muxrpc.WithIsServer(!isServer),
	)
	n.addRemote(edp)

	srv := edp.(muxrpc.Server)

	err = srv.Serve()
	// level.Warn(n.log).Log("conn", "serve-return", "err", err)
	if err != nil && !isConnBrokenErr(err) && !errors.Is(err, context.Canceled) {
		level.Debug(rLogger).Log("conn", "serve exited", "err", err)
	}
	n.removeRemote(edp)

	// panic("serve exited")
	err = edp.Terminate()
	// level.Error(n.log).Log("conn", "serve-defer-terminate", "err", err)
}

// Serve starts the network listener and configured resources like local discovery.
// Canceling the passed context makes the function return. Defers take care of stopping these resources.
func (n *node) Serve(ctx context.Context, wrappers ...muxrpc.HandlerWrapper) error {
	n.listenerLock.Lock()
	lis, err := net.Listen(n.opts.ListenAddr.Network(), n.opts.ListenAddr.String())
	if err != nil {
		n.listenerLock.Unlock()
		return fmt.Errorf("error creating listener: %w", err)
	}
	n.lis = lis
	n.lisClose = sync.Once{}
	close(n.listening)
	n.listenerLock.Unlock()

	stop := context.AfterFunc(ctx, func() { lis.Close() })
	defer stop()
	defer func() {
		n.listenerLock.Lock()
		defer n.listenerLock.Unlock()
		n.lisClose.Do(func() { lis.Close() })
		n.lis = nil
		n.listening = make(chan struct{})
	}()

	// Keep slow or incomplete handshakes off the accept loop.
	pending := make(chan struct{}, maxPendingHandshakes)
	defer level.Debug(n.log).Log("event", "network listen loop exited")
	for {
		conn, err := lis.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		select {
		case pending <- struct{}{}:
			go func() {
				secured, err := n.authenticateIncoming(ctx, conn)
				<-pending
				if err != nil {
					conn.Close()
					level.Debug(n.log).Log("event", "SSB handshake failed", "err", err)
					return
				}
				n.handleConnection(ctx, secured, true, wrappers...)
			}()
		default:
			conn.Close()
		}
	}
}

func (n *node) authenticateIncoming(ctx context.Context, conn net.Conn) (net.Conn, error) {
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	if err := conn.SetDeadline(time.Now().Add(handshakeTimeout)); err != nil {
		return nil, err
	}
	wrapped := conn
	for _, wrap := range n.beforeCryptoConnWrappers {
		var err error
		wrapped, err = wrap(wrapped)
		if err != nil {
			return nil, err
		}
	}
	secured, err := n.secretServer.ConnWrapper()(wrapped)
	if err != nil {
		return nil, err
	}
	return secured, conn.SetDeadline(time.Time{})
}

func (n *node) Connect(ctx context.Context, addr net.Addr) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	shsAddr := netwrap.GetAddr(addr, "shs-bs")
	if shsAddr == nil {
		return errors.New("node/connect: expected an address containing an shs-bs addr")
	}

	var pubKey = make(ed25519.PublicKey, ed25519.PublicKeySize)
	if shsAddr, ok := shsAddr.(secretstream.Addr); ok {
		copy(pubKey[:], shsAddr.PubKey)
	} else {
		return errors.New("node/connect: expected shs-bs address to be of type secretstream.Addr")
	}

	conn, err := n.dialer(netwrap.GetAddr(addr, "tcp"), append(n.beforeCryptoConnWrappers,
		n.secretClient.ConnWrapper(pubKey))...)
	if err != nil {
		if conn != nil {
			conn.Close()
		}
		return fmt.Errorf("node/connect: error dialing: %w", err)
	}

	go func(c net.Conn) {
		n.handleConnection(ctx, c, false)
	}(conn)
	return nil
}

// GetListenAddr waits for Serve() to be called!
func (n *node) GetListenAddr() net.Addr {
	n.listenerLock.Lock()
	listening := n.listening
	n.listenerLock.Unlock()
	<-listening

	n.listenerLock.Lock()
	defer n.listenerLock.Unlock()
	if n.lis != nil {
		return netwrap.WrapAddr(n.lis.Addr(), n.secretServer.Addr())
	}
	level.Error(n.log).Log("msg", "listener not ready")
	return nil
}

func (n *node) applyConnWrappers(conn net.Conn) (net.Conn, error) {
	for i, cw := range n.afterSecureConnWrappers {
		var err error
		conn, err = cw(conn)
		if err != nil {
			return nil, fmt.Errorf("error applying connection wrapper #%d: %w", i, err)
		}
	}
	return conn, nil
}

func (n *node) Close() error {
	if n.httpLis != nil {
		err := n.httpLis.Close()
		if err != nil {
			return fmt.Errorf("ssb: failed to close http listener: %w", err)
		}
	}
	n.listenerLock.Lock()
	defer n.listenerLock.Unlock()
	if n.lis != nil {
		var closeErr error
		n.lisClose.Do(func() {
			closeErr = n.lis.Close()
		})
		if closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
			return fmt.Errorf("ssb: network node failed to close it's listener: %w", closeErr)
		}
	}

	n.remotesLock.Lock()
	defer n.remotesLock.Unlock()
	for addr, edp := range n.remotes {
		if err := edp.Terminate(); err != nil {
			n.log.Log("event", "failed to terminate endpoint", "addr", addr, "err", err)
		}
	}

	if cnt := n.connTracker.Count(); cnt > 0 {
		n.log.Log("event", "warning", "msg", "still open connections", "count", cnt)
		n.connTracker.CloseAll()
	}

	return nil
}
