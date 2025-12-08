package hackerone

import (
	"encoding/json"
	"fmt"
	"os"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/baiqll/bountytr/pkg/utils"
)

// FastEngine HackerOne快速引擎
type FastEngine struct {
	cacheDir    string
	config      utils.HackerOne
	handle      string
	silent      bool
	client      *utils.HttpClient
	concurrency int
	// cache
	githubCacheCount  int
}


// NewFastEngine 创建快速引擎
func NewFastEngine(cacheDir string, config utils.HackerOne,handle string, silent bool) *FastEngine {
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
		handle:		 handle,
		config:      config,
		silent:      silent,
		client:      client,
		concurrency: concurrency,
	}
}

// Public项目（GitHub、API） +   Private项目（API）
func (e *FastEngine) Run(output chan<- utils.NewScope) error {

	if e.handle != "" {
		handle_list, err  := utils.ReadFileToList(e.handle)
		// 判断是否是列表文件
		if err != nil{
			program := &APIProgram{}
			program.Attributes.Handle = strings.TrimSuffix(e.handle, "/")[strings.LastIndex(strings.TrimSuffix(e.handle, "/"), "/")+1:]
			e.fetchProgramScopeWithData(program, output)
		}else{
			for _,handle := range handle_list{
				program := &APIProgram{}
				program.Attributes.Handle = strings.TrimSuffix(handle, "/")[strings.LastIndex(strings.TrimSuffix(handle, "/"), "/")+1:]
				e.fetchProgramScopeWithData(program, output)
			}
		}
		
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
	const GitHubDataURL = "https://raw.githubusercontent.com/arkadiyt/bounty-targets-data/main/data/hackerone_data.json"

	data, err := e.client.FetchWithCache(GitHubDataURL, "hackerone_data.json", 1*time.Hour, e.silent)
	if err != nil {
		return err
	}

	var programs []ProgramWithScope
	if err := json.Unmarshal(data, &programs); err != nil {
		return err
	}

	var new_programs []ProgramWithScope

	for _, p := range programs {
		output <- utils.NewScope{NewPublicURL: p.URL}

		if p.SubmissionState != "open" || !p.OffersBounties || p.Targets.InScope == nil {
			continue
		}

		new_programs = append(new_programs,p)

		for _, target := range p.Targets.InScope {
			if utils.In(target.AssetType, []string{"DOMAIN", "URL", "WILDCARD"}) {
				e.handleDomainIdentifier(target.AssetIdentifier, output)
			} else if target.AssetType == "OTHER" {
				if strings.HasPrefix(target.AssetIdentifier, "*") || strings.HasSuffix(target.AssetIdentifier, "*") {
					e.handleDomainIdentifier(target.AssetIdentifier, output)
				} else {
					e.handleAsset(target.AssetIdentifier, output)
				}
			} else {
				e.handleAsset(target.AssetIdentifier, output)
			}
		}
	}
	// 更新 GitHub 数据
	cachePath := filepath.Join(e.cacheDir, "hackerone_data.json")
	e.saveCache(cachePath, new_programs)
	e.githubCacheCount = len(new_programs)

	if !e.silent {
		fmt.Printf("[*] HackerOne public 项目 %d 个\n", len(new_programs))
	}

	return nil
}

// Public + Private 获公共+私有项目（从API）
func (e *FastEngine) RunAPI(output chan<- utils.NewScope) error {
	
	if !e.config.Private.Enable {
		return nil
	}

	privateCachePath := filepath.Join(e.cacheDir, "hackerone_private_data.json")
	publicCachePath := filepath.Join(e.cacheDir, "hackerone_public_data.json")

	// 尝试从缓存加载
	if cached_programs, ok := e.loadCache(privateCachePath, 1*time.Hour); ok {
		if !e.silent {
			fmt.Printf("[*] HackerOne private 项目 %d 个 (缓存)\n", len(cached_programs))
		}
		e.outputFromCache(cached_programs, output)
		return nil
	}

	if !e.silent {
		fmt.Println("[*] HackerOne API 获取项目列表...")
	}

	publicPrograms, privatePrograms, err := e.fetchAllPrograms()
	if err != nil {
		return err
	}

	// 检查 public 项目数是否与 GitHub 数据一致
	// 数据不一致时 再详细对
	if len(publicPrograms) > 0 && len(publicPrograms) > e.githubCacheCount {
		newCount := len(publicPrograms) - e.githubCacheCount
		if !e.silent {
			fmt.Printf("[*] HackerOne public 项目新增 %d 个，获取scope中...\n", newCount)
		}
		publicCache := e.fetchScopesAndBuildCache(publicPrograms, output)
		e.saveCache(publicCachePath, publicCache)
	}

	// 缓存 private 项目
	if len(privatePrograms) > 0 {
		if !e.silent {
			fmt.Printf("[*] HackerOne private 项目 获取scope中...\n")
		}
		privateCache := e.fetchScopesAndBuildCache(privatePrograms, output)
		e.saveCache(privateCachePath, privateCache)
	}

	if !e.silent {
		fmt.Println("[*] HackerOne API 完成")
	}
	return nil
}


func (e *FastEngine) fetchAllPrograms() (publicPrograms, privatePrograms []*APIProgram, err error) {
	var pageCount int

	url := "https://api.hackerone.com/v1/hackers/programs?page[size]=100"

	for url != "" {
		pageCount++
		cacheKey := fmt.Sprintf("h1_api_page_%d", pageCount)
		headers := map[string]string{"Content-Type": "application/json"}

		data, _, err := e.client.GetWithCache(url, cacheKey, headers, 15*time.Minute)
		if err != nil && len(data) == 0 {
			return publicPrograms, privatePrograms, err
		}

		var resp APIResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			break
		}

		for _, p := range resp.Data {
			// 只要 submission_state 为 open 的项目
			if p.Attributes.SubmissionState != "open" || !p.Attributes.OffersBounties {
				continue
			}
			if p.Attributes.State == "soft_launched" {
				privatePrograms = append(privatePrograms, p)
			} else if p.Attributes.State == "public_mode" {
				publicPrograms = append(publicPrograms, p)
			}
		}

		url = resp.Links.Next
	}

	if !e.silent {
		fmt.Printf("[*] HackerOne API: public %d 个, private %d 个\n", len(publicPrograms), len(privatePrograms))
	}

	return publicPrograms, privatePrograms, nil
}

