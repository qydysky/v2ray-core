//go:build !wasm && !confonly
// +build !wasm,!confonly

package domainsocket

import (
	"context"
	gotls "crypto/tls"
	"strings"

	"github.com/v2fly/v2ray-core/v4/common"
	"github.com/v2fly/v2ray-core/v4/common/net"
	"github.com/v2fly/v2ray-core/v4/transport/internet"
	"github.com/v2fly/v2ray-core/v4/transport/internet/tls"
)

type Listener struct {
	addr      *net.UnixAddr
	ln        net.Listener
	tlsConfig *gotls.Config
	config    *Config
	addConn   internet.ConnHandler
}

func Listen(ctx context.Context, address net.Address, port net.Port, streamSettings *internet.MemoryStreamConfig, handler internet.ConnHandler) (internet.Listener, error) {
	settings := streamSettings.ProtocolSettings.(*Config)
	addr, err := settings.GetUnixAddr()
	if err != nil {
		return nil, err
	}

	unixListener, err := net.Listen("unix", addr.Name)
	if err != nil {
		return nil, newError("failed to listen domain socket").Base(err).AtWarning()
	}

	ln := &Listener{
		addr:    addr,
		ln:      unixListener,
		config:  settings,
		addConn: handler,
	}

	if config := tls.ConfigFromStreamSettings(streamSettings); config != nil {
		ln.tlsConfig = config.GetTLSConfig()
	}

	go ln.run()

	return ln, nil
}

func (ln *Listener) Addr() net.Addr {
	return ln.addr
}

func (ln *Listener) Close() error {
	return ln.ln.Close()
}

func (ln *Listener) run() {
	for {
		conn, err := ln.ln.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "closed") {
				break
			}
			newError("failed to accepted raw connections").Base(err).AtWarning().WriteToLog()
			continue
		}

		if ln.tlsConfig != nil {
			conn = tls.Server(conn, ln.tlsConfig)
		}

		ln.addConn(internet.Connection(conn))
	}
}

func init() {
	common.Must(internet.RegisterTransportListener(protocolName, Listen))
}
