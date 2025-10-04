package konfig

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type sampleDB struct {
	Type string
	Port int
}

type sampleConfig struct {
	Server   string
	Port     int
	Debug    bool
	Timeout  *int
	Database sampleDB
}

func TestGetConfUsesFirstAvailableFile(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "config/app")

	mustWrite(t, base+".toml", "Server = \"from-toml\"\nPort = 4000\n\n[Database]\nType = \"postgres\"\nPort = 5432\n")
	mustWrite(t, base+".yaml", "Server: from-yaml\nPort: 5000\nDatabase:\n  Type: mysql\n  Port: 3306\n")

	var cfg sampleConfig
	if err := GetConf(filepath.Join(dir, "config/app"), &cfg); err != nil {
		t.Fatalf("GetConf returned error: %v", err)
	}

	if cfg.Server != "from-toml" {
		t.Fatalf("expected server from toml, got %q", cfg.Server)
	}
	if cfg.Port != 4000 {
		t.Fatalf("expected port 4000, got %d", cfg.Port)
	}
	if cfg.Database.Type != "postgres" {
		t.Fatalf("expected database type postgres, got %q", cfg.Database.Type)
	}
}

func TestLoadSequentialFilesOverride(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.json")
	second := filepath.Join(dir, "second.yaml")

	mustWrite(t, first, `{"Server":"first","Port":1000,"Database":{"Type":"mysql","Port":3306}}`)
	mustWrite(t, second, "Server: second\nPort: 2000\nDatabase:\n  Type: postgres\n  Port: 5432\n")

	var cfg sampleConfig
	if err := Load(&cfg, WithFiles(first, second)); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server != "second" {
		t.Fatalf("expected override from second file, got %q", cfg.Server)
	}
	if cfg.Port != 2000 {
		t.Fatalf("expected override port 2000, got %d", cfg.Port)
	}
	if cfg.Database.Port != 5432 {
		t.Fatalf("expected override database port 5432, got %d", cfg.Database.Port)
	}
}

func TestLoadEnvironmentOverrides(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "service/config")

	mustWrite(t, base+".json", `{"Server":"file","Port":80,"Database":{"Type":"mysql","Port":3306}}`)

	// Environment overrides
	t.Setenv("APP_SERVER", "env")
	t.Setenv("APP_DATABASE_PORT", "7777")
	timeout := "15"
	t.Setenv("APP_TIMEOUT", timeout)

	type config struct {
		Server   string
		Port     int
		Timeout  *int
		Database struct {
			Type string
			Port int
		}
	}

	var cfg config
	if err := Load(&cfg, withBase(base), WithEnvPrefix("APP")); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server != "env" {
		t.Fatalf("expected server from env, got %q", cfg.Server)
	}
	if cfg.Database.Port != 7777 {
		t.Fatalf("expected database port override 7777, got %d", cfg.Database.Port)
	}
	if cfg.Timeout == nil || *cfg.Timeout != 15 {
		t.Fatalf("expected timeout pointer set to 15, got %v", cfg.Timeout)
	}
}

func TestLoadEnvironmentOnly(t *testing.T) {
	t.Setenv("CONFIG_SERVER", "only-env")

	var cfg struct {
		Server string
	}

	if err := Load(&cfg, WithEnvPrefix("CONFIG")); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server != "only-env" {
		t.Fatalf("expected env value, got %q", cfg.Server)
	}
}

func TestLoadNoSources(t *testing.T) {
	var cfg struct {
		Server string
	}

	err := Load(&cfg)
	if !errors.Is(err, ErrNoSources) {
		t.Fatalf("expected ErrNoSources, got %v", err)
	}
}

func TestGetConfigFilesWithExt(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "config.json")
	mustWrite(t, existing, "{}")

	files, err := GetConfigFilesWithExt(existing, filepath.Join(dir, "missing.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 1 || files[0] != existing {
		t.Fatalf("expected only existing file, got %#v", files)
	}
}

