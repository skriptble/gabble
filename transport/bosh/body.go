package bosh

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/namespace"
)

type Body struct {
	// BOSH
	To      string
	From    string
	Lang    string
	Ver     Version
	Wait    time.Duration
	Hold    int
	Ack     int
	Content string
	RID     int

	// Since Hold can be 0 set this to true if Hold was actually set and
	// should be added as an attribute to the element returned from
	// TransformElement.
	HoldSet bool

	// XMPP
	XMPPVer      Version
	RestartLogic bool
	Restart      bool

	SID        string
	Requests   int
	Polling    time.Duration
	Inactivity time.Duration
	Accept     string
	MaxPause   time.Duration

	Children []element.Element
}

func (b Body) TransformElement() (el element.Element) {
	var xmppNS, streamNS bool
	el = body
	if b.To != "" {
		el = el.AddAttr("to", b.To)
	}
	if b.From != "" {
		el = el.AddAttr("from", b.From)
	}
	if b.Lang != "" {
		el = el.AddAttr("xml:lang", b.Lang)
	}
	if b.Ver != (Version{}) {
		el = el.AddAttr("ver", fmt.Sprintf("%d.%d", b.Ver.Major, b.Ver.Minor))
	}
	if b.Wait != time.Duration(0) {
		el = el.AddAttr("wait", fmt.Sprintf("%d", b.Wait/time.Second))
	}

	if b.XMPPVer != (Version{}) {
		el = el.AddAttr("xmpp:version",
			fmt.Sprintf("%d.%d", b.XMPPVer.Major, b.XMPPVer.Minor))
		xmppNS = true
	}

	if b.RestartLogic == true {
		el = el.AddAttr("xmpp:restartlogic", "true")
		xmppNS = true
	}

	if b.Restart == true {
		el = el.AddAttr("xmpp:restart", "true")
		xmppNS = true
	}

	if b.HoldSet {
		el = el.AddAttr("hold", strconv.Itoa(b.Hold))
	}

	if b.Ack != 0 {
		el = el.AddAttr("ack", strconv.Itoa(b.Ack))
	}

	if b.Content != "" {
		el = el.AddAttr("content", b.Content)
	}

	if b.RID != 0 {
		el = el.AddAttr("rid", strconv.Itoa(b.RID))
	}

	if b.SID != "" {
		el = el.AddAttr("sid", b.SID)
	}

	if b.Requests != 0 {
		el = el.AddAttr("requests", strconv.Itoa(b.Requests))
	}

	if b.Polling != time.Duration(0) {
		el = el.AddAttr("polling", fmt.Sprintf("%d", b.Polling/time.Second))
	}

	if b.Inactivity != time.Duration(0) {
		el = el.AddAttr("inactivity", fmt.Sprintf("%d", b.Inactivity/time.Second))
	}

	if b.Accept != "" {
		el = el.AddAttr("accept", b.Accept)
	}

	if b.MaxPause != time.Duration(0) {
		el = el.AddAttr("maxpause", fmt.Sprintf("%d", b.MaxPause/time.Second))
	}

	for _, child := range b.Children {
		el = el.AddChild(child)
		if child.Space == "stream" {
			streamNS = true
		}
	}

	if xmppNS == true {
		el = el.AddAttr("xmlns:xmpp", namespace.XMPP)
	}

	if streamNS == true {
		el = el.AddAttr("xmlns:stream", namespace.Stream)
	}
	return
}

type BodyTransformer struct {
	dflt    Body
	lang    string
	version Version
	wait    time.Duration
	hold    int
	xmpp    Version
	content string
}

func NewBodyTransformer(dflt Body) BodyTransformer {
	return BodyTransformer{dflt: dflt}
}

