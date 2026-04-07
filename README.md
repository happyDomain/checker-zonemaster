# checker-zonemaster

Zonemaster DNS validation checker for [happyDomain](https://www.happydomain.org/).

Runs the [Zonemaster](https://zonemaster.net/) test suite against a domain via
its public JSON-RPC API and stores the full results as an observation. The
checker also produces a rich HTML report grouped by Zonemaster module and
severity.

## Usage

### Standalone HTTP server

```bash
make
./checker-zonemaster -listen :8080
```

The server exposes the standard happyDomain external checker endpoints
(`/health`, `/definition`, `/collect`, `/evaluate`, `/html-report`).

### Docker

```bash
make docker
docker run -p 8080:8080 happydomain/checker-zonemaster
```

### happyDomain plugin

```bash
make plugin
# produces checker-zonemaster.so, loadable by happyDomain as a Go plugin
```

The plugin exposes a `NewCheckerPlugin` symbol returning the checker
definition and observation provider, which happyDomain registers in its
global registries at load time.

### Versioning

The binary, plugin, and Docker image embed a version string overridable
at build time:

```bash
make CHECKER_VERSION=1.2.3
make plugin CHECKER_VERSION=1.2.3
make docker CHECKER_VERSION=1.2.3
```

### happyDomain remote endpoint

Set the `endpoint` admin option for the zonemaster checker to the URL of
the running checker-zonemaster server (e.g.,
`http://checker-zonemaster:8080`). happyDomain will delegate observation
collection to this endpoint.

## Options

| Scope     | Id                 | Description                                          |
| --------- | ------------------ | ---------------------------------------------------- |
| Run       | `domainName`       | Domain name to test (auto-filled from the domain)    |
| Run       | `profile`          | Zonemaster profile name (default: `default`)         |
| User      | `language`         | Result language (`en`, `fr`, `de`, …)                |
| Admin     | `zonemasterAPIURL` | Zonemaster JSON-RPC endpoint (default: official API) |

## Protocol

### POST /collect

Request:
```json
{
  "key": "zonemaster",
  "options": {
    "domainName": "example.com",
    "zonemasterAPIURL": "https://zonemaster.net/api",
    "language": "en",
    "profile": "default"
  }
}
```

The collect call is long-running: it starts a Zonemaster test, polls until
completion, and returns the full result tree as the observation payload.

## License

This project is licensed under the **MIT License** (see `LICENSE`). The
third-party Apache-2.0 attributions for `checker-sdk-go` are recorded in
`NOTICE` and must accompany any binary or source redistribution of this
project.
