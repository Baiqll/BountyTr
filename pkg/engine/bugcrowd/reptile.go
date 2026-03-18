package bugcrowd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/baiqll/bountytr/pkg/utils"
	"github.com/tidwall/gjson"
)

// FastEngine Bugcrowd快速引擎
type FastEngine struct {
	cacheDir         string
	config           utils.Bugcrowd
	handle           string
	silent           bool
	client           *utils.HttpClient
	concurrency      int
	githubCacheCount int
}

// NewFastEngine 创建快速引擎
func NewFastEngine(cacheDir string, config utils.Bugcrowd, handle string, silent bool) *FastEngine {
	ratePerMinute := 400
	concurrency := 50
	if !config.Private.Enable {
		ratePerMinute = 50
		concurrency = 10
	}

	var client *utils.HttpClient
	if config.Private.Enable {
		client = utils.NewHttpClientWithAuth(cacheDir, ratePerMinute, config.Private.APIName, config.Private.APIToken)
	} else {
		client = utils.NewHttpClient(cacheDir, ratePerMinute)
	}

	return &FastEngine{
		cacheDir:    cacheDir,
		config:      config,
		handle:      handle,
		silent:      silent,
		client:      client,
		concurrency: concurrency,
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
		program := &ProgramWithScope{Handle: "engagements/" + e.matchHandle(e.handle)}

		e.fetchProgramScopeWithData(program, output)

		return nil
	}

	// 从GitHub平台获取数据
	if err := e.RunGithubPublic(output); err != nil {
		return err
	}

	// 从Bugcrowd平台获取数据
	if e.config.Private.Enable {
		if err := e.RunAPI(output); err != nil {
			return err
		}
	}
	return nil
}

// Public 获取公开项目（从GitHub）
func (e *FastEngine) RunGithubPublic(output chan<- utils.NewScope) error {
	const GitHubDataURL = "https://raw.githubusercontent.com/arkadiyt/bounty-targets-data/main/data/bugcrowd_data.json"

	data, err := e.client.FetchWithCache(GitHubDataURL, "bugcrowd_data.json", 1*time.Hour, e.silent)
	if err != nil {
		return err
	}

	var programs []GitHubData
	if err := json.Unmarshal(data, &programs); err != nil {
		return err
	}

	e.githubCacheCount = len(programs)

	for _, p := range programs {
		output <- utils.NewScope{NewPublicURL: p.URL}
		for _, target := range p.Targets.InScope {
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

	if !e.silent {
		fmt.Printf("[*] Bugcrowd public 项目 %d 个\n", len(programs))
	}

	return nil
}

// Public + Private 获公共+私有项目（从API）
func (e *FastEngine) RunAPI(output chan<- utils.NewScope) error {
	if !e.config.Private.Enable {
		return nil
	}

	privateCachePath := filepath.Join(e.cacheDir, "bugcrowd_private_data.json")

	// 尝试从缓存加载
	if cached_programs, ok := e.loadCache(privateCachePath, 1*time.Hour); ok {
		if !e.silent {
			fmt.Printf("[*] Bugcrowd private 项目 %d 个 (缓存)\n", len(cached_programs))
		}
		e.outputFromCache(cached_programs, output)
		return nil
	}

	if !e.silent {
		fmt.Println("[*] Bugcrowd API 获取项目列表...")
	}

	privatePrograms, err := e.fetchAllPrograms()
	if err != nil {
		return err
	}

	// 缓存 private 项目
	if len(privatePrograms) > 0 {
		if !e.silent {
			fmt.Printf("[*] Bugcrowd private 项目 获取scope中...\n")
		}
		privateCache := e.fetchScopesAndBuildCache(privatePrograms, output)
		e.saveCache(privateCachePath, privateCache)
	}

	if !e.silent {
		fmt.Println("[*] Bugcrowd API 完成")
	}
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

func (e *FastEngine) fetchAllPrograms() ([]*APIProgram, error) {
	var allPrograms []*APIProgram
	page := 1

	for {
		url := fmt.Sprintf("https://bugcrowd.com/engagements.json?category=bug_bounty&page=%d", page)
		cacheKey := fmt.Sprintf("bc_page_%d", page)
		headers := map[string]string{"Accept": "application/json"}

		data, _, err := e.client.GetWithCache(url, cacheKey, headers, 15*time.Minute)
		if err != nil && len(data) == 0 {
			break
		}

		var resp APIResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			break
		}

		if len(resp.Engagements) == 0 {
			break
		}

		for i := range resp.Engagements {
			p := &resp.Engagements[i]
			if p.IsPrivate {
				allPrograms = append(allPrograms, p)
			}
		}

		page++
	}

	if !e.silent {
		fmt.Printf("[*] Bugcrowd API: private %d 个\n", len(allPrograms))
	}

	return allPrograms, nil
}

func (e *FastEngine) fetchScopesAndBuildCache(programs []*APIProgram, output chan<- utils.NewScope) []ProgramWithScope {
	cache := make([]ProgramWithScope, 0, len(programs))

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, e.concurrency)

	for _, p := range programs {
		output <- utils.NewScope{NewPrivateURL: "https://bugcrowd.com" + p.BriefURL}

		wg.Add(1)
		go func(prog *APIProgram) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			programData := e.fetchProgramScopeWithData(&ProgramWithScope{Handle: prog.BriefURL}, output)
			if programData != nil {
				mu.Lock()
				cache = append(cache, *programData)
				mu.Unlock()
			}
		}(p)
	}

	wg.Wait()
	return cache
}

