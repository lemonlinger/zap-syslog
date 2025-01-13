// Copyright (c) 2017 Timon Wong
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package zapsyslog

import (
	"net"
	"os"
	"syscall"
	"time"

	"go.uber.org/zap/zapcore"
)

var (
	_ zapcore.WriteSyncer = &ConnSyncer{}
)

type MetricsRecorder interface {
	RecordDialError(network, addr string)
	RecordWriteError(network, addr string)
}

// ConnSyncer describes connection sink for syslog.
type ConnSyncer struct {
	network      string
	raddr        string
	conn         net.Conn
	metrics      MetricsRecorder
	dialTimeout  time.Duration
	writeTimeout time.Duration
}

type ConnSyncerOption func(*ConnSyncer)

func WithMetricsRecorder(metrics MetricsRecorder) ConnSyncerOption {
	return func(c *ConnSyncer) {
		c.metrics = metrics
	}
}

func WithDialTimeout(dialTimeout time.Duration) ConnSyncerOption {
	return func(c *ConnSyncer) {
		c.dialTimeout = dialTimeout
	}
}

func WithWriteTimeout(writeTimeout time.Duration) ConnSyncerOption {
	return func(c *ConnSyncer) {
		c.writeTimeout = writeTimeout
	}
}

// NewConnSyncer returns a new conn sink for syslog.
func NewConnSyncer(network, raddr string, opts ...ConnSyncerOption) (*ConnSyncer, error) {
	s := &ConnSyncer{
		network: network,
		raddr:   raddr,
	}
	for _, opt := range opts {
		opt(s)
	}

	if s.metrics == nil {
		s.metrics = noopMetricsRecorder{}
	}
	if s.dialTimeout == 0 {
		s.dialTimeout = 1 * time.Second
	}
	if s.writeTimeout == 0 {
		s.writeTimeout = 1 * time.Second
	}

	err := s.connect()
	if err != nil {
		return nil, err
	}

	return s, nil
}

// connect makes a connection to the syslog server.
func (s *ConnSyncer) connect() error {
	if s.conn != nil {
		// ignore err from close, it makes sense to continue anyway
		s.conn.Close()
		s.conn = nil
	}

	var c net.Conn
	c, err := net.DialTimeout(s.network, s.raddr, s.dialTimeout)
	if err != nil {
		s.metrics.RecordDialError(s.network, s.raddr)
		return err
	}

	s.conn = c
	return nil
}

// Write writes to syslog with retry.
func (s *ConnSyncer) Write(p []byte) (n int, err error) {
	if s.conn != nil {
		s.conn.SetWriteDeadline(time.Now().Add(s.writeTimeout))
		if n, err := s.conn.Write(p); err == nil {
			return n, err
		}
		s.metrics.RecordWriteError(s.network, s.raddr)
		// No need to retry if it's a too long message error as the connection is still valid.
		if isTooLongMessageError(err) {
			return 0, err
		}
	}
	if err := s.connect(); err != nil {
		return 0, err
	}

	s.conn.SetWriteDeadline(time.Now().Add(s.writeTimeout))
	n, err = s.conn.Write(p)
	if err != nil {
		s.metrics.RecordWriteError(s.network, s.raddr)
	}
	return
}

// Sync implements zapcore.WriteSyncer interface.
func (s *ConnSyncer) Sync() error {
	return nil
}

func isTooLongMessageError(err error) bool {
	if opErr, ok := err.(*net.OpError); ok {
		if sysErr, ok := opErr.Err.(*os.SyscallError); ok {
			if errno, ok := sysErr.Err.(syscall.Errno); ok {
				return errno.Error() == "message too long"
			}
		}
	}
	return false
}

type noopMetricsRecorder struct{}

func (noopMetricsRecorder) RecordDialError(network, addr string)  {}
func (noopMetricsRecorder) RecordWriteError(network, addr string) {}
