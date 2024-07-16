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
	"github.com/baiqll/bountytr/pkg/proxypool"
	"github.com/baiqll/bountytr/pkg/utils"
)

var source_path = filepath.Join(utils.HomeDir(), ".config/bountytr/")

// type Bounty interface {
// 	hackerone.HackeroneTarget | bugcrowd.BugcrowdTarget | intigriti.IntigritiTarget
// }

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

type (
	Bountry struct {
		BugcrowdTry  bugcrowd.BugcrowdTry
		HackeroneTry hackerone.HackeroneTry
		IntigritiTry intigriti.IntigritiTry
		DingTalk     utils.DingTalk
		Config       utils.Config
		Pool         proxypool.Pool
	}
)

func NewBountry(source_path string) *Bountry {

	/*
		https://hackerone.com/directory/programs
		https://bugcrowd.com/programs
		https://www.intigriti.com/programs

	*/
	
	
	config := utils.GetConfig(source_path)
	
	proxy_pool := proxypool.NewProxyPool(config.EnableProxy).InitPool()
	

	return &Bountry{
		HackeroneTry: *hackerone.NewHackeroneTry(config.HackerOne, proxy_pool),
		BugcrowdTry:  *bugcrowd.NewBugcrowdTry(config.Bugcrowd, proxy_pool),
		IntigritiTry: *intigriti.NewIntigritiTry(config.Intigriti, proxy_pool),
		// DingTalk:     config.DingTalk,
		Config:       config,
		Pool:         proxy_pool,
	}
}

func (b Bountry) bugcrowd(new_bugcrowd chan utils.NewScope){

	b.BugcrowdTry.Program()

	// for _, target := range targets {

	// 	// 判断是否有赏金
	// 	if target.MaxRewards <= 0 {
	// 		continue
	// 	}

	// 	for _, scope := range target.Targets.InScope {

	// 		// 只打印 Web 目标
	// 		if utils.In(scope.Category, []string{"api", "website", "other"}) {
	// 			for _, domain := range b.DomainMatch(scope.Url) {
	// 				if !source_targets[domain] && !utils.In(domain, new_scope.new_targets) && !fail_targets[domain] {
	// 					if utils.DomainValid(domain) {
	// 						fmt.Println(strings.ReplaceAll(domain, "*.", ""))
	// 						new_scope.new_targets = append(new_scope.new_targets, domain)
	// 						new_scope.new_bounty_url = append(new_scope.new_bounty_url, target.Url)
	// 					} else {
	// 						new_scope.error_targets = append(new_scope.error_targets, domain)
	// 					}
	// 				}
	// 			}
	// 		} else if utils.In(scope.Category, []string{"android", "ios"}) {
	// 			new_scope.new_app = append(new_scope.new_app, scope.Url)
	// 			new_scope.new_bounty_url = append(new_scope.new_bounty_url, target.Url)
	// 		}
	// 	}

	// }

	// return
}

func (b Bountry) hackerone(new_hackerone chan utils.NewScope, handle string ){

	/*
		hackerone 
	*/

	if handle == "" {
		// 判断是否指定 项目名称，如果没有则获取所有
		b.HackeroneTry.ProgramsScope(new_hackerone)
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)

	b.HackeroneTry.GetScope(handle,new_hackerone, &wg)

	go func() {
		wg.Wait()
		
	}()

	
}

func (b Bountry) intigriti(new_intigriti chan utils.NewScope){

	b.IntigritiTry.Program()

	// for _, target := range targets {

	// 	// 判断是否有赏金
	// 	if target.MaxBounty.Value <= 0 {
	// 		continue
	// 	}

	// 	for _, scope := range target.Targets.InScope {
	// 		// 只打印 Web 目标
	// 		if utils.In(scope.Type, []string{"URL", "Other"}) {
	// 			for _, domain := range b.DomainMatch(scope.Endpoint) {
	// 				if !source_targets[domain] && !utils.In(domain, new_scope.new_targets) && !fail_targets[domain] {
	// 					if utils.DomainValid(domain) {
	// 						fmt.Println(strings.ReplaceAll(domain, "*.", ""))
	// 						new_scope.new_targets = append(new_scope.new_targets, domain)
	// 						new_scope.new_bounty_url = append(new_scope.new_bounty_url, target.Url)
	// 					} else {
	// 						new_scope.error_targets = append(new_scope.error_targets, domain)
	// 					}
	// 				}
	// 			}
	// 		} else if utils.In(scope.Type, []string{"ios", "android"}) {

	// 			new_scope.new_app = append(new_scope.new_app, scope.Endpoint)
	// 			new_scope.new_bounty_url = append(new_scope.new_bounty_url, target.Url)

	// 		}
	// 	}
	// }

	// return
}

