package programs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/baiqll/bountytr/src/models"
	"github.com/tidwall/gjson"
)

type BugcrowdTry struct {
	Url      string            `json:"url"`
	Programs []models.Bugcrowd `json:"programs"`
}

func NewBugcrowdTry(url string) *BugcrowdTry {

	return &BugcrowdTry{
		Url:      url,
		Programs: []models.Bugcrowd{},
	}
}

func (b BugcrowdTry) ProgramJson(path string) (body []byte, err error) {

	url := "https://bugcrowd.com" + path

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}

	// 设置请求头
	req.Header.Set("Accept", "*/*")

	// 发送请求
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	// 处理响应
	defer resp.Body.Close()
	// 读取响应体
	body, err = ioutil.ReadAll(resp.Body)

	return
}

func (b BugcrowdTry) Program() (programs []models.Bugcrowd) {
	/*
		获取项目列表
	*/

	page := 1
	var total_page int

	for {

		res_data, err := b.ProgramJson(fmt.Sprintf("/programs.json?vdp[]=false&page[]=%d", page))
		if err != nil {
			fmt.Println("bugcrowd 获取programs 失败", err)
			return
		}

		if total_page == 0 {
			total_page = int(gjson.GetBytes(res_data, "meta.totalPages").Int())
		}
		if page > total_page {
			break
		}

		program_result := gjson.GetBytes(res_data, "programs")

		var new_programs []models.Bugcrowd

		err = json.Unmarshal([]byte(program_result.Raw), &new_programs)

		if err != nil {
			fmt.Println("bugcrowd 解析失败")
			return
		}

		programs = append(programs, new_programs...)

		page += 1

	}

	for _, item := range programs {

		item.Url = "https://bugcrowd.com" + item.ProgramUrl

		if item.InvitedStatus != "open" || item.Participation == "private" {
			// 未开启，或者私密项目
			continue
		}

		item.Targets.InScope, item.Targets.OutOfScope = b.Scope(item.ProgramUrl)

		b.Programs = append(b.Programs, item)

	}

	programs = b.Programs

	return

}

func (b BugcrowdTry) Target(url string) (scope []models.BugcrowdScope, err error) {

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

func (b BugcrowdTry) Scope(programUrl string) (in_scopes []models.BugcrowdScope, out_scopes []models.BugcrowdScope) {
	/*
		获取项目赏金目标
	*/

	target_data, err := b.ProgramJson(programUrl + "/target_groups")
	if err != nil {
		fmt.Println("bugcrowd 获取target_groups 失败")
		return
	}

	in_result := gjson.GetBytes(target_data, "groups.#(in_scope==true).targets_url")
	out_result := gjson.GetBytes(target_data, "groups.#(in_scope==false).targets_url")

	in_scopes, _ = b.Target(in_result.Str)
	out_scopes, _ = b.Target(out_result.Str)

	return

}
