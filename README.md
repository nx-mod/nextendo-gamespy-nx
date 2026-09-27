# nextendo-gamespy-nx

The **GameSpy / Nintendo Wi-Fi Connection (NWFC)** backend for [Nextendo Network](https://nextendo.network). Source only. Not affiliated with Nintendo or GameSpy.

## Why

GameSpy powered Nintendo Wi-Fi Connection on the **DS and Wii** (it shut down with GameSpy in 2014). Native Switch titles don't use it — but the emulated **Wii/DS** games in the nx-mod stack (`wii-nx`, `gc-nx`, `wiiconnect-nx`) do. Bringing those online, the way [Wiimmfi](https://wiimmfi.de) / [AltWFC](https://github.com/polaris-/dwc_network_server_emulator) do, means answering these services:

| service | transport | role |
|---|---|---|
| **NAS** | HTTPS `nas.nintendowifi.net/ac`,`/pr` | authentication → token + challenge |
| **GP** | TCP 29900 | GameSpy Presence login (challenge/response over the token) |
| **SB** | TCP 28910 | server browser / master list |
| **QR** | UDP 27900 | query & reporting, availability |
| **NN** | UDP 27901 | NAT negotiation (P2P) |

## Status

- **NAS is implemented and tested** (`nas.go`): the `acctcreate` / `login` / `SVCLOC` actions, the base64‑`*` codec, and the token + challenge a game then uses for GameSpy. This is the entry point every NWFC game hits first.
- **The GameSpy TCP/UDP services accept and log their traffic** (a capture surface) but are not yet implemented — they need the GameSpy crypto (enctypeX / the GP challenge). See NOTES.md; reference AltWFC and OpenSpy.

## Run

```sh
go build -o server .   # Go 1.23+, stdlib only
go test ./...
./server               # NAS on :8475 (HTTPS via sni-router); GameSpy ports 29900/28910/27900/27901
```

DNS/sni-router must send `nas.nintendowifi.net` (and the `*.gamespy.com` / `*.nintendowifi.net` GameSpy hosts) here, and the emulated console must trust the stack — exactly the redirect Wiimmfi/AltWFC use.

## Credits

- **[Nextendo Network](https://nextendo.network)** — the stack, and the Wii/GC emulation (`wii-nx`, `wiiconnect-nx`) this serves.
- **[AltWFC / dwc_network_server_emulator](https://github.com/polaris-/dwc_network_server_emulator)** and **[Wiimmfi](https://wiimmfi.de)** — the NWFC NAS + GameSpy protocol documentation.
- **[OpenSpy](https://github.com/openspy)** — GameSpy service reference.

Protocol facts were read and reimplemented; no code was copied.