func (bt BodyTransformer) TransformBody(el element.Element) (b Body) {
	b.To = el.SelectAttrValue("to", "")
	b.From = el.SelectAttrValue("from", "")
	b.Lang = el.SelectAttrValue("xml:lang", bt.dflt.Lang)
	b.Accept = el.SelectAttrValue("accept", bt.dflt.Accept)
	b.Ver = bt.parseVersion(el.SelectAttrValue("ver", ""))
	b.Wait = bt.parseWait(el.SelectAttrValue("wait", ""))
	b.Polling = bt.parsePolling(el.SelectAttrValue("polling", ""))
	b.Inactivity = bt.parseInactivity(el.SelectAttrValue("inactivity", ""))
	b.MaxPause = bt.parseMaxPause(el.SelectAttrValue("maxpause", ""))
	b.Hold = bt.parseHold(el.SelectAttrValue("hold", ""))
	b.Requests = bt.parseRequests(el.SelectAttrValue("requests", ""))
	if str := el.SelectAttrValue("ack", ""); str != "" {
		ack, err := strconv.Atoi(str)
		if err == nil {
			b.Ack = ack
		}
	}
	b.Content = el.SelectAttrValue("content", bt.dflt.Content)
	b.SID = el.SelectAttrValue("sid", "")
	if rid, err := strconv.Atoi(el.SelectAttrValue("rid", "")); err == nil {
		b.RID = rid
	}
	b.XMPPVer = bt.parseXMPPVersion(el.SelectAttrValue("xmpp:version", ""))
	if el.SelectAttrValue("xmpp:restart", "false") == "true" {
		b.Restart = true
	}
	if el.SelectAttrValue("xmpp:restartlogic", "false") == "true" {
		b.RestartLogic = true
	}
	for _, child := range el.ChildElements() {
		b.Children = append(b.Children, child)
	}
	return
}

func (bt BodyTransformer) parseVersion(str string) Version {
	idx := strings.Index(str, ".")
	if idx == -1 {
		return bt.dflt.Ver
	}
	major, err := strconv.Atoi(str[:idx])
	if err != nil {
		return bt.dflt.Ver
	}
	minor, err := strconv.Atoi(str[idx+1:])
	if err != nil {
		return bt.dflt.Ver
	}
	return Version{
		Major: major,
		Minor: minor,
	}
}

func (bt BodyTransformer) parseXMPPVersion(str string) Version {
	idx := strings.Index(str, ".")
	if idx == -1 {
		return bt.dflt.XMPPVer
	}
	major, err := strconv.Atoi(str[:idx])
	if err != nil {
		return bt.dflt.XMPPVer
	}
	minor, err := strconv.Atoi(str[idx+1:])
	if err != nil {
		return bt.dflt.XMPPVer
	}
	return Version{
		Major: major,
		Minor: minor,
	}
}

func (bt BodyTransformer) parseWait(str string) time.Duration {
	seconds, err := strconv.Atoi(str)
	if err != nil {
		return bt.dflt.Wait
	}

	return time.Duration(seconds) * time.Second
}

func (bt BodyTransformer) parsePolling(str string) time.Duration {
	seconds, err := strconv.Atoi(str)
	if err != nil {
		return bt.dflt.Polling
	}

	return time.Duration(seconds) * time.Second
}

func (bt BodyTransformer) parseInactivity(str string) time.Duration {
	seconds, err := strconv.Atoi(str)
	if err != nil {
		return bt.dflt.Inactivity
	}

	return time.Duration(seconds) * time.Second
}

func (bt BodyTransformer) parseMaxPause(str string) time.Duration {
	seconds, err := strconv.Atoi(str)
	if err != nil {
		return bt.dflt.MaxPause
	}

	return time.Duration(seconds) * time.Second
}

func (bt BodyTransformer) parseHold(str string) int {
	if str == "" {
		return -1
	}
	hold, err := strconv.Atoi(str)
	if err != nil {
		return bt.dflt.Hold
	}

	return hold
}

func (bt BodyTransformer) parseRequests(str string) int {
	requests, err := strconv.Atoi(str)
	if err != nil {
		return bt.dflt.Requests
	}

	return requests
}
