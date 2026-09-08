# nbcli

[![ci](https://github.com/ravinald/nbcli/actions/workflows/ci.yml/badge.svg)](https://github.com/ravinald/nbcli/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ravinald/nbcli)](https://goreportcard.com/report/github.com/ravinald/nbcli)
[![Go Reference](https://pkg.go.dev/badge/github.com/ravinald/nbcli.svg)](https://pkg.go.dev/github.com/ravinald/nbcli)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A modern CLI + TUI for Netbox, built for network engineers.

- **`nbcli show <resource>`** — quick or machine-readable queries
- **`nbcli search [all|<module>] <key>`** — free-text search across one or every resource
- **`nbcli tui`** — full-screen browser that mirrors the Netbox web UI
- **Junos-style positional grammar** — `nbcli show sites status active region us-west format json` instead of flag soup

## Quickstart

```sh
# install
go install github.com/ravinald/nbcli/cmd/nbcli@latest

# point it at your Netbox
export NBCLI_URL=https://netbox.example.com
export NETBOX_TOKEN=nbt_KEY.TOKEN

# go
nbcli show sites
nbcli show sites status active region us-west
nbcli show ip-addresses 10.0.0 pager       # less-like interactive pager
nbcli search all hq                        # cross-resource search
nbcli tui                                  # full-screen browser
```

Want JSON? Pipe it:

```sh
nbcli show sites limit 0 format json | jq '.[] | .name'
```

Filters are positional keyword/value pairs — Junos shape, no flag soup. Pair order is free; unknown keywords fail loudly with the allowed set named in the error.

## Documentation

Comprehensive usage guide: **[docs/usage.md](docs/usage.md)**.

Covers configuration precedence and config-file layout, token auth (v1/v2), output formats, columns (per-resource registry + interactive TUI picker), search internals, the pager, plugin passthrough, TUI keybinds, shell completion install, verbose/debug logging, and the project layout.

## Status

Working surface today:

- `show`: `sites`, `racks`, `devices`, `interfaces`, `prefixes`, `ip-addresses`, `vlans`, `vrfs`, `tenants`, `contacts`, `virtual-machines`, `clusters`
- `search`: `all` + every show resource via `?q=`
- `tui`: bubbletea shell — Tenants and Contacts render live tables; other items are placeholders
- `plugin passthrough <name> <subpath>` — raw forward to any `/api/plugins/<name>/...` endpoint
- `columns [resource]` — list available columns per resource
- `version [--json]`

Streaming for `limit 0` keeps memory O(1) on json/yaml/tsv output. The 200-page × 100-item cap is a safety belt; tune in code.

## License

Apache 2.0 — see [LICENSE](LICENSE). Contributions accepted under the same license; see [NOTICE](NOTICE) for attribution.
