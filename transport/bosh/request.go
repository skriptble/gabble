package bosh

import (
	"errors"
	"io"
	"time"

	"github.com/skriptble/nine/element"
)

var ErrRequestClosed = errors.New("request has already been responded to")

type Request struct {
	rid  int
	wait time.Duration
	sid  string
	body Body

	proceed  chan struct{}
	closed   chan struct{}
	payload  []element.Element
	response Body
	spent    bool

	// The function should return the highest rid processed.
	// ack is a function called when the request is being spent and sending a
	// response. This can either happen due to a timeout or because data to
	// write has been received. This function should return the highest rid
	// processed by the session
	ack func() int
}

func NewRequest(rid int, wait time.Duration, sid string, b, response Body, ack func() int) *Request {
	return &Request{
		rid:      rid,
		wait:     wait,
		sid:      sid,
		body:     b,
		response: response,
		ack:      ack,
		proceed:  make(chan struct{}),
		closed:   make(chan struct{}),
	}
}

// Write adds the given elements as the payload for the response body.
func (r *Request) Write(els ...element.Element) error {
	if r.spent {
		return ErrRequestClosed
	}
	r.payload = els
	r.spent = true
	close(r.proceed)
	return nil
}

func (r *Request) Close() {
	close(r.closed)
}

// RID returns the request ID of this Request.
func (r *Request) RID() int { return r.rid }

func (r *Request) Elements() []element.Element {
	return r.body.Children
}

func (r *Request) Handle(w io.Writer) {
	select {
	case <-r.proceed:
	case <-r.closed:
		r.spent = true
	case <-time.After(r.wait):
		r.spent = true
	}
	r.response.Ack = r.ack()
	r.response.Children = r.payload
	w.Write(r.response.TransformElement().WriteBytes())
}
