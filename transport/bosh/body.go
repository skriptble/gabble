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

	if b.Hold != -1 {
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

func TransformBody(el element.Element) (b Body) {
	b.To = el.SelectAttrValue("to", "")
	b.Lang = el.SelectAttrValue("xml:lang", lang)
	b.Ver = parseVersion(el.SelectAttrValue("ver", ""), ver)
	b.Wait = parseWait(el.SelectAttrValue("wait", ""), maxWait)
	b.Hold = parseHold(el.SelectAttrValue("hold", ""), maxHold)
	if str := el.SelectAttrValue("ack", ""); str != "" {
		ack, err := strconv.Atoi(str)
		if err == nil {
			b.Ack = ack
		}
	}
	b.Content = el.SelectAttrValue("content", content)
	b.SID = el.SelectAttrValue("sid", "")
	if rid, err := strconv.Atoi(el.SelectAttrValue("rid", "")); err == nil {
		b.RID = rid
	}
	b.XMPPVer = parseVersion(el.SelectAttrValue("xmpp:version", ""), xmppver)
	if el.SelectAttrValue("xmpp:restart", "false") == "true" {
		b.Restart = true
	}
	for _, child := range el.ChildElements() {
		b.Children = append(b.Children, child)
	}
	return
}

func parseVersion(str string, dflt Version) Version {
	idx := strings.Index(str, ".")
	if idx == -1 {
		return dflt
	}
	major, err := strconv.Atoi(str[:idx])
	if err != nil {
		return dflt
	}
	minor, err := strconv.Atoi(str[idx+1:])
	if err != nil {
		return dflt
	}
	return Version{
		Major: major,
		Minor: minor,
	}
}

func parseWait(str string, dflt time.Duration) time.Duration {
	seconds, err := strconv.Atoi(str)
	if err != nil {
		return dflt
	}

	return time.Duration(seconds) * time.Second
}

func parseHold(str string, dflt int) int {
	if str == "" {
		return -1
	}
	hold, err := strconv.Atoi(str)
	if err != nil {
		return dflt
	}

	return hold
}
