package konfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
	"sigs.k8s.io/yaml"
)

// ErrNoSources indicates that no configuration sources produced any value.
var ErrNoSources = errors.New("konfig: no configuration sources found")

// Option modifies how Load discovers and applies configuration.
type Option func(*options)

type options struct {
	envPrefix string
	files     []string
	base      string
}

// WithEnvPrefix configures a prefix that is prepended to every generated
// environment variable key. Nested struct names are appended using underscores.
func WithEnvPrefix(prefix string) Option {
	return func(o *options) {
		o.envPrefix = prefix
	}
}

// WithFiles declares additional configuration files to evaluate in the given
// order. Later files in the list can override values from earlier ones.
func WithFiles(files ...string) Option {
	return func(o *options) {
		o.files = append(o.files, files...)
	}
}

// withBase sets the base filename (without extension) used for implicit lookup.
func withBase(base string) Option {
	return func(o *options) {
		o.base = base
	}
}

// Load populates config by reading from the declared files and environment
// variables, returning ErrNoSources when nothing supplies a value. The config
// argument must be a non-nil pointer to a struct (or a struct of structs).
func Load(config interface{}, opts ...Option) error {
	if config == nil {
		return errors.New("konfig: config must not be nil")
	}

	rv := reflect.ValueOf(config)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("konfig: config must be a non-nil pointer")
	}

	cfg := options{}
	for _, opt := range opts {
		opt(&cfg)
	}

	var loaded bool

	if cfg.base != "" {
		baseFiles := []string{
			cfg.base + ".json",
			cfg.base + ".toml",
			cfg.base + ".yaml",
			cfg.base + ".yml",
		}
		baseLoaded, err := loadFirstAvailable(baseFiles, config)
		if err != nil {
			return err
		}
		loaded = loaded || baseLoaded
	}

	if len(cfg.files) > 0 {
		fileLoaded, err := loadSequential(cfg.files, config)
		if err != nil {
			return err
		}
		loaded = loaded || fileLoaded
	}

	applied, err := applyEnvOverrides(rv, cfg.envPrefix)
	if err != nil {
		return err
	}

	if !loaded && applied == 0 {
		return ErrNoSources
	}

	return nil
}

// GetConf preserves the legacy API of resolving a base filename (without
// extension) and populating config based on the first available source.
func GetConf(base string, config interface{}) error {
	return Load(config, withBase(base))
}

// LoadConfigFileNoExt attempts to load configuration using a base filename,
// trying JSON, TOML, then YAML in that order.
func LoadConfigFileNoExt(config interface{}, base string) error {
	return Load(config, withBase(base))
}

// LoadConfigFiles sequentially loads the provided files, allowing later files
// to override earlier ones.
func LoadConfigFiles(config interface{}, files ...string) error {
	return Load(config, WithFiles(files...))
}

// GetConfigFilesWithExt returns the subset of files that exist as regular
// files, preserving the provided order. It returns ErrNoSources if none exist.
func GetConfigFilesWithExt(files ...string) ([]string, error) {
	var matched []string
	for _, file := range files {
		if file == "" {
			continue
		}
		info, err := os.Stat(file)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("konfig: stat %s: %w", file, err)
		}
		if info.Mode().IsRegular() {
			matched = append(matched, file)
		}
	}

	if len(matched) == 0 {
		return nil, ErrNoSources
	}

	return matched, nil
}

// LoadJSON reads and unmarshals a JSON configuration file into configuration.
func LoadJSON(filename string, configuration interface{}) error {
	return decodeFile(filename, configuration, json.Unmarshal)
}

// LoadTOML reads and unmarshals a TOML configuration file into configuration.
func LoadTOML(filename string, configuration interface{}) error {
	return decodeFile(filename, configuration, toml.Unmarshal)
}

// LoadYAML reads and unmarshals a YAML configuration file into configuration.
func LoadYAML(filename string, configuration interface{}) error {
	return decodeFile(filename, configuration, unmarshalYAML)
}

func unmarshalYAML(data []byte, target interface{}) error {
	return yaml.Unmarshal(data, target)
}

func decodeFile(filename string, target interface{}, unmarshal func([]byte, interface{}) error) error {
	if filename == "" {
		return nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("konfig: read %s: %w", filename, err)
	}

	if err := unmarshal(data, target); err != nil {
		return fmt.Errorf("konfig: decode %s: %w", filename, err)
	}

	return nil
}

func loadFirstAvailable(files []string, config interface{}) (bool, error) {
	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}

		data, err := os.ReadFile(file)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return false, fmt.Errorf("konfig: read %s: %w", file, err)
		}

		if err := unmarshalByExtension(file, data, config); err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

func loadSequential(files []string, config interface{}) (bool, error) {
	var loaded bool

	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}

		data, err := os.ReadFile(file)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return loaded, fmt.Errorf("konfig: read %s: %w", file, err)
		}

		if err := unmarshalByExtension(file, data, config); err != nil {
			return loaded, err
		}

		loaded = true
	}

	return loaded, nil
}

