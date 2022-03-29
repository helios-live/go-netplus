package netplus

import (
	"context"
	"io"

	"time"

	"go.ideatocode.tech/log"
)

// Piper .
type Piper struct {
	Logger  log.Logger
	Timeout time.Duration
}

// NewPiper returns a pointer to a newPiper Piper instance
func NewPiper(l log.Logger, t time.Duration) *Piper {
	return &Piper{
		Logger:  l,
		Timeout: t,
	}
}

// Run pipes data between upstream and downstream and closes one when the other closes
// times out after two hours by default
func (p Piper) Run(ctx context.Context, downstream io.ReadWriteCloser, upstream io.ReadWriteCloser) {
	var dur time.Duration
	if p.Timeout == 0 {
		dur = time.Duration(2 * time.Hour)
	} else {
		dur = p.Timeout
	}

	done := false
	cancel := func() {
		if done {
			return
		}
		done = true
		ctx.Done()
		downstream.Close()
		upstream.Close()
		p.Logger.Debug("Closing sockets")
	}
	p.idleTimeoutPipe(ctx, downstream, upstream, dur, cancel)
}

func (p Piper) idleTimeoutPipe(ctx context.Context, dst io.ReadWriter, src io.ReadWriter, timeout time.Duration,
	cancel context.CancelFunc) (written int64, err error) {
	read := make(chan int)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(timeout):

				p.Logger.Debug("idleTimeoutPipe timeout reached")
				cancel()
				return
			case <-read:
			}
		}
	}()

	go func() {
		defer cancel()

		buf := make([]byte, 32*1024)
		for {
			nr, er := dst.Read(buf)
			if nr > 0 {
				read <- nr
				nw, ew := src.Write(buf[0:nr])
				written += int64(nw)
				if ew != nil {
					err = ew
					break
				}
			}
			if er != nil {
				if er != io.EOF {
					err = er
				}
				break
			}
		}
	}()

	defer cancel()
	buf := make([]byte, 32*1024)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			read <- nr
			nw, ew := dst.Write(buf[0:nr])
			written += int64(nw)
			if ew != nil {
				err = ew
				break
			}
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
