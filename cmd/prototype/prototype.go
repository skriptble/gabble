package main

import (
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/skriptble/gabble/transport/bosh"
	"github.com/skriptble/nine/bind"
	"github.com/skriptble/nine/element/stanza"
	"github.com/skriptble/nine/namespace"
	"github.com/skriptble/nine/sasl"
	"github.com/skriptble/nine/stream"
)

var dflt = bosh.Body{
	Wait:         45 * time.Second,
	Requests:     2,
	Polling:      5 * time.Second,
	Inactivity:   75 * time.Second,
	Hold:         3,
	HoldSet:      true,
	Ver:          bosh.Version{Major: 1, Minor: 6},
	XMPPVer:      bosh.Version{Major: 1, Minor: 0},
	RestartLogic: true,
	MaxPause:     120 * time.Second,
	Lang:         "en",
	Content:      "text/xml; charset=utf8",
}
var server = "localhost"

func init() {
	// turn on debugging
	stream.Trace.SetOutput(os.Stderr)
	stream.Debug.SetOutput(os.Stderr)
}

func main() {
	reg := NewRegister()
	bt := bosh.NewBodyTransformer(bosh.Body{})
	handler := bosh.NewHandler(reg, bt, dflt, server)
	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("."))))
	srv := &http.Server{
		Addr:    ":8088",
		Handler: mux,
	}
	log.Fatal(srv.ListenAndServe())
}

type register struct {
	sessions map[string]*bosh.Session

	sync.RWMutex
}

// NewRegister returns a new initalized Register.
func NewRegister() bosh.Register {
	r := new(register)
	r.sessions = make(map[string]*bosh.Session)
	return r
}

// Add adds a session to the Register.
func (r *register) Add(sid string, s *bosh.Session) {
	r.Lock()
	defer r.Unlock()
	// create a new transport
	tp := bosh.NewTransport(stream.Receiving, s)
	runStream(tp)
	// create ta new stream
	r.sessions[sid] = s
}

// Remove removes a session from the Register.
func (r *register) Remove(sid string) {
	r.Lock()
	defer r.Unlock()
	delete(r.sessions, sid)
}

// Lookup returns the Session associated with the given sid. If the session
// doesn't exist, ErrSessionNotFound is returned.
func (r *register) Lookup(sid string) (s *bosh.Session, err error) {
	r.RLock()
	s, ok := r.sessions[sid]
	r.RUnlock()
	if !ok {
		err = bosh.ErrSessionNotFound
		return
	}
	if s.Expired() {
		r.Remove(sid)
		err = bosh.ErrSessionNotFound
		s = nil
	}
	return
}

func runStream(tp stream.Transport) {
	saslHandler := sasl.NewHandler(map[string]sasl.Mechanism{
		"PLAIN": sasl.NewPlainMechanism(sasl.FakePlain{}),
	})
	bindHandler := bind.NewHandler()
	sessionHandler := bind.NewSessionHandler()
	iqHandler := stream.NewIQMux().
		Handle(namespace.Bind, "bind", string(stanza.IQSet), bindHandler).
		Handle(namespace.Session, "session", string(stanza.IQSet), sessionHandler)

	if iqHandler.Err() != nil {
		log.Fatal(iqHandler.Err())
	}

	elHandler := stream.NewElementMux().
		Handle(namespace.SASL, "auth", saslHandler).
		Handle(namespace.SASL, "response", saslHandler).
		Handle(namespace.Client, "iq", iqHandler).
		Handle(namespace.Client, "presence", stream.Blackhole{}).
		Handle(namespace.Client, "message", stream.Blackhole{})

	if elHandler.Err() != nil {
		log.Fatal(iqHandler.Err())
	}

	fhs := []stream.FeatureGenerator{
		saslHandler,
		bindHandler,
		// sessionHandler,
	}
	props := stream.NewProperties()
	props.Domain = "localhost"
	s := stream.New(tp, elHandler, stream.Receiving).
		AddFeatureHandlers(fhs...).
		SetProperties(props)
	go s.Run()
}
