# Plan: Compress Embedded AKWF Wavetables via tar.gz

**Status: Done**

## Problem

The AKWF wavetable library is embedded as ~65 individual JSON files using
`//go:embed *.json`. Each file stores waveform samples as arrays of decimal
float64 strings. This format is verbose: decimal notation for floats produces
high redundancy, and `embed.FS` stores files verbatim with no compression.
The result is a significantly inflated binary.

## Goal

Reduce the binary size contribution of the AKWF data by 75–85% with minimal
change to the public API (`LoadAll() []Entry` stays the same).

## Approach

Replace the 65 individual embedded JSON files with a single embedded
`akwf.tar.gz`. Because tar concatenates all files into one stream before
compression, gzip can exploit repetition *across* banks — yielding better
ratios than zip, which compresses each file independently. Since `LoadAll`
already reads every bank, sequential tar reading is perfectly suited.

No change to `Entry`, `LoadAll`, or any caller is required.

## Implementation Steps

### Step 1 — Generate `akwf.tar.gz`

Add a `compress` target to the `Makefile`:

```makefile
compress:
	cd persistence/akwf && tar -czf akwf.tar.gz *.json
```

`-C` / `cd` strips directory prefixes so entries are bare filenames;
`-z` enables gzip; `-f` names the output file.

Run `make compress` once and commit `akwf.tar.gz`. Regenerate only when
the source JSON files change — this is a rare, manual operation.

> **Windows note**: `tar` is available in Windows 10+ (via the built-in
> BSD tar), Git Bash, and WSL. The generated `akwf.tar.gz` is committed
> to the repo so a normal `go build` on any platform requires no extra
> tooling.

### Step 2 — Update `loader.go`

Replace the `embed.FS` declaration and imports. At startup, decompress the
tar.gz once into a `map[string][]byte` (filename → raw JSON), then use that
map in place of `embed.FS`:

```go
//go:embed akwf.tar.gz
var tarGzData []byte

// readArchive decompresses the embedded tar.gz and returns a map of
// bare filename → raw file bytes.
func readArchive() map[string][]byte {
    gr, err := gzip.NewReader(bytes.NewReader(tarGzData))
    if err != nil {
        log.Printf("akwf: gzip: %v", err)
        return nil
    }
    defer gr.Close()
    tr := tar.NewReader(gr)
    files := make(map[string][]byte)
    for {
        hdr, err := tr.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            log.Printf("akwf: tar: %v", err)
            return nil
        }
        data, err := io.ReadAll(tr)
        if err != nil {
            log.Printf("akwf: read %s: %v", hdr.Name, err)
            return nil
        }
        files[hdr.Name] = data
    }
    return files
}
```

`LoadAll` calls `readArchive()` once, reads `manifest.json` from the map,
then passes the map to `loadBank`:

```go
func LoadAll() []Entry {
    files := readArchive()
    if files == nil {
        return nil
    }
    // unmarshal manifest.json from files["manifest.json"] ...
    // for each bank: loadBank(files, bank) ...
}
```

`loadBank` receives the map and looks up `bank + ".json"` — identical
logic to the current `fs.ReadFile` call.

### Step 3 — Remove the raw JSON files from the embed directive

Change:
```go
//go:embed *.json
var fs embed.FS
```

To:
```go
//go:embed akwf.tar.gz
var tarGzData []byte
```

The individual `*.json` files remain on disk as the source of truth (used
by `make compress` and useful for inspection), but are no longer embedded.

Update imports: remove `"embed"`, add `"archive/tar"`, `"bytes"`,
`"compress/gzip"`, `"io"`.

### Step 4 — Add `compress` target to Makefile

```makefile
compress:
	cd persistence/akwf && tar -czf akwf.tar.gz *.json
```

## Affected Files

| File                              | Change                                          |
| --------------------------------- | ----------------------------------------------- |
| `persistence/akwf/loader.go`      | Switch from `embed.FS` to embedded `[]byte` + `archive/tar` + `compress/gzip` reader |
| `persistence/akwf/akwf.tar.gz`    | Generated at build time; gitignored             |
| `.gitignore`                      | Add `persistence/akwf/akwf.tar.gz`              |
| `Makefile`                        | Add `compress` target                           |
| `.github/workflows/ci.yaml`       | Add `make compress` step before build           |
| `.github/workflows/publish.yml`   | Add `make compress` step before build           |

No changes to callers, `Entry` struct, or persistence formats. No new Go source files needed.

## Expected Size Reduction

JSON float arrays compress very well under gzip. Compressing across all
banks in a single stream (tar.gz) extracts additional savings vs per-file
compression (zip). Expect the embedded data to shrink to roughly 10–20%
of its original size — a 5× to 10× reduction.

## Risks and Considerations

**Startup cost**: The entire tar.gz is decompressed once in `LoadAll` into
a `map[string][]byte`. This is a single sequential pass — fast and simple.
JSON parsing then proceeds identically to today.

**`make compress` must be re-run when JSON files change**: A stale
`akwf.tar.gz` will silently serve old data. Run `make compress` and commit
the updated archive whenever bank files are updated.

**`manifest.json` handling**: The manifest is included in the tar.gz and
read from the in-memory map, replacing the current `fs.ReadFile("manifest.json")` call.