func TestGetConfigFilesWithExtNone(t *testing.T) {
	if _, err := GetConfigFilesWithExt("not-there.json"); !errors.Is(err, ErrNoSources) {
		t.Fatalf("expected ErrNoSources, got %v", err)
	}
}

func TestGetConfigFilesWithExtIgnoresEmpty(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "config.json")
	mustWrite(t, file, "{}")

	files, err := GetConfigFilesWithExt("", file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 1 || files[0] != file {
		t.Fatalf("expected only real file, got %#v", files)
	}
}

func TestGetConfigFilesWithExtStatError(t *testing.T) {
	dir := t.TempDir()
	restricted := filepath.Join(dir, "restricted")
	if err := os.MkdirAll(restricted, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	file := filepath.Join(restricted, "config.json")
	mustWrite(t, file, "{}")

	if err := os.Chmod(restricted, 0); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer os.Chmod(restricted, 0o755)

	if _, err := GetConfigFilesWithExt(file); err == nil || !strings.Contains(err.Error(), "stat") {
		t.Fatalf("expected stat error, got %v", err)
	}
}

func TestLoadValidationErrors(t *testing.T) {
	if err := Load(nil); err == nil || !strings.Contains(err.Error(), "must not be nil") {
		t.Fatalf("expected nil config error, got %v", err)
	}

	if err := Load(struct{}{}); err == nil || !strings.Contains(err.Error(), "non-nil pointer") {
		t.Fatalf("expected pointer error, got %v", err)
	}

	var notStruct = new(int)
	if err := Load(notStruct); err == nil || !strings.Contains(err.Error(), "pointer to struct") {
		t.Fatalf("expected struct pointer error, got %v", err)
	}
}

func TestLoadSequentialReadError(t *testing.T) {
	dir := t.TempDir()
	badPath := filepath.Join(dir, "dir")
	if err := os.MkdirAll(badPath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	var cfg struct{}
	err := Load(&cfg, WithFiles(badPath))
	if err == nil || !strings.Contains(err.Error(), "read") {
		t.Fatalf("expected read error, got %v", err)
	}
}

func TestLoadSequentialIgnoresBlank(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "config.json")
	mustWrite(t, valid, `{"Server":"ok"}`)

	var cfg struct{ Server string }
	loaded, err := loadSequential([]string{"   ", valid}, &cfg)
	if err != nil {
		t.Fatalf("loadSequential error: %v", err)
	}
	if !loaded {
		t.Fatalf("expected loaded true")
	}
	if cfg.Server != "ok" {
		t.Fatalf("expected ok, got %q", cfg.Server)
	}
}

func TestLoadFirstAvailableSkipsBlank(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "config.json")
	mustWrite(t, valid, `{"Server":"ok"}`)

	var cfg struct{ Server string }
	loaded, err := loadFirstAvailable([]string{"   ", valid}, &cfg)
	if err != nil {
		t.Fatalf("loadFirstAvailable error: %v", err)
	}
	if !loaded {
		t.Fatalf("expected loaded true")
	}
	if cfg.Server != "ok" {
		t.Fatalf("expected ok, got %q", cfg.Server)
	}
}

func TestLoadFallbackDecoder(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "app.cfg")
	mustWrite(t, configPath, `{"Server":"fallback"}`)

	var cfg struct {
		Server string
	}

	if err := Load(&cfg, WithFiles(configPath)); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server != "fallback" {
		t.Fatalf("expected fallback value, got %q", cfg.Server)
	}
}

func TestLoadFallbackDecoderToml(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "app.cfg")
	mustWrite(t, configPath, "Server = \"toml\"\n")

	var cfg struct{ Server string }
	if err := Load(&cfg, WithFiles(configPath)); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server != "toml" {
		t.Fatalf("expected toml fallback, got %q", cfg.Server)
	}
}

func TestLoadFallbackDecoderYAML(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "app.cfg")
	mustWrite(t, configPath, "Server: yaml\n")

	var cfg struct{ Server string }
	if err := Load(&cfg, WithFiles(configPath)); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server != "yaml" {
		t.Fatalf("expected yaml fallback, got %q", cfg.Server)
	}
}

