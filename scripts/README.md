# Scripts

This directory contains utility scripts for development and setup.

## Available Scripts

### install-swag.sh

Installs the swag CLI tool and downloads Go dependencies for OpenAPI specification generation.

**Usage:**

```bash
./scripts/install-swag.sh
```

**What it does:**

- Checks for Go installation
- Installs `swag` CLI tool via `go install`
- Downloads all Go module dependencies
- Verifies module integrity

**Requirements:**

- Go 1.21 or later
- Internet connection

**Output:**

- swag CLI installed to `$GOPATH/bin/swag`
- All Go dependencies downloaded to module cache

### generate-openapi.sh

Regenerates the OpenAPI specification from Go code annotations and prepares it for SDK generation.

**Usage:**

```bash
./scripts/generate-openapi.sh
```

Or via Makefile:

```bash
make swagger
```

**What it does:**

- Checks for `swag` CLI tool installation
- Runs `swag init` to regenerate OpenAPI spec from code annotations
- Generates `docs/swagger.json` and `docs/swagger.yaml`
- Creates `api/` directory if it doesn't exist
- Copies `docs/swagger.json` to `api/openapi.json` for SDK generation
- Displays file sizes for verification

**Requirements:**

- Go 1.21 or later
- swag CLI (install via `./scripts/install-swag.sh`)

**Output:**

- `docs/swagger.json` - OpenAPI 2.0 specification in JSON format
- `docs/swagger.yaml` - OpenAPI 2.0 specification in YAML format
- `docs/docs.go` - Generated Go package for embedding the spec
- `api/openapi.json` - Copy of swagger.json for SDK generation

**When to use:**

- After adding or modifying API endpoint annotations
- Before generating TypeScript SDKs
- When updating API documentation

### install-air.sh

Installs the Air CLI tool for hot reload development.

**Usage:**

```bash
./scripts/install-air.sh
```

**What it does:**

- Checks for Go installation
- Installs `air` CLI tool via `go install`
- Verifies Air installation and PATH configuration
- Checks for `.air.toml` configuration file

**Requirements:**

- Go 1.21 or later
- Internet connection

**Output:**

- Air CLI installed to `$GOPATH/bin/air`
- Ready to use with `make dev` or `air` command

**When to use:**

- First-time project setup
- After updating Go version
- When Air command is not found

## Future Scripts

Additional scripts will be added for:

- Development environment setup (subtask 5.3)
- Testing and linting automation

### subtitle-blind-eval.py

Builds a blind A/B scoring sheet for two subtitle-translation variants of the same episode, then tallies the scores against the pre-registered thresholds from story `eval-1-translation-blind-eval`.

**Usage:**

```bash
# 1. sample 50 shared cues, shuffle left/right, hide the answer key
python3 scripts/subtitle-blind-eval.py build \
  --source eval/<slug>/source.srt --a eval/<slug>/haiku.srt --b eval/<slug>/sonnet.srt \
  --out eval/<slug> --sample 50 --seed 42

# 2. fill left_score / right_score (0/1/2) in eval/<slug>/sheet.csv

# 3. un-blind and print the verdict
python3 scripts/subtitle-blind-eval.py score eval/<slug>
```

**Requirements:** Python 3.9+, stdlib only.
