package konfig

import (
	"os"
	"reflect"
	"strconv"

	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/ghodss/yaml"
	"io"
	"io/ioutil"
	"log"
	//"gopkg.in/yaml.v2"
	"errors"
	"strings"
)

var konfigs interface{}


func GetConf(filename string, config interface{}) error {
	var err error
	log.Println("Scanning config files...")

	if konfigs != nil {
		config = konfigs
		return nil
	}

	// Prioritize to check config environment first
	status := GetENVConfig(config)

	//err = GetTOMLConfig(filename, config)
	if status == "no_env" {
		err = parseConfig(filename+".json", config)
	}

	if err != nil {
		err = parseConfig(filename+".toml", config)
	}

	if err != nil {
		err = parseConfig(filename+".yaml", config)
	}

	konfigs = config

	return nil
}

// Load certains config file like (json, toml, yaml) doesn't provide extension just give it file name.
// it will autodetect for failed, error, and not found file and then resume to next precedence if available
// it take precedence json -> toml -> yaml
func LoadConfigFileNoExt(config interface{}, fileName string) error {
	log.Println("Scanning config files...")

	if konfigs != nil {
		config = konfigs
		return nil
	}

	if err := LoadJSON(fileName+".json", config); err != nil {
		if err := LoadTOML(fileName+".toml", config); err != nil {
			if err := LoadYAML(fileName+".yaml", config); err != nil {
				log.Println("FAILED through the config", fileName, "(json,toml,yaml)")
				return err
			}
		}
	}

	konfigs = config

	return nil
}

func processFile(config interface{}, file string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	switch {
	case strings.HasSuffix(file, ".yaml") || strings.HasSuffix(file, ".yml"):
		return yaml.Unmarshal(data, config)
	case strings.HasSuffix(file, ".toml"):
		return toml.Unmarshal(data, config)
	case strings.HasSuffix(file, ".json"):
		return json.Unmarshal(data, config)
	default:
		if toml.Unmarshal(data, config) != nil {
			if json.Unmarshal(data, config) != nil {
				if yaml.Unmarshal(data, config) != nil {
					return errors.New("failed to decode config")
				}
			}
		}
		return nil
	}
}

func LoadConfigFiles(config interface{}, files ...string) error {
	//var fileList []string
	_, fileList := GetConfigFilesWithExt(files...)
	//fmt.Println(err)
	for _, file := range fileList {
		if err := processFile(config, file); err != nil {
			return err
		}
	}
	return nil
}

func parseConfig(file string, config interface{}) error{
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	switch {
	case strings.HasSuffix(file, ".yaml") || strings.HasSuffix(file, ".yml"):
		return yaml.Unmarshal(data, config)
	//return LoadYAMLConfig(file, config)
	case strings.HasSuffix(file, ".toml"):
		return toml.Unmarshal(data, config)

	case strings.HasSuffix(file, ".json"):
		return json.Unmarshal(data, config)
	default:
		if toml.Unmarshal(data, config) != nil {
			if json.Unmarshal(data, config) != nil {
				if yaml.Unmarshal(data, config) != nil {
					return errors.New("failed to decode config")
				}
			}
		}
		return nil
	}
}

func LoadJSON(filename string, configuration interface{}) error {
	log.Println("load config from:", filename)
	if len(filename) == 0 {
		return nil
	}

	var err error
	var input = io.ReadCloser(os.Stdin)
	if input, err = os.Open(filename); err != nil {
		log.Println("Open file:", err)
		return err
	}

	// read the config file
	jsonBytes, err := ioutil.ReadAll(input)
	input.Close()
	if err != nil {
		log.Println("ioutil err", err)
		return err
	}

	err = json.Unmarshal(jsonBytes, configuration)
	if err != nil {
		log.Println("cannot parse json", filename, err)
		return err
	}

	return nil
}

func LoadTOML(filename string, configuration interface{}) error {
	log.Println("load config from:", filename)
	if len(filename) == 0 {
		return nil
	}

	var err error
	var input = io.ReadCloser(os.Stdin)
	if input, err = os.Open(filename); err != nil {
		log.Println("Open file:", err)
		return err
	}

	// read the config file
	tomlBytes, err := ioutil.ReadAll(input)
	input.Close()
	if err != nil {
		log.Println("ioutil err", err)
		return err
	}

	err = toml.Unmarshal(tomlBytes, configuration)
	if err != nil {
		log.Println("cannot parse toml", filename, err)
		return err
	}

	return nil
}

