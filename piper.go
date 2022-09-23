package netplus

import (
	"context"
	"errors"
	"io"
	"sync/atomic"

	"time"

	"go.ideatocode.tech/log"
)

// ErrShortWrite means that a write accepted fewer bytes than requested
// but failed to return an explicit error.
var ErrShortWrite = errors.New("short write")

// errInvalidWrite means that a write returned an impossible count.
var errInvalidWrite = errors.New("invalid write result")

// Piper .
type Piper struct {
	Logger  log.Logger
	Timeout time.Duration
	debug   bool
}

// NewPiper returns a pointer to a newPiper Piper instance
func NewPiper(l log.Logger, t time.Duration) *Piper {
	return &Piper{
		Logger:  l,
		Timeout: t,
	}
}

// Debug turns debugging on and off
func (p *Piper) Debug(debug bool) {
	p.debug = debug
}

// Run pipes data between upstream and downstream and closes one when the other closes
// times out after two hours by default
func (p *Piper) Run(ctx context.Context, downstream io.ReadWriteCloser, upstream io.ReadWriteCloser) (written int64, err error) {
	var dur time.Duration
	if p.Timeout == 0 {
		dur = time.Duration(2 * time.Hour)
	} else {
		dur = p.Timeout
	}

	return p.idleTimeoutPipe(ctx, downstream, upstream, dur)
}

func (p *Piper) idleTimeoutPipe(ctx context.Context, dst io.ReadWriteCloser, src io.ReadWriteCloser, timeout time.Duration) (written int64, err error) {

	var done int32 = 0

	ctx, closeContext := context.WithCancel(ctx)

	closeBothSockets := func() {
		if atomic.CompareAndSwapInt32(&done, 1, 1) {
			return
		}
		closeContext()
		src.Close()
		dst.Close()
	}
	timekeeper := make(chan struct{})
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(timeout):
				if p.debug {
					p.Logger.Debug("idletimeoutpipe: timeout reached")
				}
				closeBothSockets()
				return
			case <-timekeeper:
			}
		}
	}()
	var w1, w2 int64
	var err1, err2 error
	var ec = make(chan error, 2)
	go func() {
		w1, err1 = copy(src, dst, timekeeper)
		ec <- err1
	}()
	go func() {
		w2, err2 = copy(dst, src, timekeeper)
		ec <- err2
	}()
	firstErr := <-ec
	closeBothSockets()
	<-ec // empty the channel, equivallent to wg.Wait

	return w1 + w2, firstErr
}

func copy(src io.Reader, dst io.Writer, timekeeper chan struct{}) (written int64, err error) {

	size := 32 * 1024
	if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
		if l.N < 1 {
			size = 1
		} else {
			size = int(l.N)
		}
	}
	buf := make([]byte, size)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errInvalidWrite
				}
			}
			written += int64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = ErrShortWrite
				break
			}
			timekeeper <- struct{}{}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}