func (b Bountry) DomainMatch(url string) []string {
	return utils.DomainMatch(url, b.Config.Blacklist)
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
	var handle string

	flag.Int64Var(&cycle_time, "t", 0, "监控周期(分钟)")
	flag.BoolVar(&silent, "silent", false, "是否静默状态")
	flag.StringVar(&handle, "handle", "", "指定项目名/获取项目列表名称")

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
				run(silent, handle)
			}),
		}
		for _, task := range tasks {
			go task.Run()
		}
		// 等待任务结束
		select {}
	} else {
		run(silent, handle)
	}
}

func run(silent bool, handle string) {

	if !silent {
		now := time.Now().Format("2006-01-02 15:04:05")

		fmt.Println("[*] Date:", now)
	}
	// init config
	bountry := NewBountry(source_path)

	// 读取源目标文件
	source_targets := utils.ReadFileToMap(filepath.Join(source_path, "domain.txt"))
	source_fail_targets := utils.ReadFileToMap(filepath.Join(source_path, "faildomain.txt"))
	source_bugbounty_url := utils.ReadFileToMap(filepath.Join(source_path, "bugbounty-public.txt"))
	private_bugbounty_url := utils.ReadFileToMap(filepath.Join(source_path, "bugbounty-private.txt"))

	// 获取新增赏金目标
	new_scope_chan := make(chan utils.NewScope)


	var outputWG sync.WaitGroup
	outputWG.Add(1)

	go func() {
		defer outputWG.Done()

		for scope := range new_scope_chan {

			if scope.NewTarget!= "" && !source_targets[scope.NewTarget]{

			
				re := regexp.MustCompile(strings.Join(bountry.Config.Blacklist, "|"))

				if !re.MatchString(scope.NewTarget){
					fmt.Println(scope.NewTarget)

					// 保存新增目标
					utils.SaveTargetsToFile(filepath.Join(source_path, "domain.txt"), scope.NewTarget)
				}

			}

			if scope.NewFailTarget!= "" && !source_fail_targets[scope.NewFailTarget] {

				utils.SaveTargetsToFile(filepath.Join(source_path, "faildomain.txt"), scope.NewFailTarget)
			}

			if  scope.NewPublicURL != "" && !source_bugbounty_url[scope.NewPublicURL]{

				utils.SaveTargetsToFile(filepath.Join(source_path, "bugbounty-public.txt"), scope.NewPublicURL)

			}

			if scope.NewPrivateURL!= "" && !private_bugbounty_url[scope.NewPrivateURL]{

				utils.SaveTargetsToFile(filepath.Join(source_path, "bugbounty-private.txt"), scope.NewPrivateURL)
			}

		}

	}()


	

	handle_list, err  := utils.ReadFileToList(handle)
	// 判断是否是列表文件
	if err != nil{
		
		if bountry.Config.HackerOne.Enable {
			bountry.hackerone(new_scope_chan, handle)
		}
		if bountry.Config.Bugcrowd.Enable {
			bountry.bugcrowd(new_scope_chan)
		}
		if bountry.Config.Intigriti.Enable {
			bountry.intigriti(new_scope_chan)
		}
		
	}else{

		for _, item := range handle_list {
			// 读取文件列表
			
			if bountry.Config.HackerOne.Enable {
				bountry.hackerone(new_scope_chan, item)
			}
			if bountry.Config.Bugcrowd.Enable {
				bountry.bugcrowd(new_scope_chan)
			}
			if bountry.Config.Intigriti.Enable {
				bountry.intigriti(new_scope_chan)
			}
		}

	}
	

	go func() {
		outputWG.Wait()
		close(new_scope_chan)
	}()




	// // 发送通知信息
	// var msg_content = notify.BountyContent{
	// 	Hackerone: notify.MessageContent{
	// 		Urls:    new_hackerone.new_bounty_url,
	// 		Targets: new_hackerone.new_targets,
	// 		App:     new_hackerone.new_app,
	// 	},
	// 	Bugcrowd: notify.MessageContent{
	// 		Urls:    new_bugcrowd.new_bounty_url,
	// 		Targets: new_bugcrowd.new_targets,
	// 		App:     new_bugcrowd.new_app,
	// 	},
	// 	Intigriti: notify.MessageContent{
	// 		Urls:    new_intigriti.new_bounty_url,
	// 		Targets: new_intigriti.new_targets,
	// 		App:     new_intigriti.new_app,
	// 	},
	// }

	// bountry.SendDingtalk(msg_content)

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
