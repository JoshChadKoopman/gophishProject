package middleware

import (
	"encoding/gob"

	"github.com/gophish/gophish/models"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

// init registers the necessary models to be saved in the session later
func init() {
	gob.Register(&models.User{})
	gob.Register(&models.Flash{})
	Store.Options.HttpOnly = true
	// This sets the maxAge to 24 hours for all cookies
	Store.MaxAge(86400)
}

// mustGenerateRandomKey wraps securecookie.GenerateRandomKey and panics on
// failure. A nil return from GenerateRandomKey indicates the system CSPRNG is
// unavailable, which is a fatal condition for any security-sensitive process.
func mustGenerateRandomKey(length int) []byte {
	key := securecookie.GenerateRandomKey(length)
	if key == nil {
		panic("middleware: failed to generate random session key — system CSPRNG may be unavailable")
	}
	return key
}

// Store contains the session information for the request
var Store = sessions.NewCookieStore(
	mustGenerateRandomKey(64), // Signing key
	mustGenerateRandomKey(32)) // Encryption key
