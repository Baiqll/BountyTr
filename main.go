package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/baiqll/bountytr/src/lib"
	"github.com/baiqll/bountytr/src/models"
	"github.com/baiqll/bountytr/src/notify"
)

// var bugcrowdurl = "https://raw.githubusercontent.com/arkadiyt/bounty-targets-data/main/data/bugcrowd_data.json"
// var hackeroneurl = "https://raw.githubusercontent.com/arkadiyt/bounty-targets-data/main/data/hackerone_data.json"
// var intigritiurl = "https://raw.githubusercontent.com/arkadiyt/bounty-targets-data/main/data/intigriti_data.json"

var source_path = filepath.Join(lib.HomeDir(), ".config/bountytr/")

type Bounty interface {
	models.HackeroneTarget | models.BugcrowdTarget | models.IntigritiTarget
}

type Task struct {
	Name    string
	Timeout time.Duration
	fn      func()
}

func NewTask(name string, timeout time.Duration, fn func()) *Task {
	return &Task{
		Name:    name,
		Timeout: timeout,
		fn:      fn,
	}
}

func (t *Task) Run() {
	ticker := time.NewTicker(t.Timeout)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.fn()
		}
	}
}

type Bountry struct {
	BugcrowdUrl  string
	HackeroneUrl string
	IntigritiUrl string
	DingTalk     lib.DingTalk
}

func NewBountry(source_path string) *Bountry {

	config := lib.GetConfig(source_path)
	return &Bountry{
		HackeroneUrl: config.HackerOne.Url,
		BugcrowdUrl:  config.Bugcrowd.Url,
		IntigritiUrl: config.Intigriti.Url,
		DingTalk:     config.DingTalk,
	}
}

func BountyTarget(url string) []byte {

	// 请求JSON数据
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return body

}

func (b Bountry) bugcrowd(source_targets map[string]bool, fail_targets map[string]bool) (error_targets []string, new_targets []string, new_bounty_url []string) {

	var body = BountyTarget(b.BugcrowdUrl)

	// 解析JSON数据
	var targets []models.Bugcrowd
	err := json.Unmarshal(body, &targets)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, target := range targets {

		// 判断是否有赏金
		if target.MaxPayout <= 0 {
			continue
		}

		for _, scope := range target.Targets.InScope {

			// 只打印 Web 目标
			if lib.In(scope.Type, []string{"api", "website"}) {
				for _, domain := range lib.DomainMatch(scope.Target) {
					if !source_targets[domain] && !lib.In(domain, new_targets) && !fail_targets[domain] {
						if lib.DomainValid(domain) && !strings.Contains(scope.Target, `\*`) {
							fmt.Println(domain)
							new_targets = append(new_targets, domain)
							new_bounty_url = append(new_bounty_url, target.Url)
						} else {
							error_targets = append(error_targets, domain)
						}
					}
				}
			}
		}
	}

	return
}

func (b Bountry) hackerone(source_targets map[string]bool, fail_targets map[string]bool) (error_targets []string, new_targets []string, new_bounty_url []string) {

	var body = BountyTarget(b.HackeroneUrl)

	// 解析JSON数据
	var targets []models.Hackerone
	err := json.Unmarshal(body, &targets)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, target := range targets {

		// 判断是否有赏金
		if !target.OffersBounties {
			continue
		}

		for _, scope := range target.Targets.InScope {

			// 只打印 Web 目标
			if lib.In(scope.AssetType, []string{"URL", "WILDCARD"}) {
				for _, domain := range lib.DomainMatch(scope.AssetIdentifier) {
					if !source_targets[domain] && !lib.In(domain, new_targets) && !fail_targets[domain] {
						if lib.DomainValid(domain) && !strings.Contains(scope.AssetIdentifier, `\*`) {
							fmt.Println(domain)
							new_targets = append(new_targets, domain)
							new_bounty_url = append(new_bounty_url, target.Url)
						} else {
							error_targets = append(error_targets, domain)
						}
					}
				}
			}

			// 其他
			if lib.In(scope.AssetType, []string{"OTHER"}) {
				for _, domain := range lib.DomainMatch(scope.AssetIdentifier) {
					if !source_targets[domain] && !lib.In(domain, new_targets) && !fail_targets[domain] {
						if !lib.DomainValid(domain) && !strings.Contains(scope.AssetIdentifier, `\*`) {
							fmt.Println(domain)
							new_targets = append(new_targets, domain)
							new_bounty_url = append(new_bounty_url, target.Url)
						} else {
							error_targets = append(error_targets, domain)
						}
					}
				}
				for _, domain := range lib.DomainMatch(scope.Instruction) {
					if !source_targets[domain] && !lib.In(domain, new_targets) && !fail_targets[domain] {
						if lib.DomainValid(domain) && !strings.Contains(scope.Instruction, `\*`) {
							fmt.Println(domain)
							new_targets = append(new_targets, domain)
							new_bounty_url = append(new_bounty_url, target.Url)
						} else {
							error_targets = append(error_targets, domain)
						}
					}
				}
			}
		}
	}

	return
}

