package konfig

import (
	"os"
	"reflect"
	"strconv"
	"github.com/Sirupsen/logrus"

	"io"
	"io/ioutil"
	"encoding/json"
	"github.com/BurntSushi/toml"
	"fmt"
	"gopkg.in/yaml.v2"
)

var konfigs interface{}

func GetConf(filename string, configuration interface{}) error {
	var err error
	logrus.Infoln("Scanning config files...")
	//var err error = nil

	if konfigs != nil {
		configuration = konfigs
		return nil
	}

	// Prioritize to check config environment first
	status := GetENVConfig(configuration)

	//err = GetTOMLConfig(filename, configuration)
	if status == "no_env" {
		err = GetJSONConfig(filename+".json", configuration)
	}

	if err != nil {
		err = GetTOMLConfig(filename+".toml", configuration)
	}

	if err != nil {
		err = GetYAMLConfig(filename+".yaml", configuration)
	}

	konfigs = configuration

	return nil
}

func GetJSONConfig(filename string, configuration interface{}) error {
	logrus.Infoln("load config from:", filename)
	if len(filename) == 0 {
		return nil
	}

	var err error
	var input = io.ReadCloser(os.Stdin)
	if input, err = os.Open(filename); err != nil {
		logrus.Warnln("Open file:", err)
		return err
	}

	// read the config file
	jsonBytes, err := ioutil.ReadAll(input)
	input.Close()
	if err != nil {
		logrus.Warnln("ioutil err", err)
		return err
	}

	err = json.Unmarshal(jsonBytes, configuration)
	if err != nil {
		logrus.Warnln("cannot parse json", filename, err)
		return err
	}

	return nil
}

func GetTOMLConfig(filename string, configuration interface{}) error {
	logrus.Infoln("load config from:", filename)
	if len(filename) == 0 {
		return nil
	}

	var err error
	var input = io.ReadCloser(os.Stdin)
	if input, err = os.Open(filename); err != nil {
		logrus.Warnln("Open file:", err)
		return err
	}

	// read the config file
	tomlBytes, err := ioutil.ReadAll(input)
	input.Close()
	if err != nil {
		logrus.Warnln("ioutil err", err)
		return err
	}

	err = toml.Unmarshal(tomlBytes, configuration)
	if err != nil {
		logrus.Warnln("cannot parse toml", filename, err)
		return err
	}

	return nil
}

func GetYAMLConfig(filename string, configuration interface{}) error {
	logrus.Infoln("load config from:", filename)
	if len(filename) == 0 {
		return nil
	}

	var err error
	var input = io.ReadCloser(os.Stdin)
	if input, err = os.Open(filename); err != nil {
		logrus.Warnln("Open file:", err)
		return err
	}

	// read the config file
	yamlBytes, err := ioutil.ReadAll(input)
	input.Close()
	if err != nil {
		logrus.Warnln("ioutil err", err)
		return err
	}

	err = yaml.Unmarshal(yamlBytes, configuration)
	if err != nil {
		logrus.Warnln("cannot parse yaml", filename, err)
		return err
	}

	return nil
}


func GetENVConfig(configuration interface{}) string {

	logrus.Infoln("Loading config from Environment...")
	typ := reflect.TypeOf(configuration)
	// if a pointer to a struct is passed, get the type of the derefference object
	if typ.Kind() == reflect.Ptr {
		fmt.Println("reflect.Ptr")
		typ = typ.Elem()
	}

	if os.Getenv(typ.Field(0).Name) == "" {
		logrus.Warnln("No environment value")
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
							setStringToInt(f, value, 64)
						} else if kind == reflect.Int32 {
							setStringToInt(f, value, 32)
						} else if kind == reflect.Int16 {
							setStringToInt(f, value, 16)
						} else if kind == reflect.Uint || kind == reflect.Uint64 {
							setStringToUInt(f, value, 64)
						} else if kind == reflect.Uint32 {
							setStringToUInt(f, value, 32)
						} else if kind == reflect.Uint16 {
							setStringToUInt(f, value, 16)
						} else if kind == reflect.Bool {
							setStringToBool(f, value)
						} else if kind == reflect.Float64 {
							setStringToFloat(f, value, 64)
						} else if kind == reflect.Float32 {
							setStringToFloat(f, value, 32)
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

func setStringToInt(f reflect.Value, value string, bitSize int) {
	convertedValue, err := strconv.ParseInt(value, 10, bitSize)

	if err == nil {
		if !f.OverflowInt(convertedValue) {
			f.SetInt(convertedValue)
		}
	}
}

func setStringToUInt(f reflect.Value, value string, bitSize int) {
	convertedValue, err := strconv.ParseUint(value, 10, bitSize)

	if err == nil {
		if !f.OverflowUint(convertedValue) {
			f.SetUint(convertedValue)
		}
	}
}

func setStringToBool(f reflect.Value, value string) {
	convertedValue, err := strconv.ParseBool(value)

	if err == nil {
		f.SetBool(convertedValue)
	}
}

func setStringToFloat(f reflect.Value, value string, bitSize int) {
	convertedValue, err := strconv.ParseFloat(value, bitSize)

	if err == nil {
		if !f.OverflowFloat(convertedValue) {
			f.SetFloat(convertedValue)
		}
	}
}