func LoadYAML(filename string, configuration interface{}) error {
	log.Println("load config from:", filename)
	if len(filename) == 0 {
		return nil
	}

	var err error
	var input = io.ReadCloser(os.Stdin)
	if input, err = os.Open(filename); err != nil {
		log.Println("Open file:", err)
		return err
	}

	// read the config file
	yamlBytes, err := ioutil.ReadFile(filename)
	input.Close()
	if err != nil {
		log.Println("ioutil err", err)
		return err
	}

	err = yaml.Unmarshal(yamlBytes, configuration)
	if err != nil {
		log.Println("cannot parse yaml", filename, err)
		return err
	}

	return nil
}

func GetENVConfig(configuration interface{}) string {

	log.Println("Loading config from Environment...")
	typ := reflect.TypeOf(configuration)
	// if a pointer to a struct is passed, get the type of the derefference object
	if typ.Kind() == reflect.Ptr {
		fmt.Println("reflect.Ptr")
		typ = typ.Elem()
	}

	if os.Getenv(typ.Field(0).Name) == "" {
		log.Println("No environment value")
		return "no_env"
	}

	for i := 0; i < typ.NumField(); i++ {
		p := typ.Field(i)
		value := os.Getenv(p.Name)
		if !p.Anonymous && len(value) > 0 {
			//fmt.Println("!p.Anonymous")
			// struct
			s := reflect.ValueOf(configuration).Elem()

			if s.Kind() == reflect.Struct {
				// exported field
				f := s.FieldByName(p.Name)
				if f.IsValid() {
					// A Value can be changed only if it is
					// addressable and was not obtained by
					// the use of unexported struct fields.
					if f.CanSet() {
						// change value
						kind := f.Kind()
						if kind == reflect.Int || kind == reflect.Int64 {
							StringToInt(f, value, 64)
						} else if kind == reflect.Int32 {
							StringToInt(f, value, 32)
						} else if kind == reflect.Int16 {
							StringToInt(f, value, 16)
						} else if kind == reflect.Uint || kind == reflect.Uint64 {
							StringToUInt(f, value, 64)
						} else if kind == reflect.Uint32 {
							StringToUInt(f, value, 32)
						} else if kind == reflect.Uint16 {
							StringToUInt(f, value, 16)
						} else if kind == reflect.Bool {
							StringToBool(f, value)
						} else if kind == reflect.Float64 {
							StringToFloat(f, value, 64)
						} else if kind == reflect.Float32 {
							StringToFloat(f, value, 32)
						} else if kind == reflect.String {
							f.SetString(value)
						}
					}
				}
			}
		}
	}
	return "env"
}

func StringToInt(f reflect.Value, value string, bitSize int) {
	convertedValue, err := strconv.ParseInt(value, 10, bitSize)

	if err == nil {
		if !f.OverflowInt(convertedValue) {
			f.SetInt(convertedValue)
		}
	}
}

func StringToUInt(f reflect.Value, value string, bitSize int) {
	convertedValue, err := strconv.ParseUint(value, 10, bitSize)

	if err == nil {
		if !f.OverflowUint(convertedValue) {
			f.SetUint(convertedValue)
		}
	}
}

func StringToBool(f reflect.Value, value string) {
	convertedValue, err := strconv.ParseBool(value)

	if err == nil {
		f.SetBool(convertedValue)
	}
}

func StringToFloat(f reflect.Value, value string, bitSize int) {
	convertedValue, err := strconv.ParseFloat(value, bitSize)

	if err == nil {
		if !f.OverflowFloat(convertedValue) {
			f.SetFloat(convertedValue)
		}
	}
}

func GetConfigFilesWithExt(files ...string) (error, []string){
	var fileList []string

	for i := len(files) - 1; i >=0; i-- {
		found := false
		file := files[i]

		if fileInfo, err := os.Stat(file); err == nil && fileInfo.Mode().IsRegular() {
			found = true
			fileList = append(fileList, file)
			fmt.Println("file found:", fileList)
		}

		if !found {
			//err := error
			fmt.Println(fileList)
			return errors.New("Not found configuration files"), nil
		}
	}

	return nil, fileList
}