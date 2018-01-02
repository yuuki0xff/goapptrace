package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"

	"github.com/yuuki0xff/goapptrace/info"
)

// Directory Layout
//   $dir/targets.json        - includes target, trace, build
//   $dir/logs/               - managed under tracer.storage

type Config struct {
	dir      string
	Targets  Targets
	Servers  Servers
	wantSave bool
}

func NewConfig(dir string) *Config {
	if dir == "" {
		dir = info.DEFAULT_CONFIG_DIR
	}

	return &Config{
		dir: dir,
	}
}

func (c *Config) Load() error {
	if _, err := os.Stat(c.targetsPath()); os.IsNotExist(err) {
		c.Targets = *NewTargets()
	} else {
		js, err := ioutil.ReadFile(c.targetsPath())
		if err != nil {
			return err
		}
		if err := json.Unmarshal(js, &c.Targets); err != nil {
			return err
		}
	}
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

	js, err := json.Marshal(c.Targets)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(c.targetsPath(), js, os.ModePerm^0111); err != nil {
		return err
	}

	js, err = json.Marshal(c.Servers)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(c.serversPath(), js, os.ModePerm^0111); err != nil {
		return err
	}
	return nil
}

func (c *Config) SaveIfWant() error {
	if c.wantSave {
		return c.Save()
	}
	return nil
}

func (c Config) targetsPath() string {
	return path.Join(c.dir, "targets.json")
}
func (c Config) serversPath() string {
	return path.Join(c.dir, "servers.json")
}

func (c Config) LogsDir() string {
	return path.Join(c.dir, "logs")
}
