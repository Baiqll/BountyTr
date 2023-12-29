package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/baiqll/bountytr/src/lib"
	"github.com/baiqll/bountytr/src/models"
	"github.com/baiqll/bountytr/src/notify"
	"github.com/baiqll/bountytr/src/programs"
)

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
	BugcrowdTry  programs.BugcrowdTry
	HackeroneTry programs.HackeroneTry
	IntigritiTry programs.IntigritiTry
	DingTalk     lib.DingTalk
	Config       lib.Config
}

func NewBountry(source_path string) *Bountry {

	config := lib.GetConfig(source_path)
	return &Bountry{
		HackeroneTry: *programs.NewHackeroneTry("https://hackerone.com/directory/programs"),
		BugcrowdTry:  *programs.NewBugcrowdTry("https://bugcrowd.com/programs"),
		IntigritiTry: *programs.NewIntigritiTry("https://www.intigriti.com/programs"),
		DingTalk:     config.DingTalk,
		Config:       config,
	}
}

func (b Bountry) bugcrowd(source_targets map[string]bool, fail_targets map[string]bool) (error_targets []string, new_targets []string, new_bounty_url []string) {

	targets := b.BugcrowdTry.Program()

	for _, target := range targets {

		// 判断是否有赏金
		if target.MaxRewards <= 0 {
			continue
		}

		for _, scope := range target.Targets.InScope {

			// 只打印 Web 目标
			if lib.In(scope.Category, []string{"api", "website", "other"}) {
				for _, domain := range b.DomainMatch(scope.Name) {
					if !source_targets[domain] && !lib.In(domain, new_targets) && !fail_targets[domain] {
						if lib.DomainValid(domain) {
							fmt.Println(strings.ReplaceAll(domain, "*.", ""))
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

	targets := b.HackeroneTry.Program()

	for _, target := range targets {

		for _, scope := range target.Targets.InScope {

			/*  AssetType 类型：
			URL
			OTHER
			WILDCARD
			SMART_CONTRACT
			OTHER_IPA
			WINDOWS_APP_STORE_APP_ID
			HARDWARE
			GOOGLE_PLAY_APP_ID
			TESTFLIGHT
			CIDR
			APPLE_STORE_APP_ID
			DOWNLOADABLE_EXECUTABLES
			SOURCE_CODE
			OTHER_APK
			*/

			// 只打印 Web 目标
			if lib.In(scope.AssetType, []string{"URL", "WILDCARD", "Domain"}) {
				for _, domain := range b.DomainMatch(scope.AssetIdentifier) {
					if !source_targets[domain] && !lib.In(domain, new_targets) && !fail_targets[domain] {
						if lib.DomainValid(domain) {
							fmt.Println(strings.ReplaceAll(domain, "*.", ""))
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
				for _, domain := range b.DomainMatch(scope.AssetIdentifier) {
					if !source_targets[domain] && !lib.In(domain, new_targets) && !fail_targets[domain] {
						if !lib.DomainValid(domain) {
							fmt.Println(strings.ReplaceAll(domain, "*.", ""))
							new_targets = append(new_targets, domain)
							new_bounty_url = append(new_bounty_url, target.Url)
						} else {
							error_targets = append(error_targets, domain)
						}
					}
				}
				for _, domain := range b.DomainMatch(scope.Instruction) {
					if !source_targets[domain] && !lib.In(domain, new_targets) && !fail_targets[domain] {
						if lib.DomainValid(domain) {
							fmt.Println(strings.ReplaceAll(domain, "*.", ""))
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

	targets := b.IntigritiTry.Program()

	for _, target := range targets {

		// 判断是否有赏金
		if target.MaxBounty.Value <= 0 {
			continue
		}

		for _, scope := range target.Targets.InScope {
			// 只打印 Web 目标
			if lib.In(scope.Type, []string{"URL", "Other"}) {
				for _, domain := range b.DomainMatch(scope.Endpoint) {
					if !source_targets[domain] && !lib.In(domain, new_targets) && !fail_targets[domain] {
						if lib.DomainValid(domain) {
							fmt.Println(strings.ReplaceAll(domain, "*.", ""))
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

func (b Bountry) DomainMatch(url string) []string {
	return lib.DomainMatch(url, b.Config.Blacklist)
}

func main() {

	var banner = `

         __                      __        __      
        / /_  ____  __  ______  / /___  __/ /______
       / __ \/ __ \/ / / / __ \/ __/ / / / __/ ___/
      / /_/ / /_/ / /_/ / / / / /_/ /_/ / /_/ /    
     /_.___/\____/\__,_/_/ /_/\__/\__, /\__/_/     
                                 /____/       v2.0       

   
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

	// 获取新增赏金目标
	var new_hackerone_fail_targets, new_hackerone_targets, new_hackerone_url, new_bugcrowd_fail_targets, new_bugcrowd_targets, new_bugcrowd_url, new_intigriti_fail_targets, new_intigriti_targets, new_intigriti_url []string

	if bountry.Config.HackerOne {
		new_hackerone_fail_targets, new_hackerone_targets, new_hackerone_url = bountry.hackerone(source_targets, fail_targets)
	}
	if bountry.Config.Bugcrowd {
		new_bugcrowd_fail_targets, new_bugcrowd_targets, new_bugcrowd_url = bountry.bugcrowd(source_targets, fail_targets)
	}
	if bountry.Config.Intigriti {
		new_intigriti_fail_targets, new_intigriti_targets, new_intigriti_url = bountry.intigriti(source_targets, fail_targets)
	}

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
		notify.TargetMarkdown("Bugcrowd", content.Bugcrowd) +
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
