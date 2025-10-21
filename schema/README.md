# RunLite Configuration Schema

JSON Schema files for `runlite.yml` configuration validation and IDE support.

## Files

- **`runlite.schema.yml`** - Schema in YAML format (easier to read/maintain)
- **`runlite.schema.json`** - Schema in JSON format (wider tool support)

Both files contain identical validation rules and can be used interchangeably.

## IDE Integration

### VSCode

#### Option 1: Inline Schema Reference (Recommended)

Add this comment at the top of your `runlite.yml`:

```yaml
# yaml-language-server: $schema=https://runlite.dev/schema/runlite.yml.json

app:
  name: my-app
```

Or use local schema:

```yaml
# yaml-language-server: $schema=./schema/runlite.schema.json

app:
  name: my-app
```

#### Option 2: VSCode Settings

Add to your `.vscode/settings.json`:

```json
{
  "yaml.schemas": {
    "https://runlite.dev/schema/runlite.yml.json": "runlite.yml",
    // Or use local schema:
    "./schema/runlite.schema.json": "runlite.yml"
  }
}
```

#### Option 3: Global User Settings

Add to VSCode user settings (Cmd+Shift+P → "Preferences: Open Settings (JSON)"):

```json
{
  "yaml.schemas": {
    "https://runlite.dev/schema/runlite.yml.json": ["**/runlite.yml", "**/runlite.yaml"]
  }
}
```

**Required Extension:** [YAML by Red Hat](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml)

### IntelliJ IDEA / WebStorm / GoLand

1. Open **Preferences** → **Languages & Frameworks** → **Schemas and DTDs** → **JSON Schema Mappings**
2. Click **+** to add new schema
3. Name: `RunLite Configuration`
4. Schema URL: `https://runlite.dev/schema/runlite.yml.json` (or local path)
5. Schema version: `JSON Schema version 7`
6. Add file path pattern: `runlite.yml` or `runlite.yaml`

### Vim / Neovim

With [coc.nvim](https://github.com/neoclide/coc.nvim) and [coc-yaml](https://github.com/neoclide/coc-yaml):

Add to `coc-settings.json`:

```json
{
  "yaml.schemas": {
    "https://runlite.dev/schema/runlite.yml.json": "runlite.yml"
  }
}
```

### Zed

#### Option 1: Inline Schema Reference (Recommended)

Add this comment at the top of your `runlite.yml`:

```yaml
# yaml-language-server: $schema=https://runlite.dev/schema/runlite.yml.json

app:
  name: my-app
```

#### Option 2: Zed Settings

Add to your `settings.json` (Cmd+, or Zed → Settings):

```json
{
  "lsp": {
    "yaml-language-server": {
      "settings": {
        "yaml": {
          "schemas": {
            "https://runlite.dev/schema/runlite.yml.json": "runlite.yml"
          }
        }
      }
    }
  }
}
```

Or for local schema:

```json
{
  "lsp": {
    "yaml-language-server": {
      "settings": {
        "yaml": {
          "schemas": {
            "./schema/runlite.schema.json": "runlite.yml"
          }
        }
      }
    }
  }
}
```

**Note:** Zed includes YAML language server support by default, no additional extensions needed.

## Features

### Validation

The schema validates:

- **App name**: Alphanumeric, dash, underscore only (`^[a-zA-Z0-9_-]+$`)
- **Paths**: Must be relative (start with `./`)
- **Health path**: Must start with `/`
- **Durations**: Go duration format (`30s`, `1m`, `1h30m`, etc.)
- **Command**: Must be relative path (no absolute paths)

### Auto-completion

Type-ahead suggestions for:
- All configuration keys
- Common values (e.g., health paths, durations)
- Template variables in `args`

### Documentation on Hover

Hover over any field to see:
- Field description
- Type information
- Default values
- Example values

### Error Detection

Real-time error highlighting for:
- Invalid app names
- Absolute paths (when relative required)
- Invalid duration formats
- Missing required fields
- Unknown properties

## Examples

### Valid Configuration

```yaml
# yaml-language-server: $schema=https://runlite.dev/schema/runlite.yml.json

app:
  name: my-app  # ✓ Valid

build:
  script: |
    go build -o ./bin/server .

  artifacts:
    - ./bin/server  # ✓ Relative path
    - source: ./public/
      dest: ./static/

run:
  command: ./bin/server  # ✓ Relative path

health:
  path: /health  # ✓ Starts with /
  timeout: 30s   # ✓ Valid Go duration
```

### Invalid Configuration (Schema Errors)

```yaml
app:
  name: my app!  # ✗ Invalid: spaces and special chars not allowed

build:
  artifacts:
    - /usr/bin/server  # ✗ Invalid: absolute path

run:
  command: /bin/server  # ✗ Invalid: must be relative

health:
  path: health  # ✗ Invalid: must start with /
  timeout: invalid  # ✗ Invalid: not a duration
```

## Publishing to Schema Store

To make the schema available globally without configuration:

1. Submit PR to [schemastore.org](https://github.com/SchemaStore/schemastore)
2. Add entry to `src/api/json/catalog.json`:

```json
{
  "name": "RunLite Configuration",
  "description": "Configuration file for RunLite deployment platform",
  "fileMatch": ["runlite.yml", "runlite.yaml"],
  "url": "https://runlite.dev/schema/runlite.yml.json"
}
```

Once merged, all IDEs will automatically validate `runlite.yml` files without any setup.

## Validation Rules

### Required Fields

- `app.name` - Application identifier

### Optional Sections

All other sections (`build`, `run`, `health`, `database`, `static`, `deploy`) are optional.

### Field Patterns

| Field | Pattern | Description |
|-------|---------|-------------|
| `app.name` | `^[a-zA-Z0-9_-]+$` | Alphanumeric, dash, underscore |
| `*.path` | `^\\.\\/` | Must start with `./` |
| `run.command` | `^\\.\\/` | Must start with `./` |
| `health.path` | `^\\/` | Must start with `/` |
| Durations | `^[0-9]+(ns\|us\|µs\|ms\|s\|m\|h)+$` | Go duration format |

### Artifacts Syntax

Supports two formats:

**Simple array:**
```yaml
artifacts:
  - ./bin/server
  - ./public/
```

**Object with rename:**
```yaml
artifacts:
  - source: ./bin/server
    dest: ./server
  - source: ./public/
    dest: ./static/
```

## Development

### Testing Schema Changes

1. Edit `runlite.schema.yml`
2. Convert to JSON: `yq eval -o=json runlite.schema.yml > runlite.schema.json`
3. Test in IDE with sample `runlite.yml`

### Validating Schema File

Validate the schema itself:

```bash
# Using ajv-cli
npm install -g ajv-cli
ajv compile -s schema/runlite.schema.json

# Using check-jsonschema
pip install check-jsonschema
check-jsonschema --check-metaschema schema/runlite.schema.json
```

## See Also

- [Configuration Documentation](../docs/config-file.md)
- [JSON Schema Documentation](https://json-schema.org/)
- [VSCode YAML Extension](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml)
- [Schema Store](https://www.schemastore.org/)
