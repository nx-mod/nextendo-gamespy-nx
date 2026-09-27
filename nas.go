package main

// NAS — the Nintendo Authentication Server half of Nintendo Wi-Fi Connection.
//
// Before a DS/Wii game reaches GameSpy it authenticates with NAS over HTTPS
// (nas.nintendowifi.net/ac, /pr). The request is a urlencoded form whose values
// are base64 with '=' replaced by '*'. NAS replies with the same encoding and,
// on login, hands back the token + challenge the game then uses for the GameSpy
// GP login. Modeled on AltWFC / dwc_network_server_emulator.
//
//	action=acctcreate  -> returncd=002, userid
//	action=login       -> returncd=001, token, challenge, locator=gamespy.com
//	action=SVCLOC      -> returncd=007, servicetoken (per-service token)
//
// This is the tractable, fully-implemented core. The GameSpy TCP/UDP services
// (GP login, server browser, NatNeg) are the remaining work — see NOTES.md.

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"sync"
	"time"
)

// nasEncode/nasDecode are base64 with '=' -> '*', NWFC's variant.
func nasEncode(b []byte) string {
	return strings.ReplaceAll(base64.StdEncoding.EncodeToString(b), "=", "*")
}
func nasDecode(s string) []byte {
	b, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(s, "*", "="))
	if err != nil {
		return nil
	}
	return b
}

// parseNASForm decodes a NAS request body into decoded string fields.
func parseNASForm(body string) map[string]string {
	out := map[string]string{}
	vals, err := url.ParseQuery(body)
	if err != nil {
		return out
	}
	for k, v := range vals {
		if len(v) > 0 {
			out[k] = string(nasDecode(v[0]))
		}
	}
	return out
}

// encodeNASResponse renders response fields in NAS form (base64 '*' values).
func encodeNASResponse(fields map[string]string) string {
	parts := make([]string, 0, len(fields))
	// datetime and returncd first is not required, but keep a stable order.
	for k, v := range fields {
		parts = append(parts, k+"="+nasEncode([]byte(v)))
	}
	return strings.Join(parts, "&")
}

// account is a minimal NWFC account.
type account struct {
	UserID    string
	Token     string
	Challenge string
	Created   time.Time
}

type nasStore struct {
	mu       sync.Mutex
	byGameCd map[string]*account // gamecd|userinput -> account
	logins   int64
	creates  int64
}

func newNASStore() *nasStore { return &nasStore{byGameCd: map[string]*account{}} }

// handle produces the response fields for a decoded NAS request.
func (s *nasStore) handle(f map[string]string) map[string]string {
	action := strings.ToLower(f["action"])
	now := nasDateTime()
	switch action {
	case "acctcreate":
		s.mu.Lock()
		s.creates++
		s.mu.Unlock()
		return map[string]string{
			"returncd": "002",
			"userid":   newUserID(),
			"datetime": now,
		}
	case "login":
		acc := s.loginAccount(f)
		return map[string]string{
			"returncd":  "001",
			"locator":   "gamespy.com",
			"challenge": acc.Challenge,
			"token":     acc.Token,
			"retry":     "0",
			"datetime":  now,
		}
	case "svcloc":
		return map[string]string{
			"returncd":     "007",
			"servicetoken": newToken(),
			"statusdata":   "Y",
			"svchost":      f["svc"] + ".available.gs.nintendowifi.net",
			"datetime":     now,
		}
	default:
		return map[string]string{"returncd": "109", "reason": "unknown action", "datetime": now}
	}
}

// loginAccount finds or creates the account for a login request and refreshes
// its token + challenge.
func (s *nasStore) loginAccount(f map[string]string) *account {
	key := f["gamecd"] + "|" + f["userid"] + "|" + f["user"]
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logins++
	acc := s.byGameCd[key]
	if acc == nil {
		acc = &account{UserID: newUserID(), Created: time.Now()}
		s.byGameCd[key] = acc
	}
	acc.Token = newToken()
	acc.Challenge = randAlnum(8)
	return acc
}

func nasDateTime() string { return time.Now().UTC().Format("200601021504") }

func newUserID() string {
	// A 13-digit numeric user id, as NWFC uses.
	var sb strings.Builder
	for i := 0; i < 13; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(10))
		sb.WriteByte(byte('0' + n.Int64()))
	}
	return sb.String()
}

func newToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return "NDS" + fmt.Sprintf("%x", b)
}

const alnum = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func randAlnum(n int) string {
	b := make([]byte, n)
	for i := range b {
		x, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alnum))))
		b[i] = alnum[x.Int64()]
	}
	return string(b)
}
