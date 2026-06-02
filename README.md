# nbcli

[![ci](https://github.com/ravinald/nbcli/actions/workflows/ci.yml/badge.svg)](https://github.com/ravinald/nbcli/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ravinald/nbcli)](https://goreportcard.com/report/github.com/ravinald/nbcli)
[![Go Reference](https://pkg.go.dev/badge/github.com/ravinald/nbcli.svg)](https://pkg.go.dev/github.com/ravinald/nbcli)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Modern CLI + TUI for Netbox.

`nbcli show <resource>` for quick or machine-readable queries. `nbcli tui` for a full-screen browser that mirrors the Netbox web UI.

## Status

Working surface:

- `nbcli show sites [keyword value]...` — list DCIM sites
- `nbcli show tenants [keyword value]...` — list tenants
- `nbcli show contacts [keyword value]...` — list contacts
- `nbcli search [all|<module>] <key> [limit value] [pager]` — free-text search (per-module via `?q=`, `all` via Netbox's GraphQL endpoint)
- `nbcli tui` — bubbletea shell; **Tenants** and **Contacts** items render live tables; other items are placeholders
- `nbcli plugin passthrough <name> <subpath> [key value ...]` — raw forward to `/api/plugins/<name>/...`
- `nbcli plugin list` — show compiled-in named plugins
- `nbcli version [--json]`

Every `show` command takes `limit 0` to auto-paginate every page (capped at 200 pages × 100 items as a safety belt; tune in code). For `--format json|yaml|tsv`, rows stream as they arrive — memory stays O(1) and output begins immediately. `--format table` buffers because it needs all rows to align columns.

DCIM (racks, devices, interfaces), IPAM, Virtualization, and the remaining TUI views are still placeholders.

## CLI shape: positional, not flags

API filters are **positional keyword/value pairs**, Junos-style. Flags are reserved for operational concerns (`--format`, `--url`, `--config`, `--insecure`, `--timeout`, `--verbose`).

```sh
nbcli show sites
nbcli show sites status active
nbcli show sites region us-west status active limit 100
nbcli show sites name hq --format json        # operational flag still works
```

Pair order is free — `name hq status active` and `status active name hq` are equivalent. Unknown keywords fail loudly with the allowed set in the error:

```
$ nbcli show sites foo bar
Error: unknown keyword "foo" (expected one of: limit, name, offset, region, slug, status, tenant)
```

The parser lives in [`internal/cmdutils/positional_args.go`](internal/cmdutils/positional_args.go). Add a new resource command by declaring its `[]cmdutils.KeywordSpec` and handing it to `Validator()`, `UsageLine()`, `HelpTable()`, and `CompletionFunc()`.

## Search

```sh
nbcli search sites hq                 # one resource, ?q=hq
nbcli search ip-addresses 10.0.0      # IPAM, ?q=10.0.0
nbcli search vrfs prod limit 200      # explicit page size
nbcli search all hq                   # cross-resource via /api/search/
nbcli search all hq pager             # interactive pager
```

`search <module> <key>` uses the same column set as `show <module>` (REST `?q=`). `search all <key>` batches a single GraphQL query against `/api/graphql/` covering every Netbox resource type and renders results in a four-column view:

```
TYPE              FIELD         VALUE         DISPLAY
dcim.site         name          hq            HQ
dcim.device       comments      hq backup     edge-1
ipam.ipaddress    description   hq mgmt       10.0.0.1/24
```

## Install

```sh
make build       # ./bin/nbcli
make install     # $GOPATH/bin/nbcli
```

## Configure

Precedence (highest wins):

1. flag (e.g. `--url`, `--format`)
2. env: `NBCLI_*` (e.g. `NBCLI_URL`, `NBCLI_FORMAT`)
3. config file: `$XDG_CONFIG_HOME/nbcli/config.yaml` or `~/.config/nbcli/config.yaml`
4. built-in defaults

Minimum env:

```sh
export NBCLI_URL=https://netbox.example.com
export NETBOX_TOKEN=nbt_KEY.TOKEN     # NBCLI_TOKEN also works
```

### Token sources (precedence, highest wins)

1. Process env: `NBCLI_TOKEN`, then `NETBOX_TOKEN`, then composed `NETBOX_API_V2_KEY` + `NETBOX_API_V2_TOKEN`
2. `--env-file <path>`
3. `$XDG_CONFIG_HOME/nbcli/secrets.env` (or `~/.config/nbcli/secrets.env`)
4. `~/.env.netbox`

Empty values count as "not set" so a real-env override of `""` won't clobber a file value.

Example `~/.env.netbox`:

```sh
# Either form works:
NETBOX_TOKEN=nbt_KEY.TOKEN
# ...or split, nbcli will compose them with a "." separator:
NETBOX_API_V2_KEY=nbt_KEY
NETBOX_API_V2_TOKEN=TOKEN
```

Format: `KEY=value` lines, `#` comments, optional quotes, optional `export` prefix. No shell expansion — what you `cat` is what nbcli reads.

Example `config.yaml`:

```yaml
url: https://netbox.example.com
format: table          # implicit default is table on a TTY, json when piped
timeout_seconds: 30
insecure_skip_verify: false
auth_scheme: v2        # v2 (default, Bearer header) or v1 (legacy, Token header)

# Configurable columns (mirrors the web UI). Same config drives CLI tables
# and the TUI. Resource key matches the API path segment.
columns:
  sites:        [id, name, status, region, tenant]
  devices:      [id, name, type, site, rack, status, primary_ip4]
  ip-addresses: [id, address, family, status, dns_name, tenant]
```

List the names available for a resource:

```sh
nbcli columns                # all resources
nbcli columns sites          # available columns + headers + default flag
```

Per-call override (CLI only):

```sh
nbcli show devices --columns id,name,primary_ip4,serial
```

**Interactive column picker (TUI).** Press `C` from any sidebar item in `nbcli tui` to open the picker popup. Toggle columns with `space`/`x`, reorder with `K`/`J` (or `Ctrl+↑`/`Ctrl+↓`), commit with `Enter` (writes back to `config.yaml` and refreshes the table), or `Esc` to cancel.

### Token auth scheme (v1 vs v2)

Netbox supports two [token authorization styles](https://netboxlabs.com/docs/netbox/integrations/rest-api/#v1-and-v2-tokens):

| Scheme | Header sent | When |
|---|---|---|
| `v2` (default) | `Authorization: Bearer nbt_KEY.TOKEN` | Tokens created with v2 hashing (recommended) |
| `v1` | `Authorization: Token <token>` | Legacy plaintext-stored tokens |

Override per-call with `--auth-scheme v1`, in config with `auth_scheme: v1`, or in env via `NBCLI_AUTH_SCHEME=v1`. A 403 with `{"detail":"Invalid v1 token"}` means your Netbox needs v2 and the client is sending v1.

The token is intentionally **not** read from the config file — env only.

## Output formats

| Format | When |
|--------|------|
| `table` | Default on a TTY. Padded columns. |
| `json`  | Default when stdout is piped/redirected. `jq` friendly. |
| `yaml`  | Human-readable structured output. |
| `tsv`   | Headered tab-separated. Embedded tabs/newlines are stripped. |

Override per-call: `nbcli show sites --format json`.

## Plugin passthrough

Until typed plugin wrappers land, hit any plugin endpoint generically. Trailing positional pairs become URL query parameters:

```sh
nbcli plugin passthrough wireless-controllers controllers/
nbcli plugin passthrough my-plugin some/endpoint/ site hq limit 10
```

The response is rendered as JSON (or YAML if you pass `--format yaml`). `--method` stays a flag since it controls HTTP behavior, not what you're asking the API for.

## TUI keybinds

| Key | Action |
|---|---|
| `Tab` / `Shift-Tab` (or `]` / `[`) | Move between sidebar items |
| `↑` / `↓` / `k` / `j` | Move between rows in the active table |
| `Enter` | Show detail of the selected row (in list); commit filter (in search) |
| `Esc` | Close detail · cancel/clear filter |
| `/` | Open the search input (substring match across all visible columns) |
| `?` | Toggle the help overlay |
| `q` / `Ctrl-C` | Quit |

**Detail view** is reflection-based: every non-zero field of the selected resource is printed as `key: value`. `NestedRef` foreign keys collapse to `Name (#id)`; `LabelValue` enums render as their label. Foreign-key fields are annotated with `[1]`, `[2]`, … markers — press the matching digit to jump to that resource's detail view (e.g. Enter on a device → see `Site: HQ (#1) [1]` → press `1` → Sites view opens to detail of `HQ`).

**Search** filters the active table client-side as you type. Commit with `Enter` to keep the filter visible (and exit the input); cancel with `Esc` to restore all rows. While a committed filter is active, the status line shows `filter "foo" · 12/247 rows`; pressing `Esc` from the list clears it.

## Shell completion

cobra generates the completion script; `cmdutils.CompletionFunc` adds positional-keyword awareness on top. After installing, `TAB` completes:

- subcommands (`nbcli sh<TAB>` → `show`)
- resources (`nbcli show <TAB>` → `sites devices racks …`)
- positional keywords per resource, with already-typed ones filtered out (`nbcli show sites <TAB>` → `name slug status region tenant limit offset pager`)
- static value enums (`nbcli show sites status <TAB>` → `active planned staging decommissioning retired`)
- switch keywords like `pager` (advance the cursor without expecting a value)
- root flags (`--format`, `--url`, `--verbose`, …)

### Install

```sh
# bash (macOS via homebrew)
nbcli completion bash > $(brew --prefix)/etc/bash_completion.d/nbcli

# bash (Linux)
nbcli completion bash | sudo tee /etc/bash_completion.d/nbcli >/dev/null

# zsh — file must be in fpath, name must be _nbcli
nbcli completion zsh > "${fpath[1]}/_nbcli"   # site-wide
# or user-local:
mkdir -p ~/.zfunc && nbcli completion zsh > ~/.zfunc/_nbcli
# then add to ~/.zshrc: fpath+=~/.zfunc; autoload -U compinit && compinit

# fish
nbcli completion fish > ~/.config/fish/completions/nbcli.fish

# powershell
nbcli completion powershell | Out-String | Invoke-Expression   # current session
```

### Drive completion from the API (optional)

cobra also supports completion via a one-shot exec: `nbcli __complete show sites ""`. The first run primes your shell cache; subsequent `TAB`s are instant. To verify completion without installing:

```sh
$ nbcli __complete show sites ""
limit
name
offset
pager
region
slug
status
tenant
:4    # ShellCompDirectiveNoFileComp — no filename fallback
```

The `pager` line is the new switch keyword. It appears in the keyword list, takes no value, and the completer correctly hands control back to the keyword position after the user types it.

## Verbose / debug logging

`--verbose` (or `-v`) flips the slog level to DEBUG and writes structured logs to stderr. Quiet by default — set it when you want to see every Netbox request:

```sh
$ nbcli -v show sites status active
time=2026-05-27T13:50:01Z level=DEBUG msg="netbox request" method=GET url=https://nb.example.com/api/dcim/sites/?status=active
time=2026-05-27T13:50:01Z level=DEBUG msg="netbox response" method=GET url=… status=200 bytes=4321 elapsed=87ms
ID   NAME  SLUG  STATUS  REGION   TENANT
1    hq    hq    Active  us-west  acme
```

Errors flow through return values, not the logger.

## Development

```sh
make tidy        # resolve deps
make lint        # golangci-lint
make test        # go test -race
make vuln        # govulncheck
make ci          # all of the above
```

Project layout:

```
cmd/nbcli/                # main entry point
internal/cmd/             # cobra command tree
internal/cmdutils/        # Junos-style positional keyword parser + limit/offset helper
internal/config/          # viper-layered config
internal/netbox/          # hand-rolled API client + generic pagination (ListAll / Iterate)
internal/output/          # table/json/yaml/tsv renderers
internal/plugins/         # plugin registry + generic passthrough
internal/tui/             # bubbletea root shell (sidebar nav)
internal/tui/views/       # per-resource bubbletea views (Tenants, Contacts, ...)
internal/version/         # ldflags-baked build metadata
```

## License

Apache 2.0 — see [LICENSE](LICENSE). Contributions accepted under the same license; see [NOTICE](NOTICE) for attribution.
