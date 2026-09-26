package web

// Flash messages in a signed cookie (Phoenix's put_flash + cookie session).

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
)

const flashCookie = "_pinchflat_flash"

// PutFlash sets a flash message shown on the next rendered page (put_flash).
func (s *Server) PutFlash(w http.ResponseWriter, r *http.Request, kind, msg string) {
	page := PageOf(r.Context())
	page.Flash[kind] = msg
	b, _ := json.Marshal(page.Flash)
	http.SetCookie(w, &http.Cookie{
		Name: flashCookie, Value: s.sign(b), Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
}

// takeFlash reads and clears the flash cookie.
func (s *Server) takeFlash(w http.ResponseWriter, r *http.Request) map[string]string {
	out := map[string]string{}
	c, err := r.Cookie(flashCookie)
	if err != nil {
		return out
	}
	http.SetCookie(w, &http.Cookie{Name: flashCookie, Value: "", Path: "/", MaxAge: -1})
	if b, ok := s.verify(c.Value); ok {
		_ = json.Unmarshal(b, &out)
	}
	return out
}

func (s *Server) sign(b []byte) string {
	mac := hmac.New(sha256.New, []byte(s.Opts.SecretKeyBase))
	mac.Write(b)
	return base64.RawURLEncoding.EncodeToString(b) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Server) verify(v string) ([]byte, bool) {
	payload, sig, ok := strings.Cut(v, ".")
	if !ok {
		return nil, false
	}
	b, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, false
	}
	return b, hmac.Equal([]byte(s.sign(b)), []byte(v)) && sig != ""
}
