package lib

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type DingTalk struct {
	AppKey    string `yaml:"AppKey"`
	AppSecret string `yaml:"AppSecret"`
}

type Bugcrowd struct {
	Url string `yaml:"Url"`
}

type HackerOne struct {
	Url string `yaml:"Url"`
}

type Intigriti struct {
	Url string `yaml:"Url"`
}

type Config struct {
	DingTalk  DingTalk  `yaml:"DingTalk"`
	Bugcrowd  Bugcrowd  `yaml:"Bugcrowd"`
	HackerOne HackerOne `yaml:"HackerOne"`
	Intigriti Intigriti `yaml:"Intigriti"`
}

func Initconfig(source_path string) (config Config) {
	config = Config{
		HackerOne: HackerOne{Url: "https://raw.githubusercontent.com/arkadiyt/bounty-targets-data/main/data/hackerone_data.json"},
		Bugcrowd:  Bugcrowd{Url: "https://raw.githubusercontent.com/arkadiyt/bounty-targets-data/main/data/bugcrowd_data.json"},
		Intigriti: Intigriti{Url: "https://raw.githubusercontent.com/arkadiyt/bounty-targets-data/main/data/intigriti_data.json"},
		DingTalk: DingTalk{
			AppKey:    "",
			AppSecret: "",
		},
	}

	data, _ := yaml.Marshal(config)
	ioutil.WriteFile(filepath.Join(source_path, "config.yaml"), data, 0777)

	return

}

func GetConfig(source_path string) (config Config) {

	content, err := ioutil.ReadFile(filepath.Join(source_path, "config.yaml"))

	if err != nil {
		config = Initconfig(source_path)
		fmt.Println("sssss")
		return
	}
	err = yaml.Unmarshal(content, &config)

	return
}
