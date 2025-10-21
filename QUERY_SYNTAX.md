# Query Syntax Documentation

The Hue CLI supports a powerful query system for controlling multiple lights at once using pattern matching and filters.

## Quick Start

Queries start with `@"..."` and support progressive filtering:

```bash
# Simple queries
hue lights list @"office"                    # All office lights
hue lights on @"bedroom"                     # Turn on bedroom lights
hue lights color @"sultan" red               # All sultan bulbs to red

# Preview before executing
hue lights color @"sultan" red --dry-run
```

## Query Syntax

### Basic Pattern

```
@"<filter1> <filter2> <filter3> ..."
   └─AND──┘ └─AND──┘ └─AND──┘
```

Each space-separated term narrows down the results (AND logic).

## Filter Types

### Text Matching (Default)

When you don't specify a prefix, the query searches **both name AND archetype**:

```bash
hue lights list @"sultan"        # Finds lights with "sultan" in name OR archetype
hue lights list @"lamp"          # Finds lights with "lamp" in name OR archetype
hue lights list @"office"        # Finds lights with "office" in name
```

**Wildcards:**
```bash
hue lights list @"*lamp"         # Ends with "lamp"
hue lights list @"bedroom*"      # Starts with "bedroom"
hue lights list @"*strip*"       # Contains "strip"
```

### Explicit Field Filters

#### `name:value` - Search name only
```bash
hue lights list @"name:lamp"           # Only names containing "lamp"
hue lights color @"name:desk" blue     # Names with "desk"
```

#### `type:value` - Search archetype only
```bash
hue lights list @"type:sultan"         # sultan_bulb archetype
hue lights list @"type:go"             # hue_go archetype
hue lights on @"type:strip"            # All lightstrips
```

#### `room:value` - Filter by actual room membership
```bash
hue lights color @"room:bedroom" red       # All lights in bedroom room
hue lights on @"room:office"               # All lights in office room
hue lights list @"room:kitchen,dining"     # Kitchen OR dining room lights
```

Uses actual room metadata - finds ALL lights in the room regardless of their names.

### State Filters

#### `on:` - Currently powered on
```bash
hue lights list @"on:"                 # All lights that are on
hue lights off @"on: brightness>80"    # Turn off bright lights
```

#### `off:` - Currently powered off
```bash
hue lights list @"off:"                # All lights that are off
hue lights on @"off: room:bedroom"     # Turn on bedroom lights that are off
```

### Capability Filters

#### `with:effects` - Supports effects
```bash
hue lights list @"with:effects"
hue lights effect @"with:effects" fire
```

#### `with:color` - Supports color
```bash
hue lights list @"with:color"
hue lights color @"with:color" blue
```

### Effect Filters

#### `effect:value` - Currently running effect
```bash
hue lights list @"effect:fire"         # Lights running fire effect
hue lights list @"effect:none"         # No active effect
hue lights list @"effect:*"            # Any active effect (not none)
hue lights effect @"effect:*" no_effect # Stop all effects
```

### Brightness Filters

#### `brightness>N` - Greater than threshold
```bash
hue lights list @"brightness>80"
hue lights off @"brightness>90"        # Turn off very bright lights
hue lights brightness @"brightness>50" 30  # Dim the bright ones
```

#### `brightness<N` - Less than threshold
```bash
hue lights list @"brightness<30"
hue lights brightness @"brightness<50" 100  # Boost dim lights
hue lights on @"off: brightness<20"    # Turn on lights that were dim
```

### Advanced Filters

#### `mode:value` - Operating mode
```bash
hue lights list @"mode:normal"
hue lights list @"mode:streaming"
```

#### `gamut:A|B|C` - Color gamut type
```bash
hue lights list @"gamut:C"             # Latest color gamut
hue lights color @"gamut:A,B" blue     # Older gamut lights
```

#### `id:pattern` - Match by ID
```bash
hue lights list @"id:abc*"             # IDs starting with "abc"
hue lights state @"id:f23f9f2e*"       # Match specific ID pattern
```

#### `all` or `*` - Everything
```bash
hue lights list @"all"
hue lights list @"*"
```

## Logical Operators

### AND (Space)

Each space-separated term filters the results:

