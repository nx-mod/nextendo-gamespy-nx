# nextendo-gamespy-nx - notes

## Implemented
- NAS (nas.go): /ac,/pr; base64 with '=' -> '*'; actions acctcreate (002),
  login (001 -> token+challenge+locator=gamespy.com), SVCLOC (007). Tested.

## Remaining (needs the GameSpy crypto + a capture)
1. **GP login (TCP 29900).** `\login\` with a challenge/response: the client
   proves the NAS token with an MD5-based response over the server + client
   challenges and the game's secret key. Then `\lc\2\` etc. See AltWFC gpcm.
2. **Server browser (TCP 28910).** enctypeX-encrypted server list; the key is the
   game's GameSpy secret key. Reference OpenSpy serverbrowser.
3. **QR (UDP 27900).** heartbeats + availability; servers register here.
4. **NatNeg (UDP 27901).** connect two peers behind NAT.
5. Per-game **secret keys** and gamecd/gamename mapping (a table like AltWFC's).

The GameSpy ports currently accept and LOG traffic (main.go) so the exact
per-game exchange can be captured before implementing.
