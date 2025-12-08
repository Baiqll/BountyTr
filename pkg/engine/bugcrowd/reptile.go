package bugcrowd

import (
	"encoding/json"
	"fmt"
	"strings"
	"io/ioutil"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/baiqll/bountytr/pkg/utils"
	"github.com/tidwall/gjson"
)

// FastEngine Bugcrowd快速引擎
type FastEngine struct {
	cacheDir string
	config   utils.Bugcrowd
	handle   string
	silent   bool
	client   *utils.HttpClient
}

// NewFastEngine 创建快速引擎
func NewFastEngine(cacheDir string, config utils.Bugcrowd,handle string, silent bool) *FastEngine {
	
	var client *utils.HttpClient
	if config.Private.Enable {
		client = utils.NewHttpClientWithAuth(cacheDir, 400, config.Private.APIName, config.Private.APIToken)
	} else {
		client = utils.NewHttpClient(cacheDir, 400)
	}

	return &FastEngine{
		cacheDir: cacheDir,
		config:   config,
		handle:    handle,
		silent:   silent,
		client:   client,
	}
}

// GitHubData GitHub数据结构
type GitHubData struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Targets struct {
		InScope []struct {
			Type   string `json:"type"`
			Target string `json:"target"`
		} `json:"in_scope"`
	} `json:"targets"`
}

// Public项目（GitHub、API） +   Private项目（API）
func (e *FastEngine) Run(output chan<- utils.NewScope) error {

	if e.handle != "" {
		// handle_list, err  := utils.ReadFileToList(e.handle)
		// 判断是否是列表文件
		// if err != nil{
		// 	return nil
		// }else{
		// 	for _,handle := range handle_list{
		// 		return nil
		// 	}
		// }
		
		return nil
	}

	// 从GitHub平台获取数据
	if err := e.RunGithubPublic(output); err != nil {
		return err
	}

	// 从hackerone平台获取数据
	if e.config.Private.Enable {
		if err := e.RunAPI(output); err != nil {
			return err
		}
	}
	return nil
}

// Public 获取公开项目（从GitHub）
func (e *FastEngine) RunGithubPublic(output chan<- utils.NewScope) error {
	
	// GitHub数据URL
	const GitHubDataURL = "https://raw.githubusercontent.com/arkadiyt/bounty-targets-data/main/data/bugcrowd_data.json"

	data, err := e.client.FetchWithCache(GitHubDataURL, "bugcrowd_data.json", 1*time.Hour, e.silent)
	if err != nil {
		return err
	}

	var programs []GitHubData
	if err := json.Unmarshal(data, &programs); err != nil {
		return err
	}

	if !e.silent {
		fmt.Printf("[*] Bugcrowd 共 %d 个项目\n", len(programs))
	}

	for _, p := range programs {
		output <- utils.NewScope{NewPublicURL: p.URL}
		for _, target := range p.Targets.InScope {
			// if utils.In(target.Type, []string{"website", "api"}) {
			// 	output <- utils.NewScope{NewTarget: e.cleanDomain(target.Target)}
			// } else {
			// 	output <- utils.NewScope{NewApp: e.cleanDomain(target.Target)}
			// }

			if utils.In(target.Type, []string{"website", "api"}) {
				e.handleDomainIdentifier(target.Target, output)
			} else {
				if strings.HasPrefix(target.Target, "*") || strings.HasSuffix(target.Target, "*") {
					e.handleDomainIdentifier(target.Target, output)
				} else {
					e.handleAsset(target.Target, output)
				}
			}
		}
	}
	return nil
}

// RunPrivate 获取私有项目（暂未实现）
func (e *FastEngine) RunAPI(output chan<- utils.NewScope) error {
	return nil
}

// cleanDomain 清理域名
func (e *FastEngine) cleanDomain(domain string) string {
	pattern := `[\w]+[\w\-_~\.]+\.[a-zA-Z]+`
	r, err := regexp.Compile(pattern)
	if err != nil {
		return domain
	}
	cDomain := r.FindString(domain)
	if cDomain != "" {
		return cDomain
	}
	return ""
}

// handleDomainIdentifier 处理域名类型的标识符
func (e *FastEngine) handleDomainIdentifier(identifier string, output chan<- utils.NewScope) {
	identifier = e.cleanDomain(identifier)
	for _, domain := range strings.Split(identifier, ",") {
		domain = strings.TrimSpace(domain)
		if domain != "" {
			output <- utils.NewScope{NewTarget: domain}
		}
	}
}

