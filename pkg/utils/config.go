package utils

import (
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Private struct{
	Enable      bool `yaml:"Enable"`
	APIToken string  `yaml:"ApiToken"`
	APIName string  `yaml:"ApiName"`

} 

type Bugcrowd struct {
	Enable bool `yaml:"Enable"`
	Private Private `yaml:"Private"`
}

type HackerOne struct {
	Enable  bool    `yaml:"Enable"`
	Private Private `yaml:"Private"`
}

type Intigriti struct {
	Enable bool `yaml:"Enable"`
	Private Private `yaml:"Private"`
}
type DingTalk struct {
	AppKey    string `yaml:"AppKey"`
	AppSecret string `yaml:"AppSecret"`
}

type Config struct {
	Bugcrowd  Bugcrowd  `yaml:"Bugcrowd"`
	HackerOne HackerOne `yaml:"HackerOne"`
	Intigriti Intigriti `yaml:"Intigriti"`
	Blacklist []string  `yaml:"Black"`
	DingTalk  DingTalk  `yaml:"DingTalk"`
}

func Initconfig(source_path string) (config Config) {
	config = Config{
		HackerOne: HackerOne{Enable: true},
		Bugcrowd:  Bugcrowd{Enable: true},
		Intigriti: Intigriti{Enable: true},
		DingTalk: DingTalk{
			AppKey:    "",
			AppSecret: "",
		},
		Blacklist: []string{
			".gov",
			".edu",
			".json",
			".[0-9.]+$",
			"github.com/",
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