func unmarshalByExtension(file string, data []byte, config interface{}) error {
	switch ext := strings.ToLower(filepath.Ext(file)); ext {
	case ".json":
		if err := json.Unmarshal(data, config); err != nil {
			return fmt.Errorf("konfig: decode %s: %w", file, err)
		}
	case ".toml":
		if err := toml.Unmarshal(data, config); err != nil {
			return fmt.Errorf("konfig: decode %s: %w", file, err)
		}
	case ".yaml", ".yml":
		if err := unmarshalYAML(data, config); err != nil {
			return fmt.Errorf("konfig: decode %s: %w", file, err)
		}
	default:
		if err := tryFallbackDecoders(data, config); err != nil {
			return fmt.Errorf("konfig: decode %s: %w", file, err)
		}
	}

	return nil
}

func tryFallbackDecoders(data []byte, config interface{}) error {
	if err := toml.Unmarshal(data, config); err == nil {
		return nil
	}
	if err := json.Unmarshal(data, config); err == nil {
		return nil
	}
	if err := unmarshalYAML(data, config); err == nil {
		return nil
	}
	return errors.New("konfig: failed to decode configuration data")
}

func applyEnvOverrides(rv reflect.Value, prefix string) (int, error) {
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return 0, errors.New("konfig: env overrides require a struct pointer")
	}

	elem := rv.Elem()
	if elem.Kind() != reflect.Struct {
		return 0, errors.New("konfig: env overrides require a pointer to struct")
	}

	return setStructFieldsFromEnv(elem, prefix)
}

func setStructFieldsFromEnv(structValue reflect.Value, prefix string) (int, error) {
	var applied int
	structType := structValue.Type()

	for i := 0; i < structValue.NumField(); i++ {
		fieldType := structType.Field(i)
		if !fieldType.IsExported() {
			continue
		}

		fieldValue := structValue.Field(i)
		key, ok := envKey(fieldType, prefix)
		if !ok {
			continue
		}

		if fieldValue.Kind() == reflect.Struct {
			nestedCount, err := setStructFieldsFromEnv(fieldValue, key)
			if err != nil {
				return applied, err
			}
			applied += nestedCount
			continue
		}

		if fieldValue.Kind() == reflect.Ptr && fieldValue.Type().Elem().Kind() == reflect.Struct {
			if fieldValue.IsNil() {
				fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
			}
			nestedCount, err := setStructFieldsFromEnv(fieldValue.Elem(), key)
			if err != nil {
				return applied, err
			}
			applied += nestedCount
			continue
		}

		value, ok := os.LookupEnv(key)
		if !ok {
			continue
		}

		if err := assignFromString(fieldValue, value); err != nil {
			return applied, fmt.Errorf("konfig: set %s: %w", key, err)
		}

		applied++
	}

	return applied, nil
}

func envKey(field reflect.StructField, prefix string) (string, bool) {
	tag := field.Tag.Get("env")
	if tag == "-" {
		return "", false
	}

	if tag != "" {
		tag = strings.Split(tag, ",")[0]
	}

	name := tag
	if name == "" {
		name = firstNonEmptyTagValue(field, "konfig", "json", "yaml", "toml")
	}
	if name == "" {
		name = field.Name
	}

	key := toEnvKey(name)
	if key == "" {
		return "", false
	}

	if prefix != "" {
		key = prefix + "_" + key
	}

	return key, true
}

func firstNonEmptyTagValue(field reflect.StructField, names ...string) string {
	for _, name := range names {
		tag := field.Tag.Get(name)
		if tag == "" {
			continue
		}
		tag = strings.Split(tag, ",")[0]
		if tag == "-" || tag == "" {
			continue
		}
		return tag
	}
	return ""
}

func toEnvKey(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	var out []rune
	var prevRune rune
	var hasPrev bool

	appendUnderscore := func() {
		if len(out) == 0 || out[len(out)-1] == '_' {
			return
		}
		out = append(out, '_')
	}

	for _, r := range name {
		switch {
		case r == '_' || r == '-' || r == '.' || unicode.IsSpace(r):
			appendUnderscore()
			hasPrev = false
			continue
		case hasPrev && isBoundary(prevRune, r):
			appendUnderscore()
		}

		out = append(out, unicode.ToUpper(r))
		prevRune = r
		hasPrev = true
	}

	// trim leading/trailing underscores and collapse duplicates
	j := 0
	for _, r := range out {
		if r == '_' {
			if j == 0 || out[j-1] == '_' {
				continue
			}
		}
		out[j] = r
		j++
	}
	out = out[:j]
	for len(out) > 0 && out[len(out)-1] == '_' {
		out = out[:len(out)-1]
	}

	return string(out)
}

func isBoundary(prev rune, current rune) bool {
	if unicode.IsLower(prev) && unicode.IsUpper(current) {
		return true
	}
	if unicode.IsDigit(current) && !unicode.IsDigit(prev) {
		return true
	}
	if unicode.IsDigit(prev) && !unicode.IsDigit(current) {
		return true
	}
	return false
}

func assignFromString(field reflect.Value, value string) error {
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}

	if !field.CanSet() {
		return errors.New("field cannot be set")
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Bool:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(value, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		v, err := strconv.ParseUint(value, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(v)
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(value, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetFloat(v)
	default:
		return fmt.Errorf("unsupported kind %s", field.Kind())
	}

	return nil
}
