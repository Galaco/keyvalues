[![Go Reference](https://pkg.go.dev/badge/github.com/galaco/KeyValues/v2.svg)](https://pkg.go.dev/github.com/galaco/KeyValues/v2)
[![Go report card](https://goreportcard.com/badge/github.com/galaco/KeyValues)](https://goreportcard.com/report/github.com/galaco/KeyValues)
[![Test](https://github.com/galaco/KeyValues/actions/workflows/test.yml/badge.svg)](https://github.com/galaco/KeyValues/actions/workflows/test.yml)

# keyvalues
> A zero-dependency library for reading and writing Valve KeyValues data.

Go library for Valve's KeyValues text format, as used by gameinfo.txt, .vmt, .vmf, .res and many other Source engine
files. It reads a document into a tree of nodes you can query, modify, merge and write back out.

```
go get github.com/galaco/KeyValues/v2
```

### Usage
```golang
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/galaco/KeyValues/v2"
)

func main() {
	file, err := os.Open("gameinfo.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	doc, err := keyvalues.NewReader(file).Read()
	if err != nil {
		log.Fatal(err)
	}

	gameInfo, err := doc.Find("GameInfo")
	if err != nil {
		log.Fatal(err)
	}

	// Counter-Strike: Source's gameinfo.txt prints "Counter-Strike Source".
	game, err := gameInfo.Find("game")
	if err != nil {
		log.Fatal(err)
	}
	name, err := game.AsString()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(name)

	// Keys are matched ignoring case, and can repeat. Prints each search path.
	fileSystem, err := gameInfo.Find("filesystem")
	if err != nil {
		log.Fatal(err)
	}
	searchPaths, err := fileSystem.Find("SearchPaths")
	if err != nil {
		log.Fatal(err)
	}
	for _, path := range searchPaths.FindAll("Game") {
		value, _ := path.AsString() // only fails for a block
		fmt.Println(value)
	}
}
```

Each node is either a **leaf**, holding a string value, or a **block**, holding child nodes. `Read` returns a document
root: a block with an empty key whose children are the top-level nodes of the file, so files with several top-level
nodes work the same way as files with one.

Values are stored exactly as written. `AsString` always works on a leaf, and `AsInt` and `AsFloat` parse the value
when called.

### Format support
* Quoted and unquoted strings, including quoted values that span several lines.
* `//` comments. A `//` inside a quoted string, such as a URL, is part of the string.
* Braces anywhere: `"key" {`, `{ "a" "b" }` on one line, or no whitespace at all.
* Conditionals such as `"key" "value" [$WIN32]`. They are kept on the node (`Condition()` returns `$WIN32`) but not
  evaluated.
* `#base` and `#include` directives are read as ordinary top-level leaves. They are not resolved.
* UTF-8, and UTF-16 (either byte order) as used by localization files. UTF-16 is detected from its byte order mark;
  anything without one is read as UTF-8.
* `\n`, `\r\n` or `\r` line endings.
* Escape sequences (`\n`, `\t`, `\"`, `\\` and so on) are off by default, because most engine files use backslashes in
  paths such as `"concrete\concretefloor001a"`. Enable them with `Reader.UseEscapeSequences` and
  `Writer.UseEscapeSequences` for files that use them, such as localization and some UI resource files.

Malformed input returns a `*keyvalues.SyntaxError` with the line and column of the problem. Columns count characters,
not bytes.

### Writing
```golang
material := keyvalues.NewBlock("LightmappedGeneric",
	keyvalues.NewKeyValue("$basetexture", `concrete\concretefloor001a`),
	keyvalues.NewKeyValue("$surfaceprop", "concrete"),
)
err := keyvalues.NewWriter(os.Stdout).Write(material)
```

Writing a document returned by `Read` writes its top-level nodes. Trees can be changed with `AddChild`, `RemoveChild`
and `SetValue` before writing. Output is UTF-8; `Writer.UseUTF16` writes little-endian UTF-16 with a byte order mark
instead, as localization files use.

### Merging
`Patch` and `Replace` merge one tree into another, matching keys ignoring case. `Patch` only adds keys that are
missing, while `Replace` also overwrites existing values. Neither changes its inputs. The overlay's root key must match
the base's, or be `patch` or `replace` respectively:

```golang
// changes holds: "replace" { "game" "My Mod" }
merged, err := changes.Replace(gameInfo)
```

### Migrating from v1
v2 is a new parser and a smaller, stricter API. The module path is now `github.com/galaco/KeyValues/v2`.

* `Read` returns `(*KeyValue, error)` and always returns a document root. Use `doc.Find("GameInfo")` even when the file
  has a single root. v1 returned that root directly, or a `$root` node when there was more than one.
* `Read` returns errors for malformed input and for errors from the underlying reader. v1 silently dropped or misread
  data in many cases, and could panic or loop forever.
* `NewReader` returns `*Reader`.
* `ValueType`, `Type()` and value type guessing are gone. Use `IsBlock()` instead of `HasChildren()` or
  `Type() == ValueArray`. `AsString` now works on every leaf, including `"123"`, and `AsFloat` accepts `"1"`.
* `NewKeyValuePair(key, value, type)` is replaced by `NewKeyValue(key, value)` and `NewBlock(key, children...)`.
* `Children()` returns a slice with no error, and `nil` for a leaf. `FindAll` returns `nil` rather than an error when
  nothing matches.
* `Find`, `RemoveChild` and the accessors return errors wrapping `ErrNotFound`, `ErrNotBlock` or `ErrNotLeaf`, for use
  with `errors.Is`. `AsInt` and `AsFloat` return `0` rather than `-1` on error.
* `Patch` and `Replace` return `*KeyValue` and no longer modify either tree.
