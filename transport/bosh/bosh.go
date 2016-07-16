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

	"github.com/skriptble/nine/element"
)

var ErrMalformedXML = errors.New("malformed xml received")

type Handler struct {
	r      Register
	bt     BodyTransformer
	dflt   Body
	server string
}

// NewHandler creates a new Handler and returns it
func NewHandler(r Register, bt BodyTransformer, dflt Body, server string) *Handler {
	h := new(Handler)
	h.r = r
	h.bt = bt
	h.dflt = dflt
	h.server = server
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

	if r.Method != "POST" {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

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
		// TODO(skriptble): This should close the stream.
		log.Println(err)
		http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	switch elem := token.(type) {
	case xml.StartElement:
		if elem.Name.Local != "body" {
			b := BadRequest.WriteBytes()
			rw.Write(b)
			log.Println("Not a body element")
			return
		}
		el, err = h.createElement(elem, dec)
		if err != nil {
			b := BadRequest.WriteBytes()
			rw.Write(b)
			log.Println("Couldn't create the element")
			log.Println(err)
			return
		}
	default:
		b := BadRequest.WriteBytes()
		rw.Write(b)
		log.Println("Malformed XML")
		return
	}
	fmt.Println(el)
	bdy := h.bt.TransformBody(el)
	if bdy.RID == 0 {
		b := BadRequest.WriteBytes()
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
		rsp = h.negotiate(bdy)
		log.Println("Creating session.")
		s := NewSession(rsp.SID, bdy.RID, rsp.Hold, rsp.Wait, rsp.Inactivity)
		h.r.Add(rsp.SID, s)
		req := NewRequest(bdy.RID, rsp.Wait, rsp.SID, bdy, rsp, s.UnregisterRequest())
		err = s.Process(req)
		if err != nil {
			b := BadRequest.WriteBytes()
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
		b := BadRequest.WriteBytes()
		rw.Write(b)
		return
	}
	// Transform the body element into a Body and invoke the process method
	// of the stream with the Request.
	// Invoke the Handle method of the request.
	req := NewRequest(bdy.RID, s.Wait(), bdy.SID, bdy, rsp, s.UnregisterRequest())
	log.Printf("Request to be processed: %+v", req)
	err = s.Process(req)
	if err != nil {
		b := BadRequest.WriteBytes()
		rw.Write(b)
		return
	}

	req.Handle(rw)
}

func (h *Handler) negotiate(bdy Body) (rsp Body) {
	var dflt = h.dflt
	rsp.SID = h.sessionID()
	rsp.Wait = bdy.Wait
	if dflt.Wait < bdy.Wait {
		rsp.Wait = dflt.Wait
	}

	rsp.Requests = bdy.Hold + 1
	if bdy.Hold+1 > dflt.Requests {
		rsp.Requests = dflt.Requests
	}

	rsp.Ver = bdy.Ver.Compare(dflt.Ver)
	rsp.Polling = dflt.Polling
	rsp.Inactivity = dflt.Inactivity

	rsp.Hold = bdy.Hold
	if bdy.Hold > dflt.Hold && dflt.HoldSet {
		rsp.Hold = dflt.Hold
	}
	rsp.HoldSet = true

	rsp.To = h.server
	rsp.Ack = bdy.RID
	rsp.MaxPause = dflt.MaxPause

	rsp.RestartLogic = dflt.RestartLogic
	rsp.XMPPVer = bdy.XMPPVer.Compare(dflt.XMPPVer)
	return
}

func (h *Handler) createElement(start xml.StartElement, dec *xml.Decoder) (el element.Element, err error) {
	ns := make(map[string]string)
	return h.childElementsHelper(start, dec, ns)
}

func (h *Handler) childElementsHelper(start xml.StartElement, dec *xml.Decoder, ns map[string]string) (el element.Element, err error) {
	var children []element.Token

	el = element.Element{
		Space:      start.Name.Space,
		Tag:        start.Name.Local,
		Namespaces: ns,
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
			el.Namespaces[""] = attr.Value
		}

		if attr.Name.Space == "xmlns" && el.Space == attr.Name.Local {
			el.Namespaces[attr.Name.Local] = attr.Value
		}
	}

	nns := make(map[string]string)
	for k, v := range el.Namespaces {
		nns[k] = v
	}
	children, err = h.childElements(dec, nns)
	el.Child = children
	return
}

func (h *Handler) childElements(dec *xml.Decoder, ns map[string]string) (children []element.Token, err error) {
	var token xml.Token
	var el element.Element
	for {
		token, err = dec.RawToken()
		if err != nil {
			return
		}

		switch elem := token.(type) {
		case xml.StartElement:
			el, err = h.childElementsHelper(elem, dec, ns)
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
