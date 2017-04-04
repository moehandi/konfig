package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/moehandi/konfig"
	"path/filepath"
	"runtime"
	"log"
)

type Configuration struct {
	Server  string `json:"Server" yaml:"Server" toml:"Server"`
	Port    int `json:"Port" yaml:"Port" toml:"Port"`
	Enabled bool `json:"Enabled" yaml:"Enabled" toml:"Enabled"`
}

func main() {

	// mock env variable
	//os.Setenv("Server", "test")
	//os.Setenv("Port", "8081")
	//os.Setenv("Enabled", "false")

	myConfig := Configuration{}
	err := konfig.GetConf("config/config", &myConfig)
	//err := konfig.GetConf("config/config.toml", &myConfig)
	//err := konfig.GetConf(getFileName(), &myConfig)
	if err != nil {
		fmt.Println("os Exit 500", err)
		os.Exit(500)

	}

	log.Println("Server:", myConfig.Server)
	log.Println("Port:", myConfig.Port)
	log.Println("Enabled:", myConfig.Enabled)

}

func getFileName() string {
	env := os.Getenv("ENV")
	if len(env) == 0 {
		env = "development"
	}
	filename := []string{"config.", env, ".json"}
	_, dirname, _, _ := runtime.Caller(0)
	filePath := path.Join(filepath.Dir(dirname), strings.Join(filename, ""))

	return filePath
}
