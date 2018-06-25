package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/mitchellh/go-homedir"
	"github.com/yuuki0xff/goapptrace/info"
)

const (
	// goapptraceによって作成されたディレクトリのデフォルトのパーミッション
	DefaultDirPerm = 0700
	// goapptraceによって作成されたファイルのデフォルトのパーミッション
	DefaultFilePerm = 0600
	// configファイルのデフォルトのパーミッション
	ConfigFilePerm = 0666
)

type Config struct {
	// Path to config directory.
	dir string
	// API server address.
	apiServer string
	// Log server address.
	logServer string
	wantSave  bool
}

func NewConfig(dir, apiServer, logServer string) *Config {
	if dir == "" {
		dir = info.DefaultConfigDir
	}
	if apiServer == "" {
		apiServer = DefaultApiServerAddr
	}
	if logServer == "" {
		logServer = DefaultLogServerAddr
	}

	dir, err := homedir.Expand(dir)
	if err != nil {
		log.Panic(err)
	}
	return &Config{
		dir:       dir,
		apiServer: apiServer,
		logServer: logServer,
	}
}

func (c *Config) Load() error {
	return nil
}

func (c *Config) WantSave() {
	c.wantSave = true
}

func (c *Config) Save() error {
	if _, err := os.Stat(c.dir); os.IsNotExist(err) {
		if err := os.MkdirAll(c.dir, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) SaveIfWant() error {
	if c.wantSave {
		return c.Save()
	}
	return nil
}

func (c Config) ApiServer() string {
	return c.apiServer
}

func (c Config) LogServer() string {
	return c.logServer
}

func (c Config) LogsDir() string {
	return path.Join(c.dir, "logs")
}

func readFromJsonFile(filepath string, data interface{}) error {
	js, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}
	return json.Unmarshal(js, data)
}
func writeToJsonFile(filepath string, data interface{}) error {
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath, js, ConfigFilePerm)
}