func (e *FastEngine) fetchProgramScopeWithData(programData *ProgramWithScope, output chan<- utils.NewScope) *ProgramWithScope {
	handle := programData.Handle
	briefURL := "https://bugcrowd.com/" + handle

	// 获取项目页面提取 changelog UUID
	pageData, _, err := e.client.GetWithCache(briefURL, fmt.Sprintf("bc_page_%s", handle), map[string]string{"Accept": "text/html"}, 1*time.Hour)
	if err != nil || len(pageData) == 0 {
		return nil
	}

	// 从页面中提取 changelog UUID
	changelogPattern := regexp.MustCompile(handle + `/changelog/([a-f0-9\-]+)`)
	matches := changelogPattern.FindSubmatch(pageData)
	if len(matches) < 2 {
		return nil
	}

	changelogUUID := string(matches[1])
	targetURL := fmt.Sprintf("https://bugcrowd.com/%s/changelog/%s.json", handle, changelogUUID)

	data, _, err := e.client.GetWithCache(targetURL, fmt.Sprintf("bc_scope_%s", handle), map[string]string{"Accept": "application/json"}, 1*time.Hour)
	if err != nil || len(data) == 0 {
		return nil
	}

	programData.URL = briefURL

	// 解析 scope 数组中的 targets
	scopeResult := gjson.GetBytes(data, "data.scope")
	for _, scope := range scopeResult.Array() {
		if !scope.Get("inScope").Bool() {
			continue
		}

		targetsResult := scope.Get("targets")
		for _, targetItem := range targetsResult.Array() {
			target := APITarget{
				UUID:     targetItem.Get("id").String(),
				Name:     targetItem.Get("name").String(),
				Category: targetItem.Get("category").String(),
				URI:      targetItem.Get("uri").String(),
			}

			if target.Name == "" {
				continue
			}

			programData.Targets.InScope = append(programData.Targets.InScope, target)

			if utils.In(target.Category, []string{"website", "api"}) {
				e.handleDomainIdentifier(target.Name, output)
			} else {
				if strings.HasPrefix(target.Name, "*") || strings.HasSuffix(target.Name, "*") {
					e.handleDomainIdentifier(target.Name, output)
				} else {
					e.handleAsset(target.Name, output)
				}
			}
		}
	}

	if programData.Targets.InScope == nil {
		return nil
	}

	return programData
}

func (e *FastEngine) loadCache(path string, maxAge time.Duration) ([]ProgramWithScope, bool) {
	if info, err := os.Stat(path); err == nil {
		if time.Since(info.ModTime()) < maxAge {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return nil, false
			}

			var programs []ProgramWithScope
			if err := json.Unmarshal(data, &programs); err != nil {
				return nil, false
			}

			return programs, true
		}
	}

	return nil, false
}

func (e *FastEngine) saveCache(path string, cache interface{}) {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return
	}
	ioutil.WriteFile(path, data, 0644)
}

func (e *FastEngine) outputFromCache(cache []ProgramWithScope, output chan<- utils.NewScope) {
	for _, p := range cache {
		output <- utils.NewScope{NewPrivateURL: p.URL}

		for _, target := range p.Targets.InScope {
			if utils.In(target.Category, []string{"website", "api"}) {
				e.handleDomainIdentifier(target.URI, output)
			} else {
				e.handleAsset(target.URI, output)
			}
		}
	}
}

func (e *FastEngine) matchHandle(handle string) string {
	url := strings.TrimPrefix(handle, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "bugcrowd.com/engagements/")

	return strings.Split(url, "/")[0]
}
