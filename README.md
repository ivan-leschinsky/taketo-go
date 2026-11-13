# taketo-go ![Version](https://img.shields.io/github/v/tag/ivan-leschinsky/taketo-go?label=version)
![Go version](https://img.shields.io/badge/go-1.24-blue)
[![Unit Tests](https://github.com/ivan-leschinsky/taketo-go/actions/workflows/test.yml/badge.svg)](https://github.com/ivan-leschinsky/taketo-go/actions/workflows/test.yml)

**[taketo.vano.dev](https://taketo.vano.dev)** · Ever get tired of remembering which user, host, and port to use for each of your servers? `taketo-go` lets you define all your servers once in a simple YAML file and connect to any of them with a short alias — no more hunting through notes or `.ssh/config`.

It's especially handy when you work across multiple projects and environments (staging, production, etc.), each with their own set of servers. Define defaults at the project or environment level, and individual servers inherit what they need.

## Quick start

```sh
# 1. Install
brew install ivan-leschinsky/taps/taketo-go

# 2. Create your config
nano ~/.taketo.yml

# 3. See all your servers
taketo-go ls

# 4. Connect
taketo-go <alias>
```

**Pro tip:** Add a short shell alias so you type even less:

```sh
# add to ~/.zshrc or ~/.bashrc
alias to='taketo-go'
```

Then `to ls`, `to prod-web`, `to stg-wkr -c "tail -f log/production.log"` — much faster.

## Usage

```sh
taketo-go ls                           # list all configured servers
taketo-go list                         # same as ls

taketo-go <alias>                      # connect by short alias
taketo-go <project:server>             # connect by project path
taketo-go <project:environment:server> # connect in a specific environment

taketo-go <alias> -c "docker ps -a"   # run a one-off command instead of default

taketo-go aliases                      # print every alias/path (used for completion)
taketo-go completion <zsh|bash|fish>   # print a shell completion script
taketo-go --version                    # show version info
```

If you pass an alias that doesn't exist (or a typo), taketo-go opens an
interactive fuzzy picker pre-filled with what you typed — just narrow it down and
hit enter instead of re-running the command.

## Config examples

The config lives at `~/.taketo.yml`. Here's a minimal example and a more complete one.

### Minimal — a single server

```yaml
projects:
- name: myproject
  servers:
  - name: web
    alias: web
    host: 1.2.3.4
    user: ubuntu
```

Connect with: `taketo-go web`

### With environments and shared defaults

```yaml
projects:
- name: myproject
  defaults:
    user: deploy
    shell: bash

  servers:
  - name: bastion
    alias: bas
    host: bastion.example.com

  environments:
  - name: staging
    defaults:
      location: /var/www/app
    servers:
    - name: web
      alias: stg-web
      host: stg-web.example.com
    - name: worker
      alias: stg-wkr
      host: stg-worker.example.com

  - name: production
    defaults:
      location: /var/www/app
    servers:
    - name: web
      alias: prod-web
      host: prod-web.example.com
      port: "2222"
    - name: worker
      alias: prod-wkr
      host: prod-worker.example.com
      env:
      - RAILS_ENV=production
      - LOG_LEVEL=warn
```

Connect with:
- `taketo-go bas` — bastion
- `taketo-go stg-web` — staging web (or `taketo-go myproject:staging:web`)
- `taketo-go prod-wkr` — production worker

Run `taketo-go ls` to see all servers in a tree view:

```
  myproject
  ├── bastion              [bas]           deploy@bastion.example.com
  ├── staging
  │   ├── web              [stg-web]       deploy@stg-web.example.com
  │   └── worker           [stg-wkr]       deploy@stg-worker.example.com
  └── production
      ├── web              [prod-web]      deploy@prod-web.example.com:2222
      └── worker           [prod-wkr]      deploy@prod-worker.example.com
```

### Available server fields

| Field             | Description                                                              |
|-------------------|--------------------------------------------------------------------------|
| `name`            | Human-readable name (defaults to the first alias; doubles as alias if none) |
| `alias`           | Short unique identifier for quick access                                 |
| `aliases`         | List of extra aliases pointing at the same server                        |
| `host`            | Hostname or IP address                                                   |
| `hosts`           | List of hosts to try in order (first reachable wins) — see fallback      |
| `user`            | SSH username                                                             |
| `port`            | SSH port (default: 22)                                                   |
| `shell`           | Shell to open (`bash`, `zsh`, etc.)                                      |
| `location`        | Working directory to `cd` into on connect                               |
| `command`         | Default command to run on the server (a bare name looks up `commands:`)  |
| `env`             | List of `KEY=value` environment variables to set                        |
| `ssh_env`         | `KEY=value` list sent by ssh itself (`-o SetEnv`), e.g. `TERM=xterm-256color` |
| `identity`        | Path to an SSH private key (`ssh -i`)                                    |
| `forward`         | List of `local:host:remote` port forwards (`ssh -L`)                    |
| `ssh_options`     | Extra raw flags passed straight to `ssh` (e.g. `-o`, `-J`, ciphers)     |
| `connect_timeout` | Seconds to wait per host when using `hosts` fallback (default: 2)        |
| `targets`         | Extra entry points on the same server — see below                        |

`defaults` (at project or environment level) supports every field above except
`name`, `alias`/`aliases`, and `targets`.

Environments are optional — you can list `servers:` directly under a `project`
(with its own `defaults`) and skip the extra nesting level entirely.

### Fewer names

`name` and `alias` fill each other in, so one is enough:

```yaml
projects:
- name: home
  defaults: { user: root }
  servers:
  - name: nvr              # `taketo-go nvr`, path home:nvr
    host: 100.75.112.42
  - aliases: [syno, s]     # name = syno
    host: 192.168.1.235
```

A server without `alias`/`aliases` answers to its `name`, unless an earlier
server already took it (repeated names like `web` per environment stay
reachable by path). Targets work the same way.

### Multiple aliases

```yaml
- name: web
  aliases: [web, w, prod-web]   # any of these connect to the same server
  host: 1.2.3.4
```

### Targets — one server, many entry points

Instead of copy-pasting a server just to `cd` into a different folder or run a
different command, define `targets`. Each target inherits everything from the
parent server and overrides only what it needs:

```yaml
- name: vm
  alias: vm
  host: 1.2.3.4
  user: root
  shell: fish -l
  targets:
    - { alias: app,     location: /srv/app }
    - { alias: app-log, location: /srv/app, command: ./log }
    - { aliases: [worker, wkr], location: /srv/worker }
```

`taketo-go app-log` connects to the same box, `cd`s into `/srv/app`, and runs
`./log`. Targets support `alias` and `aliases`, and override any server field.

### Host fallback

Give a server several addresses and taketo-go connects to the first one that
answers (a quick TCP probe with `connect_timeout`, default 2s):

```yaml
- name: valid
  alias: valid
  hosts: ["10.0.0.5", "valid.internal.example.com"]
  connect_timeout: 2
```

### Custom ssh flags, keys and tunnels

```yaml
- name: old-switch
  alias: sw
  host: 192.168.0.10
  identity: ~/.ssh/id_legacy
  forward: ["8080:localhost:80"]
  ssh_options: ["-o", "KexAlgorithms=+diffie-hellman-group1-sha1"]
```

### Reusable named commands

Define commands once at the top level and reference them by name from any
server's `command:` field:

```yaml
commands:
  - name: log
    command: tail -f log/production.log

projects:
  - name: myapp
    servers:
      - name: web
        alias: web
        host: 1.2.3.4
        command: log        # expands to "tail -f log/production.log"
```

### Shell completion

taketo-go can complete aliases and project paths. Add the matching line to your
shell config (pass your shell alias as a second arg so completion works for it too):

```sh
# zsh (~/.zshrc)
eval "$(taketo-go completion zsh)"
eval "$(taketo-go completion zsh to)"     # if you aliased `to=taketo-go`

# bash (~/.bashrc)
eval "$(taketo-go completion bash)"

# fish (~/.config/fish/config.fish)
taketo-go completion fish | source
```

## Install

More details and examples at **[taketo.vano.dev](https://taketo.vano.dev)**.

### Homebrew (macOS)

```sh
brew install ivan-leschinsky/taps/taketo-go

# if it can't find it, tap first:
brew tap ivan-leschinsky/taps
brew install ivan-leschinsky/taps/taketo-go
```

### From source

```sh
go mod download
go install
```

### Download a release

Go to [releases](https://github.com/ivan-leschinsky/taketo-go/releases/latest) and grab the binary for your platform.

## Development

```sh
go mod download
go run . ls             # list servers from ~/.taketo.yml
go run . <alias>        # connect to a server
go test -v              # run unit tests
```
