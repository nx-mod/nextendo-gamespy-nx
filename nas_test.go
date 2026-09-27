package main

import (
	"net/url"
	"strings"
	"testing"
)

func TestNASBase64StarCodec(t *testing.T) {
	// base64 that would pad with '=' must come back as '*'
	enc := nasEncode([]byte("Mario"))
	if strings.Contains(enc, "=") {
		t.Fatalf("padding not replaced: %q", enc)
	}
	if string(nasDecode(enc)) != "Mario" {
		t.Fatalf("round trip: %q", nasDecode(enc))
	}
}

// buildNASBody encodes a request the way a console would.
func buildNASBody(fields map[string]string) string {
	v := url.Values{}
	for k, val := range fields {
		v.Set(k, nasEncode([]byte(val)))
	}
	return v.Encode()
}

func TestNASLoginReturnsTokenAndChallenge(t *testing.T) {
	s := newNASStore()
	body := buildNASBody(map[string]string{"action": "login", "gamecd": "ADAE", "userid": "1234567890123"})
	f := parseNASForm(body)
	if f["action"] != "login" || f["gamecd"] != "ADAE" {
		t.Fatalf("decoded form %+v", f)
	}
	resp := s.handle(f)
	if resp["returncd"] != "001" || resp["token"] == "" || len(resp["challenge"]) != 8 || resp["locator"] != "gamespy.com" {
		t.Fatalf("login resp %+v", resp)
	}

	// the response must be NAS-encoded and decode back
	encoded := encodeNASResponse(resp)
	back := parseNASForm(encoded)
	if back["token"] != resp["token"] || back["challenge"] != resp["challenge"] {
		t.Fatalf("response round trip: %+v vs %+v", back, resp)
	}
}

func TestNASAcctCreateAndSvcloc(t *testing.T) {
	s := newNASStore()
	create := s.handle(map[string]string{"action": "acctcreate"})
	if create["returncd"] != "002" || len(create["userid"]) != 13 {
		t.Fatalf("acctcreate %+v", create)
	}
	svc := s.handle(map[string]string{"action": "SVCLOC", "svc": "9000"})
	if svc["returncd"] != "007" || svc["servicetoken"] == "" {
		t.Fatalf("svcloc %+v", svc)
	}
	if s.logins != 0 || s.creates != 1 {
		t.Fatalf("counters logins=%d creates=%d", s.logins, s.creates)
	}
}

func TestNASUnknownAction(t *testing.T) {
	s := newNASStore()
	if r := s.handle(map[string]string{"action": "wat"}); r["returncd"] != "109" {
		t.Fatalf("unknown action returncd %s", r["returncd"])
	}
}
