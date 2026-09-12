# Strings, runes & bytes

## Why Does This Matter?

Go strings are UTF-8 bytes, not Unicode characters. Every engineer
coming from Java, Python, or JavaScript has at some point written
`s[i]` expecting a character and gotten a byte. In Go the model is
honest about what text is: a sequence of bytes, with runes (Unicode
code points) available when you ask for them. Get this model right and
you stop writing code that mangles names like "José" or "田中".

## Mental Model

A string is a read-only byte slice with two guarantees: its bytes are
valid UTF-8 as far as Go is concerned (invalid sequences become
replacement characters when printed), and it cannot be mutated in place.

```mermaid
flowchart LR
    S["\"héllo\"<br/>6 bytes"] --- B["h(1) é(2) l(1) l(1) o(1)"]
    S --- R["range over s<br/>h é l l o<br/>5 runes"]
```

Three distinct things, three distinct types:

| Concept | Type | Meaning |
|---|---|---|
| Byte | `byte` (`uint8`) | one UTF-8 code unit |
| Rune | `rune` (`int32`) | one Unicode code point |
| Character | no type | what users *see*; needs grapheme clustering, not in stdlib |

The gap between rune and perceived character matters: "é" can be one
rune (U+00E9) or two (e + combining accent), and emoji are frequently
multi-rune. Go gives you code points; user-perceived characters are a
harder problem (see Further Reading).

## Indexing vs ranging: the trap in one example

```go
s := "héllo"
s[0]              // 104: byte 'h'
s[1]              // 195: half of é, meaningless alone
len(s)            // 6: bytes

for i, r := range s {  // r is a rune; i is its BYTE index
    fmt.Println(i, r)
}
// 0 104 / 1 233 / 3 108 / 4 108 / 5 111
// note: i jumps 1 → 3 because é is 2 bytes
```

`range` decodes UTF-8 as it walks; indexing does not. Both are correct;
they answer different questions.

## string vs []byte: when to convert

| | `string` | `[]byte` |
|---|---|---|
| Mutability | never | freely |
| Map keys / comparability | yes (`==`) | no |
| Conversion cost | O(n) copy | O(n) copy |
| Typical home | identifiers, keys, messages | I/O buffers, protocols |

```go
b := []byte(s)     // copy: mutation-safe
s2 := string(b)    // copy back
```

Every conversion copies, and hot paths feel it. The compiler eliminates
some copies in specific patterns (map lookups like `m[string(b)]`,
comparisons `string(b) == s`); do not rely on it beyond those.

Never mutate bytes you got from a string via unsafe tricks: strings may
be interned or shared, and writing to their storage is undefined
behavior territory. If you need mutation, convert honestly.

## The rune/byte workhorses

```go
strings.Builder      // build strings without O(n²) copies
strings.NewReader(s) // s as an io.Reader without copying
utf8.RuneCountInString(s)          // rune count: len() is bytes
[]rune(s)                          // runes as a slice (O(n) copy)
strings.Split(s, "")               // CAUTION: splits bytes, not runes
```

`strings.Builder` is the answer to "build a big string in a loop":
it amortizes growth like a slice and its `WriteString` never copies
more than once per growth step.

```go
var b strings.Builder
for _, part := range parts {
    b.WriteString(part)
}
out := b.String()
```

The O(n²) mistake this replaces, with real benchmark numbers, is in
[19-performance/01](../19-performance/01-measure-first.md).

## Common Mistakes

- **`len(s)` as a character count**: it counts bytes. "田中" is 2
  characters and 6 bytes. Use `utf8.RuneCountInString`.
- **`s[:5]` truncation** that splits a rune, producing U+FFFD garbage.
  Cut on rune boundaries (`utf8.RuneStart`) or decode first.
- **Uppercase as a normalization** (`strings.ToUpper`): locale- and
  casefold-unaware; wrong for identifiers and dedup keys.
- **Byte/rune confusion in regex**: `\p{L}` (Go regexp) matches letters
  in UTF-8 space correctly, but hand-rolled byte ranges do not.
- **Assuming one rune = one visible character**: family emoji,
  combining accents, and ZWJ sequences are multiple runes.

## Idiomatic Go

```go
// Iterate runes when text semantics matter.
for _, r := range s {
    if unicode.IsLetter(r) { count++ }
}

// Normalize for comparison: trim, fold case, done deliberately.
key := strings.ToLower(strings.TrimSpace(raw))

// Scan byte-oriented protocols with bytes, not strings.
if bytes.HasPrefix(buf, []byte("GET ")) { /* ... */ }
```

## Performance Considerations

- `len` is O(1) (header field); `RuneCountInString` is O(n).
- Substring `s[a:b]` is O(1) and shares memory: like slices, a small
  window can pin a large string alive. Copy if the parent must be
  released (same leak as [01-arrays-and-slices](01-arrays-and-slices.md)).
- `string(b)` / `[]byte(s)` are O(n) copies; batch them at boundaries,
  not inside loops.
- Comparisons and map lookups with string keys are fast, FNV/MemHash
  based; this is why string-keyed maps are the default cache shape.

## Concurrency Considerations

Strings are immutable, so they are safe to share across goroutines
freely: one of Go's quiet superpowers for pipelines. `[]byte` buffers
are not: hand ownership to exactly one goroutine at a time, or copy.

## Security Considerations

- String concatenation of user input into queries or commands is the
  injection bug; parameterize (see [13-databases](../13-databases/README.md)
  and [21-security](../21-security/README.md)).
- Treat `strings.Replace`-based sanitization as insufficient alone;
  validation happens on parsed, typed data.
- Secrets in strings are still garbage-collectable copies; avoid
  formatting them into errors or logs (use redaction at the boundary).

## Testing Strategy

Table-driven tests with multi-script input catch rune bugs cheaply:
include ASCII, Latin-1 accents, CJK, and an emoji row. Assert on
`RuneCountInString`, byte length, and round-trip `[]rune` separately.
The `examples/text` package in this section demonstrates the pattern.

## Interview Questions

1. *What does `len("héllo")` return, and why?*: 6: bytes, not runes;
   é is two bytes in UTF-8.
2. *What is the index semantics of `for i, r := range s`?*: `i` is the
   byte offset of rune `r`; it jumps for multi-byte runes.
3. *When does `string(b)` copy, and when not?*: Always copies as far as
   the language guarantees; the compiler may optimize specific lookup/
   compare shapes, which you must not depend on.
4. *Why are strings safe to share between goroutines?*: Immutable and
   backed by read-only memory semantics; mutation requires an explicit
   `[]byte` copy.
5. *How do you truncate a UTF-8 string safely?*: Walk rune boundaries
   (`for range` or `utf8.DecodeRuneInString`) and cut between runes.

## Practice Exercises

1. Write `Truncate(s string, maxRunes int) string` that never splits a
   rune; test with emoji and combining accents.
2. Implement `CountWords` on `io.Reader` using `bufio.Scanner` with
   `ScanWords`; discuss why `Split(s, " ")` is the wrong tool.
3. Benchmark `+=` concatenation vs `strings.Builder` for 10k parts;
   record the factor and keep it in your notes for
   [19-performance](../19-performance/README.md).

## Further Reading

- [Strings, bytes, runes and characters in Go](https://go.dev/blog/strings)
- [`strings` package](https://pkg.go.dev/strings)
- [`unicode/utf8` package](https://pkg.go.dev/unicode/utf8)
- [Unicode Text Segmentation](https://unicode.org/reports/tr29/) (the
  grapheme problem, beyond stdlib)
