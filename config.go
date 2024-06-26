package main

type Config struct {
	Timezone string
}

func NewConfig() *Config {
	return &Config{}
}