// handleAsset 处理非域名类型的资产
func (e *FastEngine) handleAsset(identifier string, output chan<- utils.NewScope) {
	for _, asset := range strings.Split(identifier, ",") {
		asset = strings.TrimSpace(asset)
		if asset != "" {
			output <- utils.NewScope{NewApp: asset}
		}
	}
}

// ============ 旧引擎代码（保留） ============

type BugcrowdTry struct {
	Programs []Bugcrowd     `json:"programs"`
	Config   utils.Bugcrowd `json:"config"`
}

func NewBugcrowdTry(config utils.Bugcrowd) *BugcrowdTry {
	return &BugcrowdTry{
		Programs: []Bugcrowd{},
		Config:   config,
	}
}

func (b BugcrowdTry) ProgramJson(path string) (body []byte, err error) {
	client := &http.Client{Timeout: 10 * time.Second}
	url := "https://bugcrowd.com" + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Accept", "*/*")
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		err = fmt.Errorf(resp.Status)
	}
	return
}

func (b BugcrowdTry) ProgramPage(page int64) (total_page int64, page_program []Bugcrowd, err error) {
	res_data, err := b.ProgramJson(fmt.Sprintf("/programs.json?vdp[]=false&page[]=%d", page))
	if err != nil {
		return
	}
	total_page = gjson.GetBytes(res_data, "meta.totalPages").Int()
	program_result := gjson.GetBytes(res_data, "programs")
	err = json.Unmarshal([]byte(program_result.Raw), &page_program)
	return
}

func (b BugcrowdTry) Program() (programs []Bugcrowd) {
	var new_program []Bugcrowd
	new_bugcrowd_program := make(chan Bugcrowd)
	semaphore := make(chan struct{}, 15)

	total_page, new_program, err := b.ProgramPage(1)
	if err != nil {
		fmt.Println("bugcrowd 获取programs 失败", err)
		return
	}

	var wgp sync.WaitGroup
	wgp.Add(int(total_page) - 1)

	for i := 2; i <= int(total_page); i++ {
		go func(page int) {
			defer wgp.Done()
			_, program, err := b.ProgramPage(int64(page))
			if err != nil {
				fmt.Println("bugcrowd 获取programs 失败", err)
			}
			new_program = append(new_program, program...)
		}(i)
	}
	wgp.Wait()

	var wg sync.WaitGroup
	for _, item := range new_program {
		wg.Add(1)
		item.Url = "https://bugcrowd.com" + item.ProgramUrl
		if item.InvitedStatus != "open" || item.Participation == "private" {
			wg.Done()
			continue
		}
		go b.Scope(item, new_bugcrowd_program, semaphore, &wg)
	}

	for {
		select {
		case scope_program := <-new_bugcrowd_program:
			programs = append(programs, scope_program)
		case <-time.After(10 * time.Second):
			wg.Wait()
			return
		}
	}
}

func (b BugcrowdTry) Target(url string) (scope []BugcrowdScope, err error) {
	res_data, err := b.ProgramJson(url)
	if err != nil {
		return
	}
	result := gjson.GetBytes(res_data, "targets")
	if result.Raw == "" {
		return
	}
	err = json.Unmarshal([]byte(result.Raw), &scope)
	return
}

func (b BugcrowdTry) Scope(bugcrowd Bugcrowd, new_bugcrowd_program chan Bugcrowd, semaphore chan struct{}, wg *sync.WaitGroup) (in_scopes []BugcrowdScope, out_scopes []BugcrowdScope) {
	defer wg.Done()
	semaphore <- struct{}{}

	target_data, err := b.ProgramJson(bugcrowd.ProgramUrl + "/target_groups")
	if err != nil {
		fmt.Println("bugcrowd 获取target_groups 失败", err)
		<-semaphore
		new_bugcrowd_program <- bugcrowd
		return
	}

	in_result := gjson.GetBytes(target_data, "groups.#(in_scope==true)#.targets_url")
	for _, item := range in_result.Array() {
		new_in_scopes, _ := b.Target(item.Str)
		in_scopes = append(in_scopes, new_in_scopes...)
	}
	bugcrowd.Targets.InScope = in_scopes
	<-semaphore
	new_bugcrowd_program <- bugcrowd
	return
}
