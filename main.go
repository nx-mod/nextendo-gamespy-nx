// Command nextendo-gamespy-nx serves the GameSpy / Nintendo Wi-Fi Connection
// (NWFC) backend for Nextendo Network.
//
// GameSpy powered Nintendo Wi-Fi Connection on the DS and Wii (it shut down with
// GameSpy in 2014). It is not used by native Switch titles, but it is exactly
// what the emulated Wii/DS games in the nx-mod stack (wii-nx, gc-nx, wiiconnect-nx)
// need to go online. Bringing those titles back online, like Wiimmfi/AltWFC does,
// means answering:
//
//	NAS   (HTTPS nas.nintendowifi.net/ac,/pr)  authentication -> token + challenge
//	GP    (TCP 29900)  GameSpy Presence login (challenge/response over the token)
//	SB    (TCP 28910)  server browser / master list
//	QR    (UDP 27900)  query & reporting, availability
//	NN    (UDP 27901)  NAT negotiation for P2P
//
// NAS is implemented and tested (nas.go). The GameSpy TCP/UDP services need the
// GameSpy crypto (enctypeX / GP challenge) and are not finished; this accepts and
// LOGS their traffic so the exact per-game exchange can be mapped, the same way
// the rest of the stack is built. See NOTES.md.
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

var (
	nasPort   = envOrInt("NAS_PORT", 8475) // HTTPS NAS, behind sni-router
	certFile  = envOr("CERT_FILE", "")
	keyFile   = envOr("KEY_FILE", "")
	dashPort  = envOr("DASH_PORT", "8104")
	dashToken = envOr("DASH_TOKEN", "")

	nas       = newNASStore()
	gsConns   atomic.Int64
	dashStart = time.Now()
)

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func envOrInt(k string, d int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return d
}

func main() {
	log.SetOutput(os.Stdout)
	go startDashboard()

	// GameSpy service ports: accept + log (capture surface until implemented).
	go listenGameSpyTCP(29900, "GP-login")
	go listenGameSpyTCP(28910, "server-browser")
	go listenGameSpyUDP(27900, "qr/availability")
	go listenGameSpyUDP(27901, "natneg")

	mux := http.NewServeMux()
	mux.HandleFunc("/ac", handleNAS)
	mux.HandleFunc("/pr", handleNAS)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "ok") })

	addr := fmt.Sprintf(":%d", nasPort)
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 15 * time.Second}
	if certFile != "" && keyFile != "" {
		log.Printf("[GameSpy NAS] listening HTTPS %s", addr)
		log.Fatal(srv.ListenAndServeTLS(certFile, keyFile))
	}
	log.Printf("[GameSpy NAS] listening HTTP %s (TLS via sni-router)", addr)
	log.Fatal(srv.ListenAndServe())
}

// handleNAS answers a NAS /ac or /pr request.
func handleNAS(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	r.Body.Close()
	fields := parseNASForm(string(body))
	resp := nas.handle(fields)
	log.Printf("[GameSpy NAS] %s action=%q gamecd=%q -> returncd=%s", r.URL.Path, fields["action"], fields["gamecd"], resp["returncd"])

	// NWFC expects the response over HTTP with this content type; the body is the
	// NAS form (base64 '*' values).
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("X-Organization", "Nintendo")
	w.Header().Set("Server", "Nintendo Wii (http)")
	_, _ = w.Write([]byte(encodeNASResponse(resp)))
}

// listenGameSpyTCP accepts connections on a GameSpy TCP port and logs the first
// bytes of each, a capture surface for the unfinished GameSpy protocols.
func listenGameSpyTCP(port int, name string) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Printf("[GameSpy %s] tcp %d: %v", name, port, err)
		return
	}
	log.Printf("[GameSpy %s] listening TCP %d (capture only)", name, port)
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		gsConns.Add(1)
		go func(c net.Conn) {
			defer c.Close()
			_ = c.SetReadDeadline(time.Now().Add(30 * time.Second))
			buf := make([]byte, 512)
			n, _ := c.Read(buf)
			if n > 0 {
				log.Printf("[GameSpy %s] from %s (%d bytes): %q", name, c.RemoteAddr(), n, sample(buf[:n]))
			}
		}(c)
	}
}

func listenGameSpyUDP(port int, name string) {
	pc, err := net.ListenPacket("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Printf("[GameSpy %s] udp %d: %v", name, port, err)
		return
	}
	log.Printf("[GameSpy %s] listening UDP %d (capture only)", name, port)
	buf := make([]byte, 2048)
	for {
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			return
		}
		if n > 0 {
			gsConns.Add(1)
			log.Printf("[GameSpy %s] from %s (%d bytes): %s", name, addr, n, hex.EncodeToString(buf[:min(n, 64)]))
		}
	}
}

// sample returns a printable-ish preview of a GameSpy packet (they are
// backslash-delimited ASCII like \login\\challenge\...).
func sample(b []byte) string {
	if len(b) > 200 {
		b = b[:200]
	}
	out := make([]byte, len(b))
	for i, c := range b {
		if c >= 0x20 && c < 0x7f {
			out[i] = c
		} else {
			out[i] = '.'
		}
	}
	return string(out)
}

func startDashboard() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		if dashToken != "" && r.URL.Query().Get("key") != dashToken {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		nas.mu.Lock()
		logins, creates, accts := nas.logins, nas.creates, len(nas.byGameCd)
		nas.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uptimeSeconds": int(time.Since(dashStart).Seconds()),
			"nasLogins":     logins, "nasAcctCreates": creates, "accounts": accts,
			"gamespyConnections": gsConns.Load(), "stack": "gamespy",
		})
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "ok") })
	log.Printf("[GameSpy Dashboard] :%s", dashPort)
	if err := http.ListenAndServe(":"+dashPort, mux); err != nil {
		log.Printf("[GameSpy Dashboard] %v", err)
	}
}
