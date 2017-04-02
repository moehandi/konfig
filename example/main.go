package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/moehandi/konfig"
	"path/filepath"
	"runtime"
	"github.com/moehandi/imagehost/Godeps/_workspace/src/github.com/Sirupsen/logrus"
)

type Configuration struct {
	Server  string
	Port    int
	Enabled bool
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

	logrus.Infoln(myConfig.Server)
	logrus.Infoln(myConfig.Port)
	logrus.Infoln(myConfig.Enabled)

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
