package main

import (
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/go-yaml/yaml"
)

type Conf struct {
	Server struct {
		Http struct {
			Http_port string `yaml:"http_port"`
		} `yaml:"http"`
		Dns struct {
			Dns_port string `yaml:"dns_port"`
			Network  string `yaml:"network"`
			Ttl      uint32 `yaml:"ttl"`
		} `yaml:"dns"`
	} `yaml:"server"`

	Log struct {
		Network_type string `yaml:"network_type"`
		Log_host     string `yaml:"log_host"`
		Log_port     string `yaml:"log_port"`
		File_path    string `yaml:"file_path"`
		File_name    string `yaml:"file_name"`
	} `yaml:"log"`

	Storage struct {
		Login    string   `yaml:"login"`
		Password string   `yaml:"password"`
		Bucket   string   `yaml:"bucket"`
		Hosts    []string `yaml:"hosts"`
	} `yaml:"storage"`
}

var configFlag = flag.String("config", "./default.yaml", "set config file in the yaml format")
var config Conf

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
