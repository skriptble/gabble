package bosh

import (
	"bytes"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/skriptble/nine/element"
)

func TestRequestWrite(t *testing.T) {
	t.Parallel()

	var err, gotErr error
	r := NewRequest(1, time.Second, "bosh", Body{}, Body{}, func() int { return 0 })
	r.spent = true
	// Return ErrRequestClosed if request is spent
	err = ErrRequestClosed
	gotErr = r.Write()
	if !reflect.DeepEqual(err, gotErr) {
		t.Error("Should return ErrRequestClosed if request is spent")
		t.Errorf("\nWant:%+v\nGot :%+v", err, gotErr)
	}
	// Return nil, set the payload, and close the proceed chan
	r = NewRequest(1, time.Second, "bosh", Body{}, Body{}, func() int { return 0 })
	err = nil
	gotErr = r.Write(element.New("foo"))
	if !reflect.DeepEqual(err, gotErr) {
		t.Error("Error from Write should be nil")
		t.Errorf("\nWant:%+v\nGot :%+v", err, gotErr)
	}
	payload := []element.Element{element.New("foo")}
	if !reflect.DeepEqual(payload, r.payload) {
		t.Error("Written elements should be added to the payload of the Request")
		t.Errorf("\nWant:%+v\nGot :%+v", payload, r.payload)
	}
	if !r.spent {
		t.Error("The Request should be spent")
	}
	select {
	case <-r.proceed:
	default:
		t.Error("The proceed channel should be closed on successful Write")
	}
}

func TestRequestClose(t *testing.T) {
	t.Parallel()

	r := NewRequest(1, time.Second, "bosh", Body{}, Body{}, func() int { return 0 })
	r.Close()
	select {
	case <-r.closed:
	default:
		t.Error("The error channel should be closed when Close is called")
	}
}

func TestRequestRID(t *testing.T) {
	t.Parallel()
	r := Request{rid: 12345}
	want := 12345
	got := r.RID()
	if want != got {
		t.Error("Call to RID should return the request id")
		t.Errorf("\nWant:%d\nGot :%d", want, got)
	}
}

func TestRequestElements(t *testing.T) {
	t.Parallel()
	// Elements returns the children on the body of the Request
	b := Body{Children: []element.Element{element.New("foo"), element.New("bar")}}
	r := Request{body: b}
	want := []element.Element{element.New("foo"), element.New("bar")}
	got := r.Elements()
	if !reflect.DeepEqual(want, got) {
		t.Error("Elements should return the children of the body of the Request")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
}

func TestRequestHandle(t *testing.T) {
	t.Parallel()

	var r *Request
	var rid int
	var buf bytes.Buffer

	// On channel closed:
	// Should set spent to true
	rid = 52934
	r = NewRequest(1, time.Second, "bosh", Body{}, Body{}, func() int { return rid })
	close(r.closed)
	r.Handle(&buf)
	if !r.spent {
		t.Error("The Request should be spent")
	}

	// On timeout
	// Should set spent to true
	rid = 293849
	r = NewRequest(1, time.Millisecond, "bosh", Body{}, Body{}, func() int { return rid })
	buf.Reset()
	r.Handle(&buf)
	if !r.spent {
		t.Error("The Request should be spent")
	}

	// Should add payload to the response's children
	// Should write response to given writer
	rid = 8298479802
	r = NewRequest(1, 5*time.Second, "bosh", Body{}, Body{}, func() int { return rid })
	payload := []element.Element{element.New("foo"), element.New("bar")}
	r.payload = payload
	close(r.proceed)
	buf.Reset()
	r.Handle(&buf)
	if r.response.Ack != rid {
		t.Error("Should set Ack if the ack has not been set")
		t.Errorf("\nWant:%d\nGot :%d", rid, r.response.Ack)
	}
	if !reflect.DeepEqual(r.response.Children, payload) {
		t.Error("Should add payload to the response's children")
		t.Errorf("\nWant:%+v\nGot :%+v", payload, r.response.Children)
	}
	got := buf.Bytes()
	want := body.
		AddAttr("ack", strconv.Itoa(rid)).
		AddChild(payload[0]).
		AddChild(payload[1]).
		WriteBytes()

	if !reflect.DeepEqual(want, got) {
		t.Error("Should write response to the given io.Writer")
		t.Errorf("\nWant:%s\nGot :%s", want, got)
	}
}