func (e *FastEngine) fetchScopesAndBuildCache(programs []*APIProgram, output chan<- utils.NewScope) []ProgramWithScope {
	cache := make([]ProgramWithScope, 0, len(programs))

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, e.concurrency)

	for _, p := range programs {
		// 根据状态输出不同的URL类型
		if p.Attributes.State == "soft_launched" {
			output <- utils.NewScope{NewPrivateURL: "https://hackerone.com/" + p.Attributes.Handle}
		} else {
			// public_mode 项目来自 API，标记为 IsFromAPI
			output <- utils.NewScope{NewPublicURL: "https://hackerone.com/" + p.Attributes.Handle, IsFromAPI: true}
		}

		wg.Add(1)
		go func(prog *APIProgram) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			programData := e.fetchProgramScopeWithData(prog, output)
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

func (e *FastEngine) fetchProgramScopeWithData(prog *APIProgram, output chan<- utils.NewScope) *ProgramWithScope {
	handle := prog.Attributes.Handle
	url := fmt.Sprintf("https://api.hackerone.com/v1/hackers/programs/%s/structured_scopes?page[size]=100", handle)
	cacheKey := fmt.Sprintf("h1_scope_%s", handle)
	headers := map[string]string{"Content-Type": "application/json"}

	data, _, err := e.client.GetWithCache(url, cacheKey, headers, 1*time.Hour)
	if err != nil || len(data) == 0 {
		return nil
	}

	var scope APIScope
	if err := json.Unmarshal(data, &scope); err != nil {
		return nil
	}

	programData := prog.Attributes
	programData.Handle = handle
	programData.URL =  "https://hackerone.com/" + handle

	for _, asset := range scope.Data {
		if !asset.Attributes.EligibleForBounty {
			continue
		}

		target := asset.Attributes
		
		programData.Targets.InScope = append(programData.Targets.InScope, target)

		if utils.In(target.AssetType, []string{"DOMAIN", "URL", "WILDCARD"}){
			e.handleDomainIdentifier(target.AssetIdentifier, output)
		} else if target.AssetType == "OTHER" {
			if strings.HasPrefix(target.AssetIdentifier, "*") || strings.HasSuffix(target.AssetIdentifier, "*") {
				e.handleDomainIdentifier(target.AssetIdentifier, output)
			} else {
				e.handleAsset(target.AssetIdentifier, output)
			}
		} else {
			e.handleAsset(target.AssetIdentifier, output)
		} 
	}
	if programData.Targets.InScope == nil {
		return nil
	}

	return &programData
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

func (e *FastEngine) loadCache(path string, maxAge time.Duration) ([]ProgramWithScope, bool) {
	
	if info, err := os.Stat(path); err == nil {
		if time.Since(info.ModTime()) < maxAge {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return nil, false
			}

			var programs []ProgramWithScope
			if err := json.Unmarshal(data, &programs); err != nil {
				return nil , false
			}

			return programs, true
		}
	}

	return nil , false

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
			if utils.In(target.AssetType, []string{"DOMAIN", "URL", "WILDCARD"}) {
				output <- utils.NewScope{NewTarget: target.AssetIdentifier}
			} else {
				output <- utils.NewScope{NewApp: target.AssetIdentifier}
			}
		}
	}
}

