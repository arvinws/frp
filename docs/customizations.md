# Custom governance patches

This fork adds a lightweight client governance layer for `frps` with minimal core hooks.

## Added packages

- `pkg/ext/banlist`: in-memory client disable/enable rules keyed by `clientID`.
- `pkg/ext/clientmgr`: online session index (`clientID` <-> `runID`) and disconnect helper.
- `pkg/ext/audit`: management action audit recorder.
- `pkg/ext/adminapi`: admin handlers for disable, enable, disconnect, and disable-and-disconnect.

## Core hook anchors

- `server/service.go`
  - login-time disabled-client check (`RegisterControl`).
  - session registration for online client index.
- `server/control.go`
  - new-proxy denied for disabled client (`handleNewProxy`).
  - session index cleanup on control close (`worker`).
- `server/api_router.go`
  - admin route registration under `/api/admin/...`.

## Admin APIs

- `POST /api/admin/clients/{clientID}/disable`
- `POST /api/admin/clients/{clientID}/enable`
- `POST /api/admin/sessions/{runID}/disconnect`
- `POST /api/admin/clients/{clientID}/disable-and-disconnect`

All endpoints support optional JSON payload:

```json
{
  "reason": "string",
  "operator": "string"
}
```

## Fork CI workflows

- `fork-auto-package.yml`: runs on every push and uploads `release/packages/*` as Actions artifacts.
- `fork-auto-release.yml`: runs on tag push (`v*`) and publishes a GitHub Release with all files in `release/packages/*`.

Example release trigger:

```bash
git tag v0.67.0-arvin.1
git push origin v0.67.0-arvin.1
```
