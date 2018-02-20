package main

import (
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/go-yaml/yaml"
)

type Conf struct {
	Title  string `yaml:"title"`
	Server struct {
		Http struct {
			Port string `yaml:"port"`
		} `yaml:"http"`
		Dns struct {
			Port    string `yaml:"port"`
			Network string `yaml:"network"`
			Ttl     uint32 `yaml:"ttl"`
		} `yaml:"dns"`
	} `yaml:"server"`

	Log struct {
		Network    string `yaml:"network"`
		Host       string `yaml:"host"`
		Port       string `yaml:"port"`
		Type       string `yaml:"type"`
		Debug_mode bool   `yaml:"debug_mode"`
		Severity   string `yaml:"severity"`
		Facility   string `yaml:"facility"`
		File_path  string `yaml:"file_path"`
		File_name  string `yaml:"file_name"`
	} `yaml:"log"`

	Storage struct {
		Login    string   `yaml:"login"`
		Password string   `yaml:"password"`
		Bucket   string   `yaml:"bucket"`
		Hosts    []string `yaml:"hosts"`
	} `yaml:"storage"`
}

var configFlag = flag.String("config", "./default.yaml", "set config file in the yaml format")
var Config Conf

func configure() Conf {

	configFile, err := ioutil.ReadFile(*configFlag)
	if err != nil {
		fmt.Println("Failed read configuration file: ", err)
	}

	var c Conf
	err = yaml.Unmarshal(configFile, &c)

	if err != nil {
		fmt.Println("Failed unmarshal ", *configFlag, err)
	}

	return c
}
