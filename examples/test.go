package main

import (
	"github.com/galaco/KeyValues"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func main() {
	Vmt()
	Vmf()
	GameInfo()
}

// Vmt creates KeyValues for all files in vmt dir
func Vmt() {
	samplesDir := "./vmt/"
	fileInfos := getFilenames(samplesDir)
	read(samplesDir, fileInfos, func(filename string, value *keyvalues.KeyValue) {
		log.Println(value.Children())
	})
}

// Vmf creates KeyValues for all files in vmf dir
func Vmf() {
	samplesDir := "./vmf/"
	fileInfos := getFilenames(samplesDir)
	read(samplesDir, fileInfos, func(filename string, value *keyvalues.KeyValue) {
		log.Println(value.Children())
	})
}

// GameInfo creates KeyValues for all files in gameinfo dir
func GameInfo() {
	samplesDir := "./gameinfo/"
	fileInfos := getFilenames(samplesDir)
	read(samplesDir, fileInfos, func(filename string, value *keyvalues.KeyValue) {
		log.Println(value.Children())
	})
}

func read(basePath string, fileInfos []os.FileInfo, callback func(filename string, value *keyvalues.KeyValue)) {
	for _, info := range fileInfos {
		if strings.HasPrefix(info.Name(), ".") {
			continue
		}
		f, err := os.Open(basePath + info.Name())
		if err != nil {
			log.Fatal(err)
		}
		reader := keyvalues.NewReader(f)
		kv, err := reader.Read()

		callback(basePath+info.Name(), &kv)

		f.Close()
	}
}

func getFilenames(dir string) []os.FileInfo {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	return files
}
