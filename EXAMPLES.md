# nextendo-gamespy-nx — example usage

`./run.sh` starts the NWFC/GameSpy backend (NAS on :8475; GameSpy TCP/UDP ports
capture-only). NAS is the auth step the emulated Wii/DS titles use to go online.

```sh
./run.sh &
./test-nas.sh           # posts a NAS login -> returncd=001 + token + challenge
```

The GameSpy GP/SB/QR/NatNeg services accept and log traffic (capture surface)
until the per-game exchange is mapped — see NOTES.md.
