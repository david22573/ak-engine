# ak-engine
An engine for ak related extensions

## Development

Required Go version: `1.25.6`.

Standalone setup from a fresh clone:

```bash
git clone git@github.com:david22573/ak-engine.git
cd ak-engine
GOWORK=off go mod download
make verify
```

The first dependency download needs network access unless the module cache is already populated. After dependencies are available, `make verify` runs vet, test, and build checks with `GOWORK=off`.

`go.work` is optional for local multi-repository development. `ak-engine` does not import `ak-rif`; the boundary in `internal/rifbridge` only parses serialized artifacts.
