package hackerone

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/baiqll/bountytr/pkg/proxypool"
	"github.com/baiqll/bountytr/pkg/utils"
)

type HackeroneTry struct {
	// Url      string             `json:"url"`
	Programs    []ProgramsScope `json:"programs"`
	Config 		utils.HackerOne    `json:"config"`
	Pool        proxypool.Pool     `json:"pool"`
}

func NewHackeroneTry(config utils.HackerOne, pool proxypool.Pool) *HackeroneTry {

	return &HackeroneTry{
		Programs:    []ProgramsScope{},
		Config: config,
		Pool:        pool,
	}
}

func (h HackeroneTry) ProgramRquest(link string) (body []byte, err error) {
	/*
		hackerone 请求体
	*/

	proxyUrl, _ := url.Parse(h.Pool.RandProxy())

	transport := http.Transport{
		Proxy: func(*http.Request) (*url.URL, error) {
			if proxyUrl.Host == ""{
				return nil, nil // 总是返回 nil，不使用代理
			}
			return proxyUrl, nil
		},
	}

	client := &http.Client{
		// Timeout:   10 * time.Second,
		Transport: &transport,
	}

	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return
	}

	// 设置token 
	req.SetBasicAuth(h.Config.Private.APIName, h.Config.Private.APIToken)

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return
	}

	// 读取响应体
	body, err = ioutil.ReadAll(resp.Body)

	return
}

func (h HackeroneTry) ProgramsScope(new_hackerone chan utils.NewScope) {
	/*
		获取项目列表
	*/
	var wg sync.WaitGroup

	// new_programs_scope := make(chan ProgramsScope) // 创建缓冲通道

	link := "https://api.hackerone.com/v1/hackers/programs?page[size]=100"
	
	programsDatas := h.GetPrograms(link)

	wg.Add(len(programsDatas))

	/*
		public_mode 公共项目
		soft_launched 私人项目
	*/


	for _, data := range programsDatas {

		if data.Attributes.State == "soft_launched"{
			new_hackerone <- utils.NewScope{NewPrivateURL: "https://hackerone.com/" + data.Attributes.Handle}
		}else{
			new_hackerone <- utils.NewScope{NewPublicURL: "https://hackerone.com/" + data.Attributes.Handle}
		}

		

		go h.GetScope(data.Attributes.Handle, new_hackerone, &wg)

		numGoroutines := runtime.NumGoroutine()

		if numGoroutines > h.Config.Concurrency {
			time.Sleep(3 * time.Second)
		}

	}

	go func() {
		wg.Wait()
	}()

}

func (h HackeroneTry) GetPrograms(link string) (programsData []*ProgramsData) {

	/*
		获取所有项目
	*/

	if link == "nil"{
		return
	}

	var new_programs Programs
	
	res_data, err := h.ProgramRquest(link)
	if err != nil {
		fmt.Println("hackerone Program 请求失败", err)
		return
	}

	err = json.Unmarshal([]byte(res_data), &new_programs)

	if err != nil {
		fmt.Println("hackerone Program 请求失败",err)
	}
	if new_programs.Links == nil || new_programs.Data == nil{
		
		return
	}

	programsData = append(programsData, new_programs.Data...)

	if new_programs.Links.Next != ""{

		new_programs_data := h.GetPrograms(new_programs.Links.Next)

		programsData = append(programsData, new_programs_data...)
	
	}

	return
	
}


func (h HackeroneTry) GetScope(handle string, new_hackerone chan utils.NewScope, wg *sync.WaitGroup) {
	/*
		获取项目赏金目标
	*/
	defer wg.Done()
	
	link := fmt.Sprintf("https://api.hackerone.com/v1/hackers/programs/%s/structured_scopes?page[size]=100", handle)

	var scope Scope

	res_data, err := h.ProgramRquest(link)
	if err != nil {
		fmt.Println("hackerone Scope 获取失败", err)
	
		return
	}

	err = json.Unmarshal([]byte(res_data), &scope)
	if err != nil {
		fmt.Println("hackerone Scope 解析失败", string(res_data))

		return
	}

	h.ProcessScope(scope, new_hackerone)

}


func  (h HackeroneTry) ProcessScope(scope Scope, new_hackerone chan utils.NewScope){


	for _, asset := range scope.Data {

		scope_attr := asset.Attributes
		
		if !asset.Attributes.EligibleForBounty {
			// Skip out of scope
			continue
		}

		identifier := scope_attr.Identifier
		assetType := scope_attr.AssetType

		if utils.In(assetType,[]string{"DOMAIN","URL","WILDCARD"}){

			h.HandleDomainIdentifier(identifier, new_hackerone)

		} else if  assetType == "OTHER" {
			if strings.HasPrefix(identifier, "*")|| strings.HasSuffix(identifier, "*") {
				h.HandleDomainIdentifier(identifier, new_hackerone)
			} else {
				h.HandleAsset(identifier, new_hackerone)
			}
		}else {
			h.HandleAsset(identifier, new_hackerone)
		}

	}

}

func (h HackeroneTry) CleanDomain(domain string) string {

	pattern := `[\w]+[\w\-_~\.]+\.[a-zA-Z]+|$`
	// pattern := `[\w]+[\w\-_~\.]*\.[a-zA-Z]+(\/[\w\-_~\.]+)*`
	r, err := regexp.Compile(pattern)
	if err != nil {
		// Whatever happened, just return the original domain
		return domain
	}

	cDomain := r.FindString(domain)
	if cDomain != "" {
		return cDomain
	}
	return domain
}

func  (h HackeroneTry) DomainSplitTrimSpace(domain string) []string {
	domainSlice := strings.Split(domain, ",")
	for i := range domainSlice {
		domainSlice[i] = strings.TrimSpace(domainSlice[i])
	}

	return domainSlice
}

func (h HackeroneTry) HandleAsset(identifier string, new_hackerone chan utils.NewScope) {
	domainsSlice := h.DomainSplitTrimSpace(identifier)
	for _, identifier := range domainsSlice {
		
		new_hackerone <- utils.NewScope{NewApp: identifier}
		
	}
}

func (h HackeroneTry) HandleDomainIdentifier(identifier string, new_hackerone chan utils.NewScope){

	identifier = h.CleanDomain(identifier)
	domainsSlice := h.DomainSplitTrimSpace(identifier)
	for _, identifier := range domainsSlice {
		
		new_hackerone <- utils.NewScope{NewTarget: identifier}
	
	}

}


