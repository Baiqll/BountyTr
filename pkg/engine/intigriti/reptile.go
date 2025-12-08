package intigriti

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/baiqll/bountytr/pkg/utils"
	"github.com/tidwall/gjson"
	"golang.org/x/net/html"
)

// FastEngine Intigriti快速引擎
type FastEngine struct {
	cacheDir string
	config   utils.Intigriti
	handle   string
	silent   bool
	client   *utils.HttpClient
}

// NewFastEngine 创建快速引擎
func NewFastEngine(cacheDir string, config utils.Intigriti,handle  string, silent bool) *FastEngine {
	
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
			Type     string `json:"type"`
			Endpoint string `json:"endpoint"`
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
	// 关闭通道
	return nil
}

// RunPublic 获取公开项目（从GitHub）
func (e *FastEngine) RunGithubPublic(output chan<- utils.NewScope) error {
	
	// GitHub数据URL
	const GitHubDataURL = "https://raw.githubusercontent.com/arkadiyt/bounty-targets-data/main/data/intigriti_data.json"

	data, err := e.client.FetchWithCache(GitHubDataURL, "intigriti_data.json", 1*time.Hour,e.silent)
	if err != nil {
		return err
	}

	var programs []GitHubData
	if err := json.Unmarshal(data, &programs); err != nil {
		return err
	}

	if !e.silent {
		fmt.Printf("[*] Intigriti 共 %d 个项目\n", len(programs))
	}

	for _, p := range programs {
		output <- utils.NewScope{NewPublicURL: p.URL}
		for _, target := range p.Targets.InScope {
			// if utils.In(target.Type, []string{"url", "domain", "wildcard", "other"}) {
			// 	output <- utils.NewScope{NewTarget: target.Endpoint}
			// } else {
			// 	output <- utils.NewScope{NewApp: target.Endpoint}
			// }

			if utils.In(target.Type, []string{"url", "domain", "wildcard", "other"}){
				e.handleDomainIdentifier(target.Endpoint, output)
			} else {
				if strings.HasPrefix(target.Endpoint, "*") || strings.HasSuffix(target.Endpoint, "*") {
					e.handleDomainIdentifier(target.Endpoint, output)
				} else {
					e.handleAsset(target.Endpoint, output)
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

type IntigritiTry struct {
	Url      string          `json:"url"`
	Programs []Intigriti     `json:"programs"`
	Config   utils.Intigriti `json:"config"`
}

func NewIntigritiTry(config utils.Intigriti) *IntigritiTry {
	return &IntigritiTry{
		Programs: []Intigriti{},
		Config:   config,
	}
}

func (i IntigritiTry) ProgramRquest(target_url string) (body []byte, err error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", target_url, nil)
	if err != nil {
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	body, err = ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		err = fmt.Errorf(resp.Status)
	}
	return
}

func (i IntigritiTry) FindByClass(n *html.Node, className string) (elements []*html.Node) {
	if n.Type == html.ElementNode && n.Data == "div" {
		for _, attr := range n.Attr {
			if attr.Key == "class" && attr.Val == className {
				elements = append(elements, n)
				return
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		elements = append(elements, i.FindByClass(c, className)...)
	}
	return
}

func (i IntigritiTry) GetText(n *html.Node) (content string) {
	if n.Type == html.TextNode {
		content = strings.TrimSpace(n.Data)
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		content = i.GetText(c)
		if content != "" {
			break
		}
	}
	return
}

func (i IntigritiTry) BuildId() (tag string, err error) {
	resp, err := http.Get("https://www.intigriti.com/program")
	if err != nil {
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	re := regexp.MustCompile(`/_next/static/([^/]+)/_buildManifest\.js`)
	match := re.FindStringSubmatch(string(body))
	if len(match) > 1 {
		tag = match[1]
	}
	return
}

func (i IntigritiTry) Program() (programs []Intigriti) {
	var new_program []Intigriti
	new_intigriti_program := make(chan Intigriti)
	semaphore := make(chan struct{}, 30)

	tag, err := i.BuildId()
	if err != nil {
		fmt.Println("intigriti 获取 BuildId 失败", err)
	}

	url := fmt.Sprintf("https://www.intigriti.com/_next/data/%s/en/programs.json", tag)
	res_data, err := i.ProgramRquest(url)
	if err != nil {
		fmt.Println("intigriti 获取 programs 失败", err)
		return
	}

	result := gjson.GetBytes(res_data, "pageProps.programs")
	json.Unmarshal([]byte(result.Raw), &new_program)

	var wg sync.WaitGroup
	for _, item := range new_program {
		wg.Add(1)
		if item.ConfidentialityLevel == 4 {
			go i.Scope(item, new_intigriti_program, semaphore, &wg)
		} else {
			wg.Done()
		}
	}

	for {
		select {
		case program := <-new_intigriti_program:
			programs = append(programs, program)
		case <-time.After(3 * time.Second):
			wg.Wait()
			return
		}
	}
}

func (i IntigritiTry) Scope(intigriti Intigriti, new_intigriti_program chan Intigriti, semaphore chan struct{}, wg *sync.WaitGroup) (in_scopes []IntigritiScope, out_scopes []IntigritiScope) {
	defer wg.Done()
	semaphore <- struct{}{}

	url := fmt.Sprintf("https://app.intigriti.com/programs/%s/%s/detail", intigriti.Handle, intigriti.Handle)
	res_data, err := i.ProgramRquest(url)
	if err != nil {
		fmt.Println("intigriti 获取 target 失败", err)
		<-semaphore
		new_intigriti_program <- intigriti
		return
	}

	doc, _ := html.Parse(strings.NewReader(string(res_data)))
	container := i.FindByClass(doc, "domain-container")

	for _, item := range container {
		domain_endpoint := i.FindByClass(item, "domainEndpoint")
		domain_type := i.FindByClass(item, "domainType")
		impact_type := i.FindByClass(item, "impact")

		new_scope := IntigritiScope{
			Endpoint: i.GetText(domain_endpoint[0]),
			Impact:   i.GetText(impact_type[0]),
			Type:     i.GetText(domain_type[0]),
		}

		if strings.Contains(new_scope.Impact, "Out") {
			out_scopes = append(out_scopes, new_scope)
		} else {
			in_scopes = append(in_scopes, new_scope)
		}
	}

	intigriti.Targets.InScope = in_scopes
	intigriti.Targets.OutOfScope = out_scopes
	<-semaphore
	new_intigriti_program <- intigriti
	return
}
