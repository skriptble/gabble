package bosh

import (
	"reflect"
	"testing"
	"time"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/stream"
)

func TestSessionWrite(t *testing.T) {
	t.Parallel()

	var el, want, got element.Element
	var err error
	var rsp chan element.Element
	var exit chan struct{}
	var s *Session
	// Should return stream.ErrStreamClosed when session is exited
	exit = make(chan struct{})
	s = &Session{exit: exit}
	close(exit)
	err = s.Write(el)
	if err != stream.ErrStreamClosed {
		t.Error("Should return stream closed error when session is exited")
		t.Errorf("\nWant:%s\nGot :%s", stream.ErrStreamClosed, err)
	}
	// Should send element to responder channel
	rsp = make(chan element.Element, 1)
	s = &Session{responder: rsp}
	want = element.New("foo")
	err = s.Write(want)
	if err != nil {
		t.Errorf("Expected <nil> error but got %s", err)
	}
	if len(rsp) != 1 {
		t.Errorf("Expected responder channel to have one value, has %d", len(rsp))
	}
	got = <-rsp
	if !reflect.DeepEqual(want, got) {
		t.Error("Should send element to responder channel")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
}

func TestSessionUnregisterRequest(t *testing.T) {
	t.Parallel()
	// Calling func returned from UnregisterRequest should return the current
	// ack of the session
	var rid int = 1928492834
	var s = new(Session)
	f := s.UnregisterRequest()
	s.ack = rid
	want := rid
	got := f()
	if want != got {
		t.Error("Calling func returned from UnregisterRequest should return the current ack of the session")
		t.Errorf("\nWant:%d\nGot :%d", want, got)
	}
}

func TestSessionClose(t *testing.T) {
	t.Parallel()
	var s *Session
	var err error
	var exit chan struct{}
	// Should return Already closed when attempting to close a session that has
	// been closed
	exit = make(chan struct{})
	close(exit)
	s = &Session{exit: exit}
	err = s.Close()
	if err.Error() != "Already closed" {
		t.Errorf("Should return 'Already closed' when attempting to close a session that has been closed. Received %s", err)
	}

	// Should close the exit channel when called.
	exit = make(chan struct{})
	s = &Session{exit: exit}
	err = s.Close()
	if err != nil {
		t.Errorf("Unexpected error while closing session: %s", err)
	}
	select {
	case <-exit:
	default:
		t.Error("Should close the exit channel when close is called")
	}
}

func TestSessionElement(t *testing.T) {
	t.Parallel()

	var want, got element.Element
	var err, gotErr error
	var s *Session
	var exit, restart chan struct{}
	var els chan element.Element

	// Should return stream closed error when session has exited
	exit = make(chan struct{})
	close(exit)
	s = &Session{exit: exit}
	want = element.Element{}
	err = stream.ErrStreamClosed
	got, gotErr = s.Element()
	if !reflect.DeepEqual(want, got) {
		t.Error("Should recieve empty element")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
	if !reflect.DeepEqual(err, gotErr) {
		t.Error("Should return stream closed error when session has exited")
		t.Errorf("\nWant:%s\nGot :%s", err, gotErr)
	}
	// Should return element from elements channel
	els = make(chan element.Element, 1)
	s = &Session{elements: els}
	want = element.New("foo")
	els <- want
	err = nil
	got, gotErr = s.Element()
	if !reflect.DeepEqual(want, got) {
		t.Error("Should return element form elements channel")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
	if !reflect.DeepEqual(err, gotErr) {
		t.Error("Should return nil error")
		t.Errorf("\nWant:%s\nGot :%s", err, gotErr)
	}
	// Should return require restart when restart channel has value available
	restart = make(chan struct{}, 1)
	restart <- struct{}{}
	s = &Session{restart: restart}
	want = element.Element{}
	err = stream.ErrRequireRestart
	got, gotErr = s.Element()
	if !reflect.DeepEqual(want, got) {
		t.Error("Should recieve empty element")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
	if !reflect.DeepEqual(err, gotErr) {
		t.Error("Should return require restart error when restart channel has a value available")
		t.Errorf("\nWant:%s\nGot :%s", err, gotErr)
	}
}

func TestSessionProcess(t *testing.T) {
	t.Parallel()

	var s *Session
	var processor chan *Request
	var err error
	var want, got *Request
	// Should pass request to processor channel
	want = &Request{rid: 1929302}
	processor = make(chan *Request, 1)
	s = &Session{processor: processor}
	err = s.Process(want)
	if err != nil {
		t.Errorf("Unexpected error while running process: %s", err)
	}
	if len(processor) != 1 {
		t.Errorf("Expected legnth of processor to be 1, not %d", len(processor))
	}
	got = <-processor
	if !reflect.DeepEqual(want, got) {
		t.Error("Should pass request to processor channel")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
}

func TestSessionSID(t *testing.T) {
	t.Parallel()

	var s *Session
	var want, got string
	// Should return session ID
	want = "foobar"
	s = &Session{sid: want}
	got = s.SID()
	if want != got {
		t.Error("Should return session ID")
		t.Errorf("\nWant:%s\nGot :%s", want, got)
	}
}

func TestSessionExpired(t *testing.T) {
	t.Parallel()

	var s *Session
	var want, got bool

	// Should return expired
	want = true
	s = &Session{expired: want}
	got = s.Expired()
	if want != got {
		t.Error("Should return expired")
		t.Errorf("\nWant:%t\nGot :%t", want, got)
	}
}

func TestSessionWait(t *testing.T) {
	t.Parallel()

	var s *Session
	var want, got time.Duration

	// Should return wait
	want = 37 * time.Second
	s = &Session{wait: want}
	got = s.Wait()
	if want != got {
		t.Error("Should return wait")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
}

func TestSessionprocess(t *testing.T) {
	t.Parallel()

	var s *Session
	var r, r2, want, got *Request
	var inactivity time.Duration
	var queue chan *Request
	var buffer chan element.Element
	var processor chan *Request
	var exit, restart chan struct{}
	var el, el2, gotEl element.Element
	// Should exit when exit channel is closed
	processor = make(chan *Request)
	exit = make(chan struct{})
	inactivity = 10 * time.Second
	s = &Session{
		inactivity: inactivity,
		exit:       exit,
		processor:  processor,
	}
	go s.process(queue, buffer)
	close(exit)
	select {
	case processor <- &Request{}:
		t.Error("Closing exit channel should exit processor goroutine")
	default:
	}
	// Should expire after inactivity period
	processor = make(chan *Request)
	inactivity = 1 * time.Nanosecond
	exit = make(chan struct{})
	s = &Session{
		inactivity: inactivity,
		exit:       exit,
		processor:  processor,
	}
	s.process(queue, buffer)
	if !s.expired {
		t.Error("Session should be set to expired after inactivity period")
	}
	select {
	case <-exit:
	default:
		t.Error("Session exit channel should be closed after expired inactivity period")
	}
	// Should process request
	processor = make(chan *Request)
	queue = make(chan *Request, 1)
	buffer = make(chan element.Element)
	exit = make(chan struct{})
	inactivity = 10 * time.Second
	// ---> Should queue request
	s = &Session{
		inactivity: inactivity,
		processor:  processor,
		exit:       exit,
		current:    29389410,
	}
	want = &Request{rid: 29389412}
	go s.process(queue, buffer)
	processor <- want
	close(exit)
	if len(queue) != 1 {
		t.Error("Process should queue the request")
	}
	got = <-queue
	if !reflect.DeepEqual(want, got) {
		t.Error("Process should queue the request")
		t.Errorf("\nWant:%+v\nGOt :%+v", want, got)
	}
	// ---> Should drop & close oldest request and add new request to end of
	//      queue
	processor = make(chan *Request)
	queue = make(chan *Request, 1)
	buffer = make(chan element.Element)
	inactivity = 10 * time.Second
	exit = make(chan struct{})
	r = &Request{
		rid:    2283740,
		closed: make(chan struct{}),
	}
	queue <- r
	want = &Request{rid: 29389413}
	s = &Session{
		inactivity: inactivity,
		processor:  processor,
		exit:       exit,
		current:    29389411,
	}
	go s.process(queue, buffer)
	processor <- want
	close(exit)
	if len(queue) != 1 {
		t.Error("Process should drop & close oldest request and add new requests to queue")
	}
	got = <-queue
	select {
	case <-r.closed:
	default:
		t.Error("Process should close the oldest rqeuest")
	}
	if !reflect.DeepEqual(want, got) {
		t.Error("Process should queue the request")
		t.Errorf("\nWant:%+v\nGOt :%+v", want, got)
	}
	// ---> Should close the request if the queue can't be read from or written to
	processor = make(chan *Request)
	queue = make(chan *Request)
	buffer = make(chan element.Element)
	inactivity = 10 * time.Second
	exit = make(chan struct{})
	r = &Request{
		rid:    23983194,
		closed: make(chan struct{}),
	}
	s = &Session{
		inactivity: inactivity,
		processor:  processor,
		exit:       exit,
		current:    23983193,
	}
	go s.process(queue, buffer)
	processor <- r
	close(exit)
	select {
	case <-r.closed:
	default:
		t.Error("Should close the request if the queue can't be read from or written to")
	}
	// ---> Should send restart request if request body has Restart set to true
	processor = make(chan *Request)
	queue = make(chan *Request)
	buffer = make(chan element.Element)
	inactivity = 10 * time.Second
	exit, restart = make(chan struct{}), make(chan struct{})
	r = &Request{
		rid:    293281938,
		body:   Body{Restart: true},
		closed: make(chan struct{}),
	}
	s = &Session{
		inactivity: inactivity,
		processor:  processor,
		exit:       exit,
		restart:    restart,
		current:    293281937,
	}
	go s.process(queue, buffer)
	processor <- r
	close(exit)
	select {
	case <-restart:
	default:
		t.Error("Should send restart request if request body has Restart set to true")
	}
	// ---> Should pass elements from request down buffer and increment ack of
	//      the session
	processor = make(chan *Request)
	queue = make(chan *Request)
	buffer = make(chan element.Element, 1)
	inactivity = 10 * time.Second
	exit = make(chan struct{})
	el = element.New("foobar")
	r = &Request{
		rid:    293281938,
		body:   Body{Children: []element.Element{el}},
		closed: make(chan struct{}),
	}
	s = &Session{
		inactivity: inactivity,
		processor:  processor,
		exit:       exit,
		restart:    restart,
		current:    293281938,
	}
	go s.process(queue, buffer)
	processor <- r
	close(exit)

	select {
	case gotEl = <-buffer:
	case <-time.After(2 * time.Second):
		t.Error("Timed out waiting for value from buffer")
	}

	if !reflect.DeepEqual(el, gotEl) {
		t.Error("Should pass elements from request down buffer")
		t.Errorf("\nWant:%+v\nGot :%+v", el, gotEl)
	}
	if s.ack != r.rid {
		t.Error("Should increment ack of the session")
		t.Errorf("\nWant:%d\nGot :%d", r.rid, s.ack)
	}
	// ---> Should process requests in order
	processor = make(chan *Request)
	queue = make(chan *Request)
	buffer = make(chan element.Element, 2)
	inactivity = 10 * time.Second
	exit = make(chan struct{})
	el = element.New("foobar")
	el2 = element.New("bazquux")
	r = &Request{
		rid:    29385023,
		body:   Body{Children: []element.Element{el}},
		closed: make(chan struct{}),
	}
	r2 = &Request{
		rid:    29385024,
		body:   Body{Children: []element.Element{el2}},
		closed: make(chan struct{}),
	}
	s = &Session{
		inactivity: inactivity,
		processor:  processor,
		exit:       exit,
		restart:    restart,
		current:    29385023,
	}
	go s.process(queue, buffer)
	processor <- r
	processor <- r2
	close(exit)
	select {
	case gotEl = <-buffer:
	case <-time.After(2 * time.Second):
		t.Error("Timed out waiting for element from buffer")
	}

	if !reflect.DeepEqual(el, gotEl) {
		t.Error("Should process requests in order")
		t.Errorf("\nWant:%+v\nGot :%+v", el, gotEl)
	}
	gotEl = <-buffer
	if !reflect.DeepEqual(el2, gotEl) {
		t.Error("Should process requests in order")
		t.Errorf("\nWant:%+v\nGot :%+v", el, gotEl)
	}
	if s.ack != r2.rid {
		t.Error("Should increment ack of the session")
		t.Errorf("\nWant:%d\nGot :%d", r.rid, s.ack)
	}

}

func TestSessionbuffer(t *testing.T) {
	t.Parallel()

	var s *Session
	var el, el2, got element.Element
	var buffer chan element.Element
	var elements chan element.Element
	var exit chan struct{}

	// Should read from buffer channel
	buffer = make(chan element.Element)
	el = element.New("foo")
	exit = make(chan struct{})
	s = &Session{exit: exit}
	go s.buffer(buffer)
	select {
	case buffer <- el:
	case <-time.After(2 * time.Second):
		t.Error("Should read from buffer channel")
	}
	close(exit)
	// Should not send value when one is not available
	buffer = make(chan element.Element)
	exit = make(chan struct{})
	elements = make(chan element.Element)
	s = &Session{
		exit:     exit,
		elements: elements,
	}
	go s.buffer(buffer)
	select {
	case <-s.elements:
		t.Error("Should not send value when one is not available")
	default:
	}
	close(exit)
	// Should send element when one is available
	buffer = make(chan element.Element)
	elements = make(chan element.Element)
	el, el2 = element.New("foo"), element.New("bar")
	exit = make(chan struct{})
	s = &Session{
		exit:     exit,
		elements: elements,
	}
	go s.buffer(buffer)
	select {
	case buffer <- el:
	case <-time.After(2 * time.Second):
		t.Error("Should read from buffer channel")
	}
	select {
	case got = <-s.elements:
	case <-time.After(2 * time.Second):
		t.Error("Should send element when one is available")
	}
	if !reflect.DeepEqual(el, got) {
		t.Error("Element passed into buffer should be read out.")
		t.Errorf("\nWant:%+v\nGot :%+v", el, got)
	}

	select {
	case buffer <- el:
	case <-time.After(2 * time.Second):
		t.Error("Should read from buffer channel")
	}
	select {
	case buffer <- el2:
	case <-time.After(2 * time.Second):
		t.Error("Should read from buffer channel")
	}
	select {
	case got = <-s.elements:
	case <-time.After(2 * time.Second):
		t.Error("Should send element when one is available")
	}
	if !reflect.DeepEqual(el, got) {
		t.Error("Element passed into buffer should be read out.")
		t.Errorf("\nWant:%+v\nGot :%+v", el, got)
	}
	select {
	case got = <-s.elements:
	case <-time.After(2 * time.Second):
		t.Error("Should send element when one is available")
	}
	if !reflect.DeepEqual(el2, got) {
		t.Error("Element passed into buffer should be read out.")
		t.Errorf("\nWant:%+v\nGot :%+v", el2, got)
	}
	close(exit)
	// Should exit
}

func TestSessionNewSession(t *testing.T) {
	t.Parallel()

	var s *Session
	var r *Request
	var want, got element.Element
	var sid string = "bo928391289sh"
	var rid, hold int = 12789247982, 8
	var wait, inactivity time.Duration = 16 * time.Second, 300 * time.Second

	// Should be able to construct new Session
	s = NewSession(sid, rid, hold, wait, inactivity)
	if s.sid != sid {
		t.Error("Session ID should be set on Session")
		t.Errorf("\nWant:%s\nGot :%s", sid, s.sid)
	}
	if s.current != rid {
		t.Error("Current request ID should be set on Session")
		t.Errorf("\nWant:%s\nGot :%s", rid, s.current)
	}
	if s.wait != wait {
		t.Error("Wait should be set on Session")
		t.Errorf("\nWant:%s\nGot :%s", wait, s.wait)
	}
	if s.inactivity != inactivity {
		t.Error("Inactivity should be set on Session")
		t.Errorf("\nWant:%s\nGot :%s", inactivity, s.inactivity)
	}
	// Goroutines should be running
	want = element.New("foo")
	r = &Request{
		rid:     12789247982,
		body:    Body{Children: []element.Element{want}},
		proceed: make(chan struct{}),
	}
	select {
	case s.processor <- r:
	case <-time.After(2 * time.Second):
		t.Error("process goroutine is not running")
	}
	select {
	case got = <-s.elements:
	case <-time.After(2 * time.Second):
		t.Error("buffer goroutine is not running")
	}
	if !reflect.DeepEqual(want, got) {
		t.Error("Recieved an unexpected element")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
	select {
	case s.responder <- got:
	case <-time.After(2 * time.Second):
		t.Error("response goroutine is not running")
	}
}

func TestSessionresponse(t *testing.T) {
	t.Parallel()

	var s *Session
	var payload []element.Element
	var r1, r2 *Request
	var queue chan *Request
	var exit chan struct{}
	var responder chan element.Element
	var proceed chan struct{}

	// Should exit on exit channel close
	exit = make(chan struct{})
	responder = make(chan element.Element)
	queue = make(chan *Request)
	s = &Session{exit: exit, responder: responder}
	close(exit)
	timer := func() chan struct{} {
		c := make(chan struct{})
		go func() {
			s.response(queue)
			c <- struct{}{}
		}()
		return c
	}()
	select {
	case <-timer:
	case <-time.After(2 * time.Second):
		t.Error("response should return on exit channel close")
	}

	// should queue response elements
	// should get a request
	responder = make(chan element.Element, 3)
	queue = make(chan *Request, 1)
	exit = make(chan struct{})
	proceed = make(chan struct{})
	responder <- element.New("foo")
	responder <- element.New("bar")
	responder <- element.New("baz")
	r1 = &Request{proceed: proceed}
	queue <- r1
	s = &Session{exit: exit, responder: responder}
	go s.response(queue)
	select {
	case <-r1.proceed:
	case <-time.After(2 * time.Second):
		t.Error("Should queue response elements and write to a request")
	}
	payload = []element.Element{element.New("foo"), element.New("bar"), element.New("baz")}
	if !reflect.DeepEqual(r1.payload, payload) {
		t.Error("Should queue response elements and write to a request")
		t.Errorf("\nGot :%+v\nWant:%+v", r1.payload, payload)
	}

	// should queue a max of 10 resonse elements
	// should get requests until an open one is found
	responder = make(chan element.Element, 11)
	exit = make(chan struct{})
	queue = make(chan *Request, 2)
	proceed = make(chan struct{})
	payload = []element.Element{
		element.New("foo"), element.New("bar"), element.New("baz"),
		element.New("qux"), element.New("quux"), element.New("foobar"),
		element.New("foobaz"), element.New("fooqux"), element.New("fooquux"),
		element.New("barbar"),
	}
	for _, el := range payload {
		responder <- el
	}
	responder <- element.New("foobarbaz")
	r1 = &Request{spent: true}
	r2 = &Request{proceed: proceed}
	queue <- r1
	queue <- r2
	s = &Session{exit: exit, responder: responder}
	go s.response(queue)
	select {
	case <-r2.proceed:
	case <-time.After(3 * time.Second):
		t.Error("Should queue max of 10 response elements and write to an open request")
	}
	if len(r1.payload) != 0 {
		t.Error("Should write to an empty request")
		t.Errorf("Got %+v, Wanted empty slice", r1.payload)
	}
	if !reflect.DeepEqual(r2.payload, payload) {
		t.Error("Should queue response elements")
		t.Error("Should write to an open request")
		t.Errorf("\nGot :%+v\nWant:%+v", r2.payload, payload)
	}
}
