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

### Deployment

The `/collect` endpoint has no built-in authentication and will issue
JSON-RPC calls to whatever Zonemaster API URL is configured via the
`zonemasterAPIURL` admin option (defaulting to the official public API
at `https://zonemaster.net/api`). Operators should point this option
only at trusted Zonemaster instances; pointing it at an untrusted host
turns the checker into an SSRF vector, since responses are parsed and
surfaced back to the caller. The checker itself is meant to run on a
trusted network, reachable only by the happyDomain instance that drives
it. Restrict access via a reverse proxy with authentication, a network
ACL, or by binding the listener to a private interface; do not expose
it directly to the public internet.

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