```bash
hue lights list @"office lamp"         # office AND lamp
hue lights color @"sultan brightness>50" red
hue lights on @"with:effects off:"     # Effect-capable AND off
```

### OR (Comma)

Commas within a term create OR logic:

```bash
hue lights on @"room:bedroom,office"          # bedroom OR office
hue lights list @"type:go,strip"              # hue_go OR lightstrip
hue lights color @"sultan,go" blue            # sultan OR go
```

### NOT (Dash)

Dash prefix excludes matches:

```bash
hue lights color @"office -desk" red          # office, exclude desk
hue lights on @"* -bedroom"                   # all except bedroom
hue lights list @"with:color -type:go"        # color-capable, not hue_go
```

### Combining Operators

```bash
# (office OR bedroom) AND lamp AND NOT desk
hue lights color @"office,bedroom lamp -desk" red

# (on) AND (brightness>50) AND NOT (bedroom OR kitchen)
hue lights off @"on: brightness>50 -bedroom -kitchen"

# (sultan OR go) AND (brightness<80) AND (with:color)
hue lights brightness @"sultan,go brightness<80 with:color" 100
```

## Global Flags

### `--dry-run`
Preview what would be affected without making changes:

```bash
hue lights color @"*" red --dry-run
# Output shows all matched lights without changing them
```

### `--verbose` / `-v`
Show detailed information about matches:

```bash
hue lights on @"office" --verbose
# Output:
# Query '@office' matched 4 light(s):
#   ✓ Office Desk Lamp
#   ✓ Office Ceiling
#   ✓ Office Strip
#   ✓ Office Hue Go
# ✓ Turned on 4/4 lights
```

### `--quiet` / `-q`
Suppress non-essential output:

```bash
hue lights color @"sultan" blue --quiet
```

### `--json`
JSON output for scripting:

```bash
hue lights list @"with:effects" --json
```

## Examples

### Common Scenarios

**Turn off all lights except bedroom:**
```bash
hue lights off @"on: -bedroom"
```

**Set all sultan bulbs to warm white:**
```bash
hue lights color @"sultan" "#FFE4B5"
```

**Fire effect on all effect-capable office lights:**
```bash
hue lights effect @"office with:effects" fire
```

**Boost all dim lights to 100%:**
```bash
hue lights brightness @"brightness<40" 100
```

**Turn on all Hue Go and lightstrips:**
```bash
hue lights on @"type:go,strip"
```

**Set bedroom lights that are on to 20%:**
```bash
hue lights brightness @"room:bedroom on:" 20
```

### Exploration

**Find all lights with active effects:**
```bash
hue lights list @"effect:*"
```

**List all high-brightness lights:**
```bash
hue lights list @"brightness>80"
```

**Show all sultan bulbs in bedroom:**
```bash
hue lights list @"sultan bedroom"
```

**Find all color lights that aren't ceiling lights:**
```bash
hue lights list @"with:color -type:ceiling"
```

### Batch Operations

**Stop all active effects:**
```bash
hue lights effect @"effect:*" no_effect
```

**Turn off all lights except those below 30% brightness:**
```bash
hue lights off @"on: -brightness<30"
```

**Set all office and bedroom lights to 50%:**
```bash
hue lights brightness @"office,bedroom" 50
```

## Performance

Commands using queries execute **concurrently** with up to 10 parallel API requests:

- **3 lights:** ~0.7s
- **10 lights:** ~1.5s
- **42 lights:** ~2.7s

Sequential execution would be ~4-5s for 42 lights, making concurrent execution **~37% faster** for large batches.

## Tips

1. **Start broad, refine progressively:**
   ```bash
   hue lights list @"office"           # See what matches
   hue lights list @"office lamp"      # Narrow it down
   hue lights color @"office lamp" red # Execute
   ```

2. **Use `--dry-run` for safety:**
   ```bash
   hue lights off @"*" --dry-run       # Preview before executing!
   ```

3. **Combine state and attribute filters:**
   ```bash
   hue lights brightness @"on: brightness>80" 50
   ```

4. **Use wildcards for patterns:**
   ```bash
   hue lights on @"*lamp"              # All lamps
   hue lights color @"bedroom*" blue   # Bedroom lights
   ```

5. **Check what matched with `-v`:**
   ```bash
   hue lights color @"sultan" red -v
   ```
