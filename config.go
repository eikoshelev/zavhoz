package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-yaml/yaml"
)

// Conf - структура конфига
type Conf struct {
	Title  string `yaml:"title"`
	Server struct {
		HTTP struct {
			Port string `yaml:"port"`
		} `yaml:"http"`
		DNS struct {
			Port    string `yaml:"port"`
			Network string `yaml:"network"`
			TTL     uint32 `yaml:"ttl"`
		} `yaml:"dns"`
	} `yaml:"server"`

	Log struct {
		Network  string `yaml:"network"`
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		Type     string `yaml:"type"`
		Debug    bool   `yaml:"debug mode"`
		Severity string `yaml:"severity"`
		Facility string `yaml:"facility"`
		FilePath string `yaml:"file path"`
		FileName string `yaml:"file name"`
	} `yaml:"log"`

	Storage struct {
		Login    string   `yaml:"login"`
		Password string   `yaml:"password"`
		Bucket   string   `yaml:"bucket"`
		Hosts    []string `yaml:"hosts"`
	} `yaml:"storage"`
}

var configFlag = flag.String("config", "./default.yaml", "set config file in the yaml format")

// Config - объявляем переменную для работы с данными конфига
var Config Conf

func configure() Conf {

	configFile, err := ioutil.ReadFile(*configFlag)
	if err != nil {
		fmt.Println("Failed read configuration file: ", err)
		os.Exit(1)
	}

	var c Conf
	err = yaml.Unmarshal(configFile, &c)

	if err != nil {
		fmt.Println("Failed unmarshal ", *configFlag, err)
	}

	return c
}
