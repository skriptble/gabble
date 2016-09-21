package bosh

import (
	"log"

	"github.com/skriptble/nine/element"
	"github.com/skriptble/nine/element/stanza"
	"github.com/skriptble/nine/stream"
)

// Transport implements a stream.Transport for BOSH. It handles a bulk of the
// state associated including holding onto the response writers and handling
// the timeouts associated with them.
type Transport struct {
	mode stream.Mode

	// restart indicates if this is a start or restart. Used in the start
	// method
	restart bool

	s *Session
}

func NewTransport(mode stream.Mode, s *Session) stream.Transport {
	t := new(Transport)
	t.mode = mode
	t.s = s
	return t
}

// Close implements io.Closer
func (t *Transport) Close() error {
	t.s.Close()
	return nil
}

// WriteElement writes the given element to the underlying Session. The Session
// handles writing the element to the approriate request. This method should
// be used for non-stanza elements, such as those used during SASL negotiation.
func (t *Transport) WriteElement(el element.Element) (err error) {
	log.Printf("Writing element: %s", el)
	err = t.s.Write(el)
	return
}

func (t *Transport) Write(p []byte) (n int, err error) {
	return
}

// WriteStanza writes the given stanza to the underlying Session. The Session
// handles writing the stanza to the appropriate request. This method should be
// used instead of transforming a stanza to an element and using WriteElement.
func (t *Transport) WriteStanza(st stanza.Stanza) error {
	el := st.TransformElement()
	return t.WriteElement(el)
}

// Next retrieves the next element from the underlying Session. This method is
// a very thin wrapper around the Session's Element method.
func (t *Transport) Next() (el element.Element, err error) {
	// TODO: This should probably catch an ErrSessionClosed and transform it
	// into an io.EOF or ErrStreamClosed error.
	return t.s.Element()
}

// Start starts or restarts the stream.
func (t *Transport) Start() (bool, error) {
	// Receiving mode
	// if p.Domain == "" {
	// 	return false, stream.ErrDomainNotSet
	// }
	if t.restart {
		// Wait for the restart from the client
		_, err := t.s.Element()
		if err != stream.ErrRequireRestart {
			log.Printf("Recieved non Restart error: %s", err)
		}
	} else {
		t.restart = true
	}
	log.Println("Sending features")
	// ftrs := element.StreamFeatures
	// for _, f := range p.Features {
	// 	ftrs = ftrs.AddChild(f)
	// }
	// err := t.WriteElement(ftrs)
	log.Println("Features sent")
	return false, nil
}
