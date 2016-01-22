package bosh

import (
	"reflect"
	"testing"
	"time"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/namespace"
)

func TestBodyTransformElement(t *testing.T) {
	t.Parallel()
	// Adds proper attributes
	body1 := Body{
		To:           "foo@bar",
		From:         "baz@quux",
		Lang:         "en-gb",
		Ver:          Version{Major: 1, Minor: 4},
		Wait:         5 * time.Second,
		Hold:         14,
		Ack:          1,
		Content:      "application/xml; charset=utf-8",
		RID:          619727392817,
		XMPPVer:      Version{Major: 2, Minor: 0},
		RestartLogic: true,
		Restart:      true,
		SID:          "bo12345sh",
		Requests:     7,
		Polling:      3 * time.Second,
		Inactivity:   37 * time.Second,
		Accept:       "deflate,gzip",
		MaxPause:     93 * time.Second,
		Children:     []element.Element{element.New("message")},
		HoldSet:      true,
	}
	want := body.
		AddAttr("to", "foo@bar").
		AddAttr("from", "baz@quux").
		AddAttr("xml:lang", "en-gb").
		AddAttr("ver", "1.4").
		AddAttr("wait", "5").
		AddAttr("xmpp:version", "2.0").
		AddAttr("xmpp:restartlogic", "true").
		AddAttr("xmpp:restart", "true").
		AddAttr("hold", "14").
		AddAttr("ack", "1").
		AddAttr("content", "application/xml; charset=utf-8").
		AddAttr("rid", "619727392817").
		AddAttr("sid", "bo12345sh").
		AddAttr("requests", "7").
		AddAttr("polling", "3").
		AddAttr("inactivity", "37").
		AddAttr("accept", "deflate,gzip").
		AddAttr("maxpause", "93").
		AddAttr("xmlns:xmpp", namespace.XMPP).
		AddChild(element.New("message"))
	got := body1.TransformElement()
	if !reflect.DeepEqual(want, got) {
		t.Error("Should properly transform body into an element.Element")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}

	// Adds the stream namespace if a child element has a stream namespace
	body1.Children = append(body1.Children, element.New("stream:error"))
	want = want.AddChild(element.New("stream:error")).AddAttr("xmlns:stream", namespace.Stream)
	got = body1.TransformElement()
	if !reflect.DeepEqual(want, got) {
		t.Error("Should add stream namespace if a child is in the stream namespace")
		t.Errorf("\nWant:%+v\nGot :%+v", want, got)
	}
}

func TestBodyTransformer(t *testing.T) {
	t.Parallel()
	elem1 := element.New("body").
		AddAttr("to", "foo@bar").
		AddAttr("from", "baz@quux").
		AddAttr("xml:lang", "en-gb").
		AddAttr("ver", "1.4").
		AddAttr("wait", "5").
		AddAttr("hold", "14").
		AddAttr("ack", "1").
		AddAttr("content", "application/xml; charset=utf-8").
		AddAttr("sid", "bo12345sh").
		AddAttr("rid", "619727392817").
		AddAttr("xmpp:version", "2.0").
		AddAttr("xmpp:restartlogic", "true").
		AddAttr("xmpp:restart", "true").
		AddAttr("requests", "7").
		AddAttr("polling", "3").
		AddAttr("inactivity", "37").
		AddAttr("accept", "deflate,gzip").
		AddAttr("maxpause", "93").
		AddChild(element.New("message"))
	elem2 := element.New("body")
	body1 := Body{
		To:           "foo@bar",
		From:         "baz@quux",
		Lang:         "en-gb",
		Ver:          Version{Major: 1, Minor: 4},
		Wait:         5 * time.Second,
		Hold:         14,
		Ack:          1,
		Content:      "application/xml; charset=utf-8",
		RID:          619727392817,
		XMPPVer:      Version{Major: 2, Minor: 0},
		RestartLogic: true,
		Restart:      true,
		SID:          "bo12345sh",
		Requests:     7,
		Polling:      3 * time.Second,
		Inactivity:   37 * time.Second,
		Accept:       "deflate,gzip",
		MaxPause:     93 * time.Second,
		Children:     []element.Element{element.New("message")},
	}
	body2 := Body{
		Lang:       "en-us",
		Ver:        Version{Major: 1, Minor: 6},
		Wait:       45 * time.Second,
		Hold:       1,
		Ack:        1,
		Content:    "text/xml; charset=utf-8",
		XMPPVer:    Version{Major: 1, Minor: 0},
		Requests:   2,
		Polling:    5 * time.Second,
		Inactivity: 75 * time.Second,
		MaxPause:   120 * time.Second,
	}
	bt := NewBodyTransformer(body2)
	got1 := bt.TransformBody(elem1)
	got2 := bt.TransformBody(elem2)

	body2.Hold = -1
	body2.Ack = 0
	if !reflect.DeepEqual(body1, got1) {
		t.Error("TransformBody should set attributes from the body element")
		t.Errorf("\nWant:%+v\nGot :%+v\n", body1, got1)
	}

	if !reflect.DeepEqual(body2, got2) {
		t.Error("TransformBody should use the defaults")
		t.Errorf("\nWant:%+v\nGot :%+v\n", body2, got2)
	}
}
