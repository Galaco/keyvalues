package keyvalues_test

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/galaco/KeyValues/v2"
)

const gameInfo = `"GameInfo"
{
	game		"Counter-Strike Source"
	nomodels	1
	FileSystem
	{
		SteamAppId	240
		SearchPaths
		{
			Game	cstrike
			Game	hl2
		}
	}
}`

func ExampleReader_Read() {
	doc, err := keyvalues.NewReader(strings.NewReader(gameInfo)).Read()
	if err != nil {
		log.Fatal(err)
	}

	gameInfo, err := doc.Find("GameInfo")
	if err != nil {
		log.Fatal(err)
	}
	game, err := gameInfo.Find("game")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(game.AsString())

	noModels, err := gameInfo.Find("nomodels")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(noModels.AsInt())

	fileSystem, err := gameInfo.Find("FileSystem")
	if err != nil {
		log.Fatal(err)
	}
	searchPaths, err := fileSystem.Find("SearchPaths")
	if err != nil {
		log.Fatal(err)
	}
	for _, path := range searchPaths.FindAll("game") {
		fmt.Println(path.AsString())
	}
	// Output:
	// Counter-Strike Source <nil>
	// 1 <nil>
	// cstrike <nil>
	// hl2 <nil>
}

func ExampleWriter_Write() {
	material := keyvalues.NewBlock("LightmappedGeneric",
		keyvalues.NewKeyValue("$basetexture", `concrete\concretefloor001a`),
		keyvalues.NewKeyValue("$surfaceprop", "concrete"),
	)
	if err := keyvalues.NewWriter(os.Stdout).Write(material); err != nil {
		log.Fatal(err)
	}
	// Output:
	// "LightmappedGeneric"
	// {
	// 	"$basetexture"	"concrete\concretefloor001a"
	// 	"$surfaceprop"	"concrete"
	// }
}

func ExampleKeyValue_Replace() {
	read := func(s string) *keyvalues.KeyValue {
		doc, err := keyvalues.NewReader(strings.NewReader(s)).Read()
		if err != nil {
			log.Fatal(err)
		}
		return doc.Children()[0]
	}
	base := read(`"GameInfo" { "game" "Half-Life 2" "type" "singleplayer_only" }`)
	changes := read(`"replace" { "game" "My Mod" "developer" "Me" }`)

	merged, err := changes.Replace(base)
	if err != nil {
		log.Fatal(err)
	}
	if err := keyvalues.NewWriter(os.Stdout).Write(merged); err != nil {
		log.Fatal(err)
	}
	// Output:
	// "GameInfo"
	// {
	// 	"game"	"My Mod"
	// 	"type"	"singleplayer_only"
	// 	"developer"	"Me"
	// }
}
