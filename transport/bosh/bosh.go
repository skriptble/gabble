package bosh

import (
	"bytes"
	"crypto/rand"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/skriptble/nine/element"
)

var ErrMalformedXML = errors.New("malformed xml received")

var maxWait = 45 * time.Second
var maxRequests = 2
var maxPolling = 5 * time.Second
var maxInactivity = 75 * time.Second
var maxHold = 3
var ver = Version{Major: 1, Minor: 6}
var xmppver = Version{Major: 1, Minor: 0}
var restartLogic = true
var maxPause = 120 * time.Second
var lang = "en"
var content = "text/xml; charset=utf-8"
var server = "localhost"

type Version struct {
	Major, Minor int
}

// Compare takes a version and returns the version with the lower version
// number.
func (v Version) Compare(o Version) Version {
	if v.Major < o.Major {
		return v
	}
	if v.Major > o.Major {
		return o
	}

	if v.Minor < o.Minor {
		return v
	}

	return o

}

type Handler struct {
	r Register
}

// NewHandler creates a new Handler and returns it
func NewHandler(r Register) *Handler {
	h := new(Handler)
	h.r = r
	return h
}

// ServeHTTP implements http.Handler. This serves as the entrypoint for all
// BOSH traffic.
//
// This method will create the required Session, Transport, and Stream for a
// new session.
//
// This method also creates a Request object which is then processed by a
// Session. This method only returns once the Request's Handle method returns.
func (h *Handler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	var err error
	var el element.Element

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	buf := bytes.NewBuffer(b)
	dec := xml.NewDecoder(buf)
	token, err := dec.RawToken()
	if err != nil {
		log.Println(err)
		http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	switch elem := token.(type) {
	case xml.StartElement:
		if elem.Name.Local != "body" {
			b := body.
				AddAttr("type", "terminate").
				AddAttr("condition", "bad-request").
				WriteBytes()
			rw.Write(b)
			log.Println("Not a body element")
			return
		}
		el, err = h.createElement(elem, dec)
		if err != nil {
			b := body.
				AddAttr("type", "terminate").
				AddAttr("condition", "bad-request").
				WriteBytes()
			rw.Write(b)
			log.Println("Couldn't create the element")
			log.Println(err)
			return
		}
	default:
		b := body.
			AddAttr("type", "terminate").
			AddAttr("condition", "bad-request").
			WriteBytes()
		rw.Write(b)
		log.Println("Malformed XML")
		return
	}
	fmt.Println(el)
	bdy := TransformBody(el)
	if bdy.RID == 0 {
		b := body.
			AddAttr("type", "terminate").
			AddAttr("condition", "bad-request").
			WriteBytes()
		rw.Write(b)
		return
	}
	// If there is no session id, create a new session and stream, run the
	// stream, and write a bosh session creation response
	// 	- Handle version matching for xmpp and bosh here
	//  - Handle handling of route here. Make sure it matches the server we can
	//	  route to
	var rsp Body
	if bdy.SID == "" {
		rsp.SID = h.sessionID()
		rsp.Wait = bdy.Wait
		if maxWait < bdy.Wait {
			rsp.Wait = maxWait
		}

		rsp.Requests = bdy.Hold + 1
		if bdy.Hold+1 > maxRequests {
			rsp.Requests = maxRequests
		}

		rsp.Ver = bdy.Ver.Compare(ver)
		rsp.Polling = maxPolling
		rsp.Inactivity = maxInactivity

		rsp.Hold = bdy.Hold
		if bdy.Hold > maxHold || bdy.Hold == -1 {
			rsp.Hold = maxHold
		}

		rsp.To = server
		rsp.Ack = bdy.RID
		rsp.MaxPause = maxPause

		rsp.RestartLogic = restartLogic
		rsp.XMPPVer = bdy.XMPPVer.Compare(xmppver)

		log.Println("Creating session.")
		s := NewSession(rsp.SID, bdy.RID, rsp.Hold, rsp.Wait, rsp.Inactivity)
		h.r.Add(rsp.SID, s)
		req := NewRequest(bdy.RID, rsp.Wait, rsp.SID, bdy, rsp, s.UnregisterRequest(rsp.RID))
		err = s.Process(req)
		if err != nil {
			b := body.
				AddAttr("type", "terminate").
				AddAttr("condition", "internal-server-error").
				WriteBytes()
			rw.Write(b)
			return
		}
		log.Printf("%s", rsp.TransformElement())

		req.Handle(rw)
		return
	}

	// If there is a session id, lookup the session id in the register
	// If a session does not exist for the session id, return a session not
	// found error.
	s, err := h.r.Lookup(bdy.SID)
	if err != nil {
		b := body.
			AddAttr("type", "terminate").
			AddAttr("condition", "item-not-found").
			WriteBytes()
		rw.Write(b)
		return
	}
	// Transform the body element into a Body and invoke the process method
	// of the stream with the Request.
	// Invoke the Handle method of the request.
	req := NewRequest(bdy.RID, s.Wait(), bdy.SID, bdy, rsp, s.UnregisterRequest(bdy.RID))
	log.Printf("Request to be processed: %+v", req)
	err = s.Process(req)
	if err != nil {
		b := body.
			AddAttr("type", "terminate").
			AddAttr("condition", "internal-server-error").
			WriteBytes()
		rw.Write(b)
		return
	}

	req.Handle(rw)
}

func (h *Handler) createElement(start xml.StartElement, dec *xml.Decoder) (el element.Element, err error) {
	var children []element.Token

	el = element.Element{
		Space: start.Name.Space,
		Tag:   start.Name.Local,
	}
	for _, attr := range start.Attr {
		el.Attr = append(
			el.Attr,
			element.Attr{
				Space: attr.Name.Space,
				Key:   attr.Name.Local,
				Value: attr.Value,
			},
		)

		if el.Space == "" && attr.Name.Space == "" && attr.Name.Local == "xmlns" {
			el.Space = attr.Value
		}

		if attr.Name.Space == "xmlns" && el.Space == attr.Name.Local {
			el.Space = attr.Value
		}
	}

	children, err = h.childElements(dec)
	el.Child = children
	return
}

func (h *Handler) childElements(dec *xml.Decoder) (children []element.Token, err error) {
	var token xml.Token
	var el element.Element
	for {
		token, err = dec.RawToken()
		if err != nil {
			return
		}

		switch elem := token.(type) {
		case xml.StartElement:
			el, err = h.createElement(elem, dec)
			if err != nil {
				return
			}
			children = append(children, el)
		case xml.EndElement:
			return
		case xml.CharData:
			data := string(elem)
			children = append(children, element.CharData{Data: data})
		}
	}
}

// sessionID generates a unique ID for the session.
func (h *Handler) sessionID() string {
	id := make([]byte, 16)
	rand.Read(id)

	id[8] = (id[8] | 0x80) & 0xBF
	id[6] = (id[6] | 0x40) & 0x4F

	return fmt.Sprintf("bo%xsh", id)
}
