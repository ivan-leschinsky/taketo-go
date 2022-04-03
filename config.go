package main

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	User    string `yaml:"user"`
	Command string `yaml:"command"`
}

func readConf(fpath string) (*Config, error) {
	buf, err := ioutil.ReadFile(fpath)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}

	err = yaml.Unmarshal(buf, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