func (b Bountry) intigriti(source_targets map[string]bool, fail_targets map[string]bool) (error_targets []string, new_targets []string, new_bounty_url []string) {

	var body = BountyTarget(b.IntigritiUrl)

	// 解析JSON数据
	var targets []models.Intigriti
	err := json.Unmarshal(body, &targets)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, target := range targets {

		// 判断是否有赏金
		if target.MaxBounty.Value <= 0 {
			continue
		}

		for _, scope := range target.Targets.InScope {

			// 只打印 Web 目标
			if lib.In(scope.Type, []string{"url"}) {
				for _, domain := range lib.DomainMatch(scope.Endpoint) {
					if !source_targets[domain] && !lib.In(domain, new_targets) && !fail_targets[domain] {
						if lib.DomainValid(domain) && !strings.Contains(scope.Endpoint, `\*`) {
							fmt.Println(domain)
							new_targets = append(new_targets, domain)
							new_bounty_url = append(new_bounty_url, target.Url)
						} else {
							error_targets = append(error_targets, domain)
						}
					}
				}
			}
		}
	}

	return
}

func main() {

	var banner = `

         __                      __        __      
        / /_  ____  __  ______  / /___  __/ /______
       / __ \/ __ \/ / / / __ \/ __/ / / / __/ ___/
      / /_/ / /_/ / /_/ / / / / /_/ /_/ / /_/ /    
     /_.___/\____/\__,_/_/ /_/\__/\__, /\__/_/     
                                 /____/       v1.0       

   
	Keep track of bounty targets
    `

	var cycle_time int64
	var silent bool

	flag.Int64Var(&cycle_time, "t", 0, "监控周期(分钟)")
	flag.BoolVar(&silent, "silent", false, "是否静默状态")

	// 解析命令行参数写入注册的flag里
	flag.Parse()

	if !silent {
		fmt.Println(string(banner))
		fmt.Println("[*] Starting tracker", "... ")
	}

	os.MkdirAll(source_path, os.ModePerm)

	// 启动定时任务
	if cycle_time > 0 {

		tasks := []*Task{
			NewTask("tracker", time.Duration(cycle_time)*time.Minute, func() {
				run(silent)
			}),
		}
		for _, task := range tasks {
			go task.Run()
		}
		// 等待任务结束
		select {}
	} else {
		run(silent)
	}
}

func run(silent bool) {

	if !silent {
		now := time.Now().Format("2006-01-02 15:04:05")

		fmt.Println("[*] Date:", now)
	}
	// init config
	bountry := NewBountry(source_path)

	// 读取源目标文件
	source_targets := lib.ReadFileToMap(filepath.Join(source_path, "domain.txt"))
	fail_targets := lib.ReadFileToMap(filepath.Join(source_path, "faildomain.txt"))

	//

	// 获取新增赏金目标
	new_hackerone_fail_targets, new_hackerone_targets, new_hackerone_url := bountry.hackerone(source_targets, fail_targets)
	new_bugcrowd_fail_targets, new_bugcrowd_targets, new_bugcrowd_url := bountry.bugcrowd(source_targets, fail_targets)
	new_intigriti_fail_targets, new_intigriti_targets, new_intigriti_url := bountry.intigriti(source_targets, fail_targets)

	var new_targets = append(append(new_hackerone_targets, new_bugcrowd_targets...), new_intigriti_targets...)
	var new_fail_targets = append(append(new_hackerone_fail_targets, new_bugcrowd_fail_targets...), new_intigriti_fail_targets...)

	// 保存新增目标
	lib.SaveTargetsToFile(filepath.Join(source_path, "domain.txt"), new_targets)

	lib.SaveTargetsToFile(filepath.Join(source_path, "faildomain.txt"), new_fail_targets)

	// 发送通知信息
	var msg_content = notify.BountyContent{
		Hackerone: notify.MessageContent{
			Urls:    new_hackerone_url,
			Targets: new_hackerone_targets,
		},
		Bugcrowd: notify.MessageContent{
			Urls:    new_bugcrowd_url,
			Targets: new_bugcrowd_targets,
		},
		Intigriti: notify.MessageContent{
			Urls:    new_intigriti_url,
			Targets: new_intigriti_targets,
		},
	}

	bountry.SendDingtalk(msg_content)

}

func (bountry Bountry) SendDingtalk(content notify.BountyContent) {

	var msg_content = notify.TargetMarkdown("Hackerone", content.Hackerone) +
		notify.TargetMarkdown("Intigriti", content.Intigriti)

	if msg_content == "" {
		return
	}

	var receiver notify.Robot
	receiver.AppKey = bountry.DingTalk.AppKey
	receiver.AppSecret = bountry.DingTalk.AppSecret
	webhookurl := receiver.Signature()
	params := receiver.SendMarkdown("Bountytr 资产监控", msg_content, []string{}, []string{}, false)

	notify.SendRequest(webhookurl, params)
}
