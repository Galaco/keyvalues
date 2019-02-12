[![GoDoc](https://godoc.org/github.com/Galaco/KeyValues?status.svg)](https://godoc.org/github.com/Galaco/KeyValues)
[![Go report card](https://goreportcard.com/badge/github.com/galaco/KeyValues)](https://goreportcard.com/badge/github.com/galaco/KeyValues)
[![Build Status](https://travis-ci.com/Galaco/keyvalues.svg?branch=master)](https://travis-ci.com/Galaco/keyvalues)

# keyvalues
Go library for parsing Valve keyvalue format files. This library constructs a simple kv node tree that you can
query any structure(s) and any property(s) of.

It has been tested against various gameinfo.txt engine files, but should work with other KeyValue files as
well (such as .vmf or .vmt).

### Usage
```golang
package main

import (
    "log"
    "os"
    "github.com/galaco/keyvalues"
)

func main() {
	file,_ := os.Open("gameinfo.txt")

	reader := keyvalues.NewReader(file)
	kv,_ := reader.Read()

    // counterstrike: source's gameinfo.txt would return "Counter-Strike Source"
    gameInfoNode,_ := kv.Find("GameInfo")
    gameNode,_ := gameInfoNode.Find("game")
    log.Println(gameNode.AsString())

    // counterstrike: source's gameinfo.txt would return 1
    noModelsNode,_ := gameInfoNode.Find("nomodels")
    log.Println(noModelsNode.AsInt())

    // counterstrike: source's gameinfo.txt would return 240
    fileSystemNode,_ := gameInfoNode.Find("FileSystem")
    appIdNode,_ := fileSystemNode.Find("SteamAppId")
    log.Println(appIdNode.AsInt())
}
```

### Todo
* Implement multi-line values. At present, a `\n` character in a quoted value will break the parser. This is how CS:GO
Hammer behaves. However, other versions of Hammer support this, as well as all engine versions. Worth noting what spec
is available doesn't cover this behaviour.
* Implement pointer value type (unsure if there is any point to this besides matching spec)
* Proper test coverage