func TestLoadFallbackDecoderError(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "bad.cfg")
	mustWrite(t, configPath, "not valid")

	var cfg struct{}
	err := Load(&cfg, WithFiles(configPath))
	if err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestLoadSequentialDecodeErrors(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name string
		data []byte
	}{
		{"bad.json", []byte("{bad")},
		{"bad.toml", []byte("key = \"value")},
		{"bad.yaml", []byte("list: [1,")},
	}

	for _, tc := range cases {
		file := filepath.Join(dir, tc.name)
		if err := os.WriteFile(file, tc.data, 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}

		var cfg struct{}
		if err := Load(&cfg, WithFiles(file)); err == nil || !strings.Contains(err.Error(), "decode") {
			t.Fatalf("expected decode error for %s, got %v", tc.name, err)
		}
	}
}

func TestLoadJSONError(t *testing.T) {
	if err := LoadJSON("missing.json", &struct{}{}); err == nil {
		t.Fatalf("expected error reading missing file")
	}
}

func TestLoadJSONEmptyFilename(t *testing.T) {
	if err := LoadJSON("", &struct{}{}); err != nil {
		t.Fatalf("expected nil error for empty filename, got %v", err)
	}
}

func TestEnvKeyPrecedence(t *testing.T) {
	type inner struct {
		Value string `json:"custom_value"`
	}

	type outer struct {
		Skip   string `env:"-"`
		Tagged string `env:"EXPLICIT"`
		Custom string `konfig:"config-name"`
		Inner  inner
		NoTag  string
	}

	fields := reflect.VisibleFields(reflect.TypeOf(outer{}))

	got := map[string]string{}
	for _, field := range fields {
		key, ok := envKey(field, "APP")
		if ok {
			got[field.Name] = key
		}
	}

	if _, ok := got["Skip"]; ok {
		t.Fatalf("expected Skip to be omitted")
	}

	if got["Tagged"] != "APP_EXPLICIT" {
		t.Fatalf("expected explicit tag, got %q", got["Tagged"])
	}

	if got["Custom"] != "APP_CONFIG_NAME" {
		t.Fatalf("expected konfig tag, got %q", got["Custom"])
	}

	if got["Inner"] != "APP_INNER" {
		t.Fatalf("expected inner struct key, got %q", got["Inner"])
	}

	if got["NoTag"] != "APP_NO_TAG" {
		t.Fatalf("expected default key, got %q", got["NoTag"])
	}
}

func TestToEnvKey(t *testing.T) {
	tests := map[string]string{
		"DatabaseURL":     "DATABASE_URL",
		"ssl-enabled":     "SSL_ENABLED",
		"  spaced name  ": "SPACED_NAME",
		"HTTP2Port":       "HTTP_2_PORT",
		"snake_case":      "SNAKE_CASE",
	}

	for input, expected := range tests {
		if got := toEnvKey(input); got != expected {
			t.Fatalf("toEnvKey %q: expected %q, got %q", input, expected, got)
		}
	}
}

func TestAssignFromStringConversions(t *testing.T) {
	type sample struct {
		S   string
		B   bool
		I   int
		U   uint
		F   float64
		PI  *int
		PF  *float32
		Bad []string
	}

	var s sample
	val := reflect.ValueOf(&s).Elem()

	if err := assignFromString(val.FieldByName("S"), "hello"); err != nil {
		t.Fatalf("assign string: %v", err)
	}
	if err := assignFromString(val.FieldByName("B"), "true"); err != nil {
		t.Fatalf("assign bool: %v", err)
	}
	if err := assignFromString(val.FieldByName("I"), "42"); err != nil {
		t.Fatalf("assign int: %v", err)
	}
	if err := assignFromString(val.FieldByName("U"), "7"); err != nil {
		t.Fatalf("assign uint: %v", err)
	}
	if err := assignFromString(val.FieldByName("F"), "3.14"); err != nil {
		t.Fatalf("assign float: %v", err)
	}
	if err := assignFromString(val.FieldByName("PI"), "21"); err != nil {
		t.Fatalf("assign pointer int: %v", err)
	}
	if err := assignFromString(val.FieldByName("PF"), "2.5"); err != nil {
		t.Fatalf("assign pointer float: %v", err)
	}

	if s.S != "hello" || !s.B || s.I != 42 || s.U != 7 || s.F != 3.14 {
		t.Fatalf("unexpected assigned values: %+v", s)
	}
	if s.PI == nil || *s.PI != 21 {
		t.Fatalf("expected pointer int set, got %v", s.PI)
	}
	if s.PF == nil || *s.PF != 2.5 {
		t.Fatalf("expected pointer float set, got %v", s.PF)
	}

	if err := assignFromString(val.FieldByName("Bad"), "oops"); err == nil {
		t.Fatalf("expected unsupported kind error")
	}
}

