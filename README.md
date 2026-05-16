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

taketo-go --version                    # show version info
```

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

| Field      | Description                                      |
|------------|--------------------------------------------------|
| `name`     | Human-readable name                              |
| `alias`    | Short unique identifier for quick access         |
| `host`     | Hostname or IP address                           |
| `user`     | SSH username                                     |
| `port`     | SSH port (default: 22)                           |
| `shell`    | Shell to open (`bash`, `zsh`, etc.)              |
| `location` | Working directory to `cd` into on connect        |
| `command`  | Default command to run on the server             |
| `env`      | List of `KEY=value` environment variables to set |

`defaults` (at project or environment level) supports: `host`, `user`, `port`, `shell`, `location`.

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
