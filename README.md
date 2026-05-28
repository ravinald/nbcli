# nbcli

Modern CLI + TUI for Netbox.

`nbcli show <resource>` for quick or machine-readable queries. `nbcli tui` for a full-screen browser that mirrors the Netbox web UI.

## Status

Working surface:

- `nbcli show sites [keyword value]...` — list DCIM sites
- `nbcli show tenants [keyword value]...` — list tenants
- `nbcli show contacts [keyword value]...` — list contacts
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
```

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

cobra generates completion for bash, zsh, fish, and powershell. Positional keywords (`status active region us-west …`) get completed too.

```sh
# bash (macOS via homebrew)
nbcli completion bash > $(brew --prefix)/etc/bash_completion.d/nbcli

# zsh
nbcli completion zsh > "${fpath[1]}/_nbcli"

# fish
nbcli completion fish > ~/.config/fish/completions/nbcli.fish
```

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

TBD.
