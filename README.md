# konfig

`konfig` is a small, bootstrapped helper for loading application configuration in Go. It supports JSON, TOML, and YAML files with deterministic precedence and environmental overrides that are ergonomic enough to make shipping 12-factor apps a breeze. 

- JSON/TOML/YAML parsing with automatic extension discovery
- Environment overrides with tag support, prefixes, nested structs, and pointer allocation
- Multiple file merging so later files override earlier definitions
- Zero global state, every call operates on the struct you pass in

---

## Installation

```bash
go get github.com/moehandi/konfig
```

The module targets Go 1.25 or newer.

## Quick Start

```go
package main

import (
    "log"

    "github.com/moehandi/konfig"
)

// Declere your configuration with regular Go structs
// and optional struct tags for env variable overrides.
type DB struct {
    Name string `json:"name"`
    Port int    `env:"DB_PORT"`
}

type Config struct {
    Server   string
    Port     int
    Debug    bool
    Database DB
}

func main() {
    var cfg Config

    if err := konfig.GetConf("config/app", &cfg); err != nil {
        log.Fatalf("load config: %v", err)
    }

    log.Printf("server %s:%d (debug=%v)", cfg.Server, cfg.Port, cfg.Debug)
}
```

`GetConf("config/app", &cfg)` tries, in order:

1. `config/app.json`
2. `config/app.toml`
3. `config/app.yaml` or `config/app.yml`
4. Environment variables (if supplied)

If no files or environment variables populate the struct, the call returns `konfig.ErrNoSources` so you can react accordingly.

## Usage Patterns

### 1. Single configuration file (extension optional)

```go
var cfg Config
if err := konfig.LoadConfigFileNoExt(&cfg, "configs/service"); err != nil {
    // hendle error
}
```

### 2. Layerred configuration files

Later files override earlier ones, which is ideal for overlaying environment specific settings.

```go
err := konfig.Load(
    &cfg,
    konfig.WithFiles(
        "/etc/myapp/defaults.yaml",
        "/etc/myapp/overrides.toml",
        "./config/demo.json",
    ),
)
```

### 3. Environment overrides with prefixes and tags

```go
err := konfig.Load(
    &cfg,
    konfig.WithEnvPrefix("MYAPP"), // yields MYAPP_SERVER, MYAPP_DATABASE_PORT, ...
    konfig.WithFiles("./config/base.yaml"),
)
```

Environment keys are generated in this order of precedence:

1. `env:"CUSTOM_KEY"`
2. `konfig:"custom"`
3. `json`, `yaml`, or `toml` tags
4. Struct field name (converted to upper snake case)

Nested structs inherit the prefix (`APP_DATABASE_PORT`), and pointer-to-struct fields are allocated automatically when a value exists.

### 4. Helper functions

- `konfig.LoadJSON`, `konfig.LoadTOML`, `konfig.LoadYAML` decode a specific format
- `konfig.GetConfigFilesWithExt` filters a list to the files that actually exist, preserving order
- `konfig.ErrNoSources` signals that no file or environment populated the struct

## Examples

The `example/` directory contains runnable scenarios:

| Example | Highlights |
| --- | --- |
| `example/basic` | Basic `GetConf` usage with format precedence |
| `example/multi` | Layered overrides with `WithFiles` |
| `example/env` | Environment-variable driven configuration with prefixes |

Run any example with:

```bash
go run ./example/basic
go run ./example/multi
go run ./example/env
```

## Development Workflow

### Build

```bash
go build ./...
```

### Test & Coverage

```bash
GOCACHE=$(pwd)/.gocache go test ./... -cover
```

Current suite coverage: **95.7%** (`go test ./... -cover`).

`GOCACHE=$(pwd)/.gocache` sets the `GOCACHE` environment variable to a `.gocache` directory in your current working directory. This ensures that Go’s build and test cache files are stored locally, which can help keep your global cache clean and make builds more reproducible, especially in CI environments.

### Static analysis

```bash
go vet ./...
```

### Formatting

```bash
gofmt -w .
```

## Validating a Release

1. Run the test suite with coverage enabled.
2. Execute each example (`go run ./example/...`) to confirm the README snippets.
3. Optionally, inspect the package docs at [pkg.go.dev](https://pkg.go.dev/github.com/moehandi/konfig).

## Contributing

Issues and pull requests are welcome. Please include tests for new behaviour and run `gofmt` before submitting.

## License

MIT © 2025 Moehandi — see [LICENSE](LICENSE).
