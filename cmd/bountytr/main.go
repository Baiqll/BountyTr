package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/baiqll/bountytr/pkg/engine/bugcrowd"
	"github.com/baiqll/bountytr/pkg/engine/hackerone"
	"github.com/baiqll/bountytr/pkg/engine/intigriti"
	"github.com/baiqll/bountytr/pkg/notify"
	"github.com/baiqll/bountytr/pkg/utils"
)

var source_path = filepath.Join(utils.HomeDir(), ".config/bountytr/")

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
	DingTalk utils.DingTalk
	Config   utils.Config
}

func NewBountry(source_path string) *Bountry {
	config := utils.GetConfig(source_path)
	return &Bountry{
		Config: config,
	}
}


func main() {
	var banner = `

         __                      __        __      
        / /_  ____  __  ______  / /___  __/ /______
       / __ \/ __ \/ / / / __ \/ __/ / / / __/ ___/
      / /_/ / /_/ / /_/ / / / / /_/ /_/ / /_/ /    
     /_.___/\____/\__,_/_/ /_/\__/\__, /\__/_/     
                                 /____/       v3.0       

   
	Keep track of bounty targets
    `

	var cycle_time int64
	var silent bool
	var handle string

	flag.Int64Var(&cycle_time, "t", 0, "监控周期(分钟)")
	flag.BoolVar(&silent, "silent", false, "是否静默状态")
	flag.StringVar(&handle, "handle", "", "指定项目名/获取项目列表名称")

	flag.Parse()

	if !silent {
		fmt.Println(string(banner))
		fmt.Println("[*] Starting tracker", "... ")
	}

	os.MkdirAll(source_path, os.ModePerm)

	// 运行即执行
	run(silent, handle)

	if cycle_time > 0 {
		// 开启定时任务
		tasks := []*Task{
			NewTask("tracker", time.Duration(cycle_time)*time.Minute, func() {
				run(silent, handle)
			}),
		}
		for _, task := range tasks {
			go task.Run()
		}
		select {}
	}
}


func run(silent bool, handle string) {
	if !silent {
		now := time.Now().Format("2006-01-02 15:04:05")
		fmt.Println("[*] Date:", now)
	}

	bountry := NewBountry(source_path)

	source_targets := utils.ReadFileToMap(filepath.Join(source_path, "domain.txt"))
	source_fail_targets := utils.ReadFileToMap(filepath.Join(source_path, "faildomain.txt"))

	// 清空 bugbounty 文件，每次运行重新写入
	publicFile := filepath.Join(source_path, "bugbounty-public.txt")
	privateFile := filepath.Join(source_path, "bugbounty-private.txt")
	
	if handle == "" {
		os.Truncate(publicFile, 0)
		os.Truncate(privateFile, 0)
	}
	
	source_bugbounty_url := make(map[string]bool)
	private_bugbounty_url := make(map[string]bool)

	new_scope_chan := make(chan utils.NewScope, 1000)

	var outputWG sync.WaitGroup
	outputWG.Add(1)

	go func() {
		defer outputWG.Done()
		black_re := regexp.MustCompile(strings.Join(bountry.Config.Blacklist, "|"))
		apiNewMarked := false // 标记是否已添加 API NEW 分隔符

		for scope := range new_scope_chan {
			// scope.NewTarget 新目标

			if handle != "" {
				if scope.NewTarget != "" {
					fmt.Println(scope.NewTarget)
				}
				continue
			}
			
			if scope.NewTarget != "" && !source_targets[scope.NewTarget] {
				if !black_re.MatchString(scope.NewTarget) {
					fmt.Println(scope.NewTarget)
					source_targets[scope.NewTarget] = true
					utils.SaveTargetsToFile(filepath.Join(source_path, "domain.txt"), scope.NewTarget)
				}
			}

			// scope.NewFailTarget 匹配失败的目标
			if scope.NewFailTarget != "" && !source_fail_targets[scope.NewFailTarget] {
				source_fail_targets[scope.NewFailTarget] = true
				utils.SaveTargetsToFile(filepath.Join(source_path, "faildomain.txt"), scope.NewFailTarget)
			}

			// scope.NewPublicURL 新公共项目
			if scope.NewPublicURL != "" && !source_bugbounty_url[scope.NewPublicURL] {
				source_bugbounty_url[scope.NewPublicURL] = true
				if scope.IsFromAPI && !apiNewMarked {
					utils.SaveTargetsToFile(filepath.Join(source_path, "bugbounty-public.txt"), "== API NEW ==")
					apiNewMarked = true
				}
				utils.SaveTargetsToFile(filepath.Join(source_path, "bugbounty-public.txt"), scope.NewPublicURL)
			}

			// scope.NewPrivateURL 新私有项目
			if scope.NewPrivateURL != "" && !private_bugbounty_url[scope.NewPrivateURL] {
				private_bugbounty_url[scope.NewPrivateURL] = true
				utils.SaveTargetsToFile(filepath.Join(source_path, "bugbounty-private.txt"), scope.NewPrivateURL)
			}
		}
	}()

	cacheDir := filepath.Join(source_path, "cache")
	runFull(cacheDir, bountry.Config, silent, handle, new_scope_chan)

	close(new_scope_chan)
	outputWG.Wait()
}

// runFull 完整模式 - 公开数据 + 私有项目
func runFull(cacheDir string, config utils.Config, silent bool,handle string, output chan<- utils.NewScope) {
	
	var wg sync.WaitGroup

	if config.HackerOne.Enable {
		// HackerOne 项目
		wg.Add(1)
		go func() {
			defer wg.Done()
			hackerone_engine := hackerone.NewFastEngine(cacheDir, config.HackerOne, handle, silent)
			if err := hackerone_engine.Run(output); err != nil && !silent {
				fmt.Printf("[-] HackerOne 错误: %v\n", err)
			}
		}()

	}

	if config.Bugcrowd.Enable {
		// Bugcrowd 项目
		wg.Add(1)
		go func() {
			defer wg.Done()
			bugcrowd_engine := bugcrowd.NewFastEngine(cacheDir, config.Bugcrowd, handle, silent)
			if err := bugcrowd_engine.Run(output); err != nil && !silent {
				fmt.Printf("[-] Bugcrowd 错误: %v\n", err)
			}
		}()
	}

	if config.Intigriti.Enable {
		// Intigriti 项目
		wg.Add(1)
		go func() {
			defer wg.Done()
			intigriti_engine := intigriti.NewFastEngine(cacheDir, config.Intigriti, handle, silent)
			if err := intigriti_engine.Run(output); err != nil && !silent {
				fmt.Printf("[-] Intigriti 错误: %v\n", err)
			}
		}()
	}

	wg.Wait()
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
