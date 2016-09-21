package bosh

import (
	"errors"
	"log"
	"time"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/stream"
)

// ErrSessionClosed is the error returned when a session has been closed and a
// call to Element is made.
var ErrSessionClosed = errors.New("Session is closed")

type Session struct {
	processor chan *Request
	restart   chan struct{}
	elements  chan element.Element
	responder chan element.Element

	expired bool

	exit chan struct{}

	// current is the current RID being processed
	current int
	// ack is the highest RID that has been processed
	ack        int
	sid        string
	wait       time.Duration
	inactivity time.Duration
}

// NewSession creates a new session and returns it.
func NewSession(sid string, rid, hold int, wait, inactivity time.Duration) *Session {
	s := new(Session)
	s.sid = sid
	s.current = rid
	s.wait = wait
	s.inactivity = inactivity

	s.processor = make(chan *Request)
	s.elements = make(chan element.Element)
	s.responder = make(chan element.Element)
	s.restart = make(chan struct{}, 1)
	s.exit = make(chan struct{})

	requests := make(chan *Request, hold)
	buffer := make(chan element.Element)

	go s.process(requests, buffer)
	go s.response(requests)
	go s.buffer(buffer)
	return s
}

// Write handles writting elements to the underlying requests. This method does
// not implement io.Writer because only well formed XML elements can be written
// into the body of a BOSH request of response.
func (s *Session) Write(el element.Element) (err error) {
	select {
	case <-s.exit:
		err = stream.ErrStreamClosed
	case s.responder <- el:
	}
	return
}

// UnregisterRequest returns a function that can be called to remove the given
// request from the registered requests for this session. This method is mainly
// used as the timeout variable for a request so that a request that has timed
// out is not used.
func (s *Session) UnregisterRequest() func() int {
	return s.Ack
}

// Close implements io.Closer.
// TODO: This has a race condition, put a lock around it.
func (s *Session) Close() error {
	select {
	case <-s.exit:
		return errors.New("Already closed")
	default:
	}
	close(s.exit)
	return nil
}

// Element returns the next element from the session.
func (s *Session) Element() (el element.Element, err error) {
	select {
	case <-s.exit:
		// TODO: This should return something like io.EOF or ErrStreamClosed so
		// the stream can properly handle it.
		err = stream.ErrStreamClosed
	case el = <-s.elements:
	case <-s.restart:
		err = stream.ErrRequireRestart
	}
	return
}

// Process processes a request
//
// TODO: Handle processing of repeated requests
// TODO: Handle overactivity as described in
// http://xmpp.org/extensions/xep-0124.html#overactive
func (s *Session) Process(r *Request) error {
	s.processor <- r
	return nil
}

// elementRunner handles processing elements from requests and adding requests
// to a queue of available requests for writers to use. If the buffer is filled
// the oldest request is removed closed and the recieved request is added to the
// buffer. This method ensures that requests' elements are buffered in order to
// meet the requirement of in-order processing.
func (s *Session) process(queue chan *Request, buffer chan<- element.Element) {
	var requests map[int]*Request = make(map[int]*Request)
	var current int = s.current
	for {
		select {
		case <-s.exit:
			return
		case <-time.After(s.inactivity):
			log.Println("session expiring")
			s.expired = true
			s.Close()
			return
		case r := <-s.processor:
			// Handle history request
			if r.RID() < current {
			}
			requests[r.RID()] = r
			log.Println("processing request")
			select {
			case queue <- r:
			case old := <-queue:
				log.Println("Removing old requests ", old.RID())
				old.Close()
				queue <- r
			default:
				r.Close()
			}
			if r.body.Restart == true {
				log.Println("Sending restart")
				s.restart <- struct{}{}
			}
			log.Println("Request queued")
			for r, ok := requests[current]; ok; r, ok = requests[current] {
				for _, el := range r.Elements() {
					log.Println("Buffered element")
					buffer <- el
				}
				s.ack = r.RID()
				log.Println("Increasing ack")
				delete(requests, current)
				current++
			}
		}
	}
}

// elementBuffer handles receiving elements from the elementRunner and buffers
// one for a call to Element(). This is necessary because the runner cannot be
// blocked waiting for a call to Element, but we only want to send an element
// down the elements channel when we have one ready.
//
// TODO(skriptble): These are two separate concerns, split them into two
// functions and put a channel inbetween, similar to response and flush
func (s *Session) buffer(buffer <-chan element.Element) {
	var elements []element.Element
	var current element.Element
	var pending bool
	for {
		if pending {
			select {
			case <-s.exit:
				return
			case el := <-buffer:
				elements = append(elements, el)
			case s.elements <- current:
				if len(elements) > 0 {
					current, elements = elements[0], elements[1:]
					pending = true
				} else {
					pending = false
				}
			}
		} else {
			select {
			case <-s.exit:
				return
			// The only way to get here is if there are no elements in the
			// slice, therefore we can assign directly to current and set
			// pending to true.
			case el := <-buffer:
				current = el
				pending = true
			}
		}
	}
}

func (s *Session) response(queue <-chan *Request) {
	var response []element.Element = make([]element.Element, 0, 10)
	var timeout time.Duration = 1 * time.Second
	var elems chan []element.Element = make(chan []element.Element)

	go s.flush(elems, queue)

	for {
		select {
		case <-s.exit:
			// TODO(skriptble): We should add callback functions to invoke
			// when shutting down. Potentially useful to route errors back to
			// senders.
			return
		case el := <-s.responder:
			response = append(response, el)
			if len(response) == 1 {
				timeout = 50 * time.Millisecond
				continue
			}
			// Exponentially decay the timeout for flushing. This creates
			// a hard limit based on time.
			timeout = timeout / 2
		case <-time.After(timeout):
			if len(response) > 0 {
				timeout = timeout * 2
				continue
			}
			select {
			case elems <- response:
				response = make([]element.Element, 0, 10)
			default:
				timeout = 50 * time.Millisecond
			}
		}
	}
}

func (s *Session) flush(elems chan []element.Element, queue <-chan *Request) {
	for {
		select {
		case <-s.exit:
			for r := range queue {
				r.Close()
			}
			return
		case rsp := <-elems:
			for {
				// Get a request
				r, ok := <-queue
				if !ok {
					return
				}
				// Write the response to the request
				err := r.Write(rsp...)
				if err == ErrRequestClosed {
					continue
				}
				break
			}
		}
	}
}

// Ack returns the highest RID the session has processed.
func (s *Session) Ack() int {
	return s.ack
}

// SID returns the session ID of this session.
func (s *Session) SID() string {
	return s.sid
}

func (s *Session) Expired() bool {
	return s.expired
}

func (s *Session) Wait() time.Duration {
	return s.wait
}
