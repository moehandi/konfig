package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/moehandi/konfig"
	"log"
	"path/filepath"
	"runtime"
)

type Configuration struct {
	Server   string `json:"Server" yaml:"Server" toml:"Server"`
	Port     int    `json:"Port" yaml:"Port" toml:"Port"`
	Debug    bool   `json:"Debug" yaml:"Debug" toml:"Debug"`
	Database DB	`json:"Database" yaml:"Database" toml:"Database"`
}

type DB struct {
	Type string `json:"Type" yaml:"Type" toml:"Type"`
	Name string `json:"Name" yaml:"Name" toml:"Name"`
	Port int    `json:"Port" yaml:"Port" toml:"Port"`
}

func main() {

	// mock env variable
	//os.Setenv("Server", "test")
	//os.Setenv("Port", "8081")
	//os.Setenv("Enabled", "false")

	myConfig := Configuration{}
	err := konfig.GetConf("config/config", &myConfig)
	if err != nil {
		fmt.Println("os Exit 500", err)
		os.Exit(500)

	}

	log.Println("Server:", myConfig.Server)
	log.Println("Port:", myConfig.Port)
	log.Println("Debug:", myConfig.Debug)
	log.Println("Db Type:", myConfig.Database.Type)
	log.Println("Db Name:", myConfig.Database.Name)
	log.Println("Db Port:", myConfig.Database.Port)

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