func TestApplyEnvOverridesErrors(t *testing.T) {
	var ptr *struct{}
	if _, err := applyEnvOverrides(reflect.ValueOf(ptr), ""); err == nil {
		t.Fatalf("expected error on nil pointer")
	}

	var notStruct = new(int)
	if _, err := applyEnvOverrides(reflect.ValueOf(notStruct), ""); err == nil {
		t.Fatalf("expected error for non-struct pointer")
	}
}

func TestEnvOverrideSkipsUnexported(t *testing.T) {
	type cfg struct {
		visible string
		Value   string
	}

	t.Setenv("VALUE", "set")
	t.Setenv("VISIBLE", "should-not-set")

	var c cfg
	if err := Load(&c); err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if c.visible != "" {
		t.Fatalf("expected unexported field to remain empty, got %q", c.visible)
	}
	if c.Value != "set" {
		t.Fatalf("expected exported value set, got %q", c.Value)
	}
}

func TestEnvKeyEmpty(t *testing.T) {
	type cfg struct {
		Ignored string `env:"   "`
		Value   string
	}

	f, _ := reflect.TypeOf(cfg{}).FieldByName("Ignored")
	if key, ok := envKey(f, ""); ok || key != "" {
		t.Fatalf("expected envKey to skip blank tag, got %q", key)
	}

	t.Setenv("VALUE", "set")
	var c cfg
	if err := Load(&c); err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if c.Value != "set" {
		t.Fatalf("expected value set, got %q", c.Value)
	}
}

func TestEnvOverrideAssignError(t *testing.T) {
	type cfg struct {
		Values []string
	}

	t.Setenv("CFG_VALUES", "oops")

	var c cfg
	if err := Load(&c, WithEnvPrefix("CFG")); err == nil || !strings.Contains(err.Error(), "unsupported kind") {
		t.Fatalf("expected unsupported kind error, got %v", err)
	}
}

func TestEnvOverrideParseErrors(t *testing.T) {
	type cfg struct {
		Bool  bool
		Int   int
		Uint  uint
		Float float64
	}

	t.Setenv("CFG_BOOL", "nope")
	t.Setenv("CFG_INT", "not-int")
	t.Setenv("CFG_UINT", "-1")
	t.Setenv("CFG_FLOAT", "nan?no")

	var c cfg
	err := Load(&c, WithEnvPrefix("CFG"))
	if err == nil {
		t.Fatalf("expected parse error")
	}

	if !strings.Contains(err.Error(), "CFG_BOOL") {
		t.Fatalf("expected error to reference CFG_BOOL, got %v", err)
	}
}

func TestLoadConfigFileNoExt(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "settings/app")

	mustWrite(t, base+".yaml", "Server: yaml\n")

	var cfg struct{ Server string }
	if err := LoadConfigFileNoExt(&cfg, base); err != nil {
		t.Fatalf("LoadConfigFileNoExt error: %v", err)
	}

	if cfg.Server != "yaml" {
		t.Fatalf("expected yaml server, got %q", cfg.Server)
	}
}

