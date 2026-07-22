---
paths:
  - "**/*.go"
---

# Go conventions

## Godoc: no restatements
Only write a doc comment when it adds information beyond the identifier's name —
constraints, formats, edge cases (IPv6 bracketing, valid port ranges,
wire-format contracts), invariants, or a non-obvious *why*. Skip comments that
merely restate the name (`// Port returns the port number.`,
`// MustNewAddress is like NewAddress but panics on error.`). This holds even
where Go's lint convention asks for an identifier-prefixed sentence, and it
applies to `Must*` constructors and trivial accessors too — the user considers
those restatements noise, not documentation.

## Blank line before each logical step
Leave a blank line before each logical statement inside a function body — `if`,
`for`, `switch`, `return`, or a new local declaration that starts a fresh step.
Lines that are part of the same step stay together. Each step should read as its
own paragraph.

## Constructor naming
- `New*` — takes structured arguments.
- `Parse*` — takes a string that needs lexical work (mirrors `time.Parse`,
  `net.ParseIP`).
- `Must*` — panics on error.

## Error wording (domain constructors)
`failed to create <X>[ from %q]: <reason>` — verb-led, identifier as `%q`,
colon-separated reason. Use `contents are not valid <FORMAT>` for parse
failures.

## Typed errors
Use the `internal/errors` layer: `errors.Newf(errors.TypeInvalidInput, …)` /
`errors.Wrapf(err, errors.TypeInternal, …)`. Types: `TypeInvalidInput`,
`TypeNotFound`, `TypeInternal`, `TypeFatal`, `TypeUnsupported`. Attach errors to
slog records with `errors.LogAttr(err)`. Do not reach for `fmt.Errorf`.
