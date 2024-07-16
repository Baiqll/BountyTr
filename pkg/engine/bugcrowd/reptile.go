package bugcrowd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/baiqll/bountytr/pkg/proxypool"
	"github.com/baiqll/bountytr/pkg/utils"
	"github.com/tidwall/gjson"
)

type BugcrowdTry struct {
	// Url        string            `json:"url"`
	Programs    []Bugcrowd `json:"programs"`
	Config  	utils.Bugcrowd    `json:"config"`
	Pool        proxypool.Pool    `json:"pool"`
}

func NewBugcrowdTry(config utils.Bugcrowd, pool proxypool.Pool) *BugcrowdTry {

	return &BugcrowdTry{
		// Url:        url,
		Programs:    []Bugcrowd{},
		Config: config,
		Pool:        pool,
	}
}

func (b BugcrowdTry) ProgramJson(path string) (body []byte, err error) {

	proxyUrl, _ := url.Parse(b.Pool.RandProxy())

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyUrl),
	}

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}

	url := "https://bugcrowd.com" + path

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}

	// 设置请求头
	req.Header.Set("Accept", "*/*")

	// 发送请求
	// resp, err := http.DefaultClient.Do(req)
	resp, err := client.Do(req)
	if err != nil {
		return
	}

	// 处理响应
	defer resp.Body.Close()
	// 读取响应体
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
	// current_page = gjson.GetBytes(res_data, "meta.currentPage").Int()
	program_result := gjson.GetBytes(res_data, "programs")

	err = json.Unmarshal([]byte(program_result.Raw), &page_program)

	return

}

func (b BugcrowdTry) Program() (programs []Bugcrowd) {
	/*
		获取项目列表
	*/

	var new_program []Bugcrowd
	new_bugcrowd_program := make(chan Bugcrowd) // 创建缓冲通道
	semaphore := make(chan struct{}, b.Config.Concurrency)    // 最高并发数

	// 获取第一页信息(获取总页数 total_page)
	total_page, new_program, err := b.ProgramPage(1)
	if err != nil {
		fmt.Println("bugcrowd 获取programs 失败", err)
		return
	}

	// 获取余下所有页面信息
	var wgp sync.WaitGroup
	wgp.Add(int(total_page) - 1) // 初始化等待组计数器

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

	/*
		获取项目具体 目标
	*/

	var wg sync.WaitGroup

	for _, item := range new_program {
		wg.Add(1)

		item.Url = "https://bugcrowd.com" + item.ProgramUrl

		if item.InvitedStatus != "open" || item.Participation == "private" {
			// 未开启，或者私密项目
			wg.Done()
			continue
		}

		go b.Scope(item, new_bugcrowd_program, semaphore, &wg)

	}

	// 从缓冲通道读取数据
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
	/*
		获取项目赏金目标
	*/
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
	// out_result := gjson.GetBytes(target_data, "groups.#(in_scope==false)#.targets_url")

	for _, item := range in_result.Array() {
		new_in_scopes, _ := b.Target(item.Str)
		in_scopes = append(in_scopes, new_in_scopes...)
	}

	// for _, item := range out_result.Array() {
	// 	new_out_scopes, _ := b.Target(item.Str)
	// 	out_scopes = append(out_scopes, new_out_scopes...)
	// }

	bugcrowd.Targets.InScope = in_scopes
	// bugcrowd.Targets.OutOfScope = out_scopes

	<-semaphore

	new_bugcrowd_program <- bugcrowd

	// fmt.Printf("【%s】\n", bugcrowd.ProgramUrl)

	return

}
