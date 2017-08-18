package config
// Directory Layout
//   $dir/log/name.jsonl.gz  - gzip compressed log file
//   $dir/targets.json        - includes target, trace, build

type Config struct {
	// TODO
	dir string
}

func NewConfig(dir string) *Config {
	return &Config{} // TODO
}

func (Config) Load() error {
	return nil // TODO
}

func (Config) Save() error {
	return nil // TODO
}