func TestLoadConfigFilesHelper(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.json")
	second := filepath.Join(dir, "second.json")

	mustWrite(t, first, `{"Server":"first"}`)
	mustWrite(t, second, `{"Server":"second"}`)

	var cfg struct{ Server string }
	if err := LoadConfigFiles(&cfg, first, second); err != nil {
		t.Fatalf("LoadConfigFiles error: %v", err)
	}

	if cfg.Server != "second" {
		t.Fatalf("expected override from second, got %q", cfg.Server)
	}
}

func TestLoadConfigFilesSkipsMissing(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "valid.json")
	mustWrite(t, valid, `{"Server":"ok"}`)

	var cfg struct{ Server string }
	if err := Load(&cfg, WithFiles(filepath.Join(dir, "missing.json"), valid)); err != nil {
		t.Fatalf("Load with missing file failed: %v", err)
	}

	if cfg.Server != "ok" {
		t.Fatalf("expected ok, got %q", cfg.Server)
	}
}

func TestLoadFirstAvailableError(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "app/config")
	if err := os.MkdirAll(base+".json", 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	var cfg struct{}
	err := GetConf(filepath.Join(dir, "app/config"), &cfg)
	if err == nil || !strings.Contains(err.Error(), "read") {
		t.Fatalf("expected read error from directory base, got %v", err)
	}
}

func TestLoadFirstAvailableDecodeError(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "app/config")
	mustWrite(t, base+".json", "{bad json}")

	var cfg struct{}
	err := GetConf(filepath.Join(dir, "app/config"), &cfg)
	if err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestLoadBaseMissingUsesEnv(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "config/app")

	type cfg struct {
		Server string
	}

	t.Setenv("APP_SERVER", "from-env")

	var c cfg
	if err := Load(&c, withBase(base), WithEnvPrefix("APP")); err != nil {
		t.Fatalf("Load unexpectedly failed: %v", err)
	}

	if c.Server != "from-env" {
		t.Fatalf("expected env value, got %q", c.Server)
	}
}

func TestLoadJSONSuccess(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "config.json")
	mustWrite(t, file, `{"Server":"json"}`)

	var cfg struct{ Server string }
	if err := LoadJSON(file, &cfg); err != nil {
		t.Fatalf("LoadJSON error: %v", err)
	}

	if cfg.Server != "json" {
		t.Fatalf("expected json, got %q", cfg.Server)
	}
}

func TestLoadJSONDecodeError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "bad.json")
	mustWrite(t, file, "{bad json}")

	if err := LoadJSON(file, &struct{}{}); err == nil {
		t.Fatalf("expected decode error")
	}
}

func TestLoadTOMLSuccess(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "config.toml")
	mustWrite(t, file, "Server = \"toml\"\n")

	var cfg struct{ Server string }
	if err := LoadTOML(file, &cfg); err != nil {
		t.Fatalf("LoadTOML error: %v", err)
	}

	if cfg.Server != "toml" {
		t.Fatalf("expected toml, got %q", cfg.Server)
	}
}

func TestLoadYAMLSuccess(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "config.yaml")
	mustWrite(t, file, "Server: yaml\n")

	var cfg struct{ Server string }
	if err := LoadYAML(file, &cfg); err != nil {
		t.Fatalf("LoadYAML error: %v", err)
	}

	if cfg.Server != "yaml" {
		t.Fatalf("expected yaml, got %q", cfg.Server)
	}
}

func TestApplyEnvOverridesPointerStruct(t *testing.T) {
	type nested struct {
		Value string
	}

	type cfg struct {
		Nested *nested
	}

	t.Setenv("CFG_NESTED_VALUE", "from-env")

	var c cfg
	if err := Load(&c, WithEnvPrefix("CFG")); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if c.Nested == nil || c.Nested.Value != "from-env" {
		t.Fatalf("expected nested pointer populated, got %#v", c.Nested)
	}
}

func TestEnvOverridesNoPrefix(t *testing.T) {
	type cfg struct {
		Value string
	}

	t.Setenv("VALUE", "no-prefix")

	var c cfg
	if err := Load(&c); err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if c.Value != "no-prefix" {
		t.Fatalf("expected env without prefix, got %q", c.Value)
	}
}

func mustWrite(t *testing.T, filename, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
}
