# keyvalues
Golang library for parsing Valve keyvalue format files. This library constructs a simple kv node tree that you can
query any structure(s) and any property(s) of.

It has been tested against various gameinfo.txt engine files, but should work with other KeyValue files as well.

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
	f,_ := reader.Read()

    // counterstrike: source's gameinfo.txt would return "Counter-Strike Source"
    log.Println(f.FindByKey("GameInfo").FindStringByKey("game")

    // counterstrike: source's gameinfo.txt would return 1
    log.Println(f.FindByKey("GameInfo").FindIntByKey("nomodels")

    // counterstrike: source's gameinfo.txt would return 240
    log.Println(f.FindByKey("GameInfo").FindByKey("FileSystem").FindIntByKey("SteamAppId")
}
```


### Todo
* Implement multi-line values. At present, a `\n` character in a quoted value will break the parser. This is how CS:GO
Hammer behaves. However, other versions of Hammer support this, as well as all engine versions. Worth noting what spec
is available doesn't cover this behaviour.
* Implement boolean value type
* Implement pointer value type (unsure if there is any point to this besides matching spec)
* Proper test coverage