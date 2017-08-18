package config

import (
	"encoding/json"
	"github.com/yuuki0xff/goapptrace/info"
	"io/ioutil"
	"os"
	"path"
)

// Directory Layout
//   $dir/log/name.jsonl.gz  - gzip compressed log file
//   $dir/targets.json        - includes target, trace, build

type Config struct {
	// TODO
	dir     string
	Targets Targets
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

func (c *Config) Save() error {
	if _, err := os.Stat(c.dir); os.IsNotExist(err) {
		os.MkdirAll(c.dir, 0660)
	}

	js, err := json.Marshal(c.Targets)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(c.targetsPath(), js, 0660); err != nil {
		return err
	}
	return nil
}

func (c Config) targetsPath() string {
	return path.Join(c.dir, "targets.json")
}
