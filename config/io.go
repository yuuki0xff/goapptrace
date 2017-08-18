package config

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
