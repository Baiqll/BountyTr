package lib

import (
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Bugcrowd struct {
	Url string `yaml:"Url"`
}

type HackerOne struct {
	Url string `yaml:"Url"`
}

type Intigriti struct {
	Url string `yaml:"Url"`
}
type DingTalk struct {
	AppKey    string `yaml:"AppKey"`
	AppSecret string `yaml:"AppSecret"`
}

type Config struct {
	Bugcrowd  bool     `yaml:"Bugcrowd"`
	HackerOne bool     `yaml:"HackerOne"`
	Intigriti bool     `yaml:"Intigriti"`
	Blacklist []string `yaml:"Black"`
	DingTalk  DingTalk `yaml:"DingTalk"`
}

func Initconfig(source_path string) (config Config) {
	config = Config{
		HackerOne: false,
		Bugcrowd:  false,
		Intigriti: false,
		DingTalk: DingTalk{
			AppKey:    "",
			AppSecret: "",
		},
		Blacklist: []string{
			".gov",
			".edu",
			".json",
			".[0-9.]+$",
			"https://github.com/",
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
		return
	}
	yaml.Unmarshal(content, &config)

	return
}
