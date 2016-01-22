package bosh

import "errors"

// TODO: Register should be changed into an interface. This interface will be
// responsible for creating the stream from the added sessions and closing the
// stream from the removed sessions. This allows a very clean separation
// between the BOSH layer and the stream. It also allows the BOSH Stream
// transport to be potentially used for clients, although the current request
// implementation is heavily geared toward the receiver. It would not take too
// much effort to add the functionality for initating.

// ErrSessionNotFound is the error returned when Lookup is called with a sid
// that does not have a corresponding session in the Register.
var ErrSessionNotFound = errors.New("session not found")

// A Register handles the logic and interactions outside of the BOSH layer. A
// register handles the creation of streams when a session is added to the
// register and the closing of streams when a session is removed. This allows
// for more sophisticated stream creation strategies, such as opening a
// connection to an XMPP server. This package provides a stream transport
// designed to work with a Session.
type Register interface {
	Add(sid string, s *Session)
	Remove(sid string)
	// Lookup finds a session by its session ID. Lookup should not return any
	// session which has expired.
	Lookup(sid string) (*Session, error)
}
