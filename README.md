# taketo-go ![Version](https://img.shields.io/github/v/tag/ivan-leschinsky/taketo-go?label=version)
![Go version](https://img.shields.io/badge/go-1.17-lightblue)
![Go version](https://img.shields.io/badge/go-1.18-blue)
[![Unit Tests](https://github.com/ivan-leschinsky/taketo-go/actions/workflows/test.yml/badge.svg)](https://github.com/ivan-leschinsky/taketo-go/actions/workflows/test.yml)

Simplified version of https://github.com/ivan-leschinsky/taketo ruby gem written in go


### Install with homebrew on macOS

```sh
brew install ivan-leschinsky/taps/taketo-go

# if it cannot see - tap manually
brew tap ivan-leschinsky/taps
brew install ivan-leschinsky/taps/taketo-go
```

### Install from source

```sh
go mod download
go install
```

### Download from releases
Go to the releases and download version for your platform: https://github.com/ivan-leschinsky/taketo-go/releases/latest

bin will be available here:
`$GOPATH/bin/taketo-go`


### Run in development:

```sh
go mod download
go run . server_alias_here
```

### Run unit tests

```
go test -v
```
