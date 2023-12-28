package programs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/baiqll/bountytr/src/models"
	"github.com/tidwall/gjson"
	"golang.org/x/net/html"
)

type IntigritiTry struct {
	Url      string             `json:"url"`
	Programs []models.Intigriti `json:"programs"`
}

func NewIntigritiTry(url string) *IntigritiTry {

	return &IntigritiTry{
		Url:      url,
		Programs: []models.Intigriti{},
	}
}

func (i IntigritiTry) ProgramRquest(url string) (body []byte, err error) {

	// 请求JSON数据
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	// 读取响应体
	body, err = ioutil.ReadAll(resp.Body)

	return
}

func (i IntigritiTry) BuildId() (tag string, err error) {

	// 请求JSON数据
	resp, err := http.Get("https://www.intigriti.com/program")
	if err != nil {
		return
	}

	// 读取响应体
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

func (i IntigritiTry) Program() (programs []models.Intigriti) {

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

	json.Unmarshal([]byte(result.Raw), &programs)

	for _, item := range programs {

		if item.ConfidentialityLevel == 4 {
			in_scopes, out_scopes := i.Scope(item.Handle)

			item.Targets.InScope = in_scopes
			item.Targets.OutOfScope = out_scopes
			i.Programs = append(i.Programs, item)
		}
	}

	programs = i.Programs
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

func (i IntigritiTry) Scope(handle string) (in_scopes []models.IntigritiScope, out_scopes []models.IntigritiScope) {
	/*
		获取项目赏金目标
	*/

	url := fmt.Sprintf("https://app.intigriti.com/programs/%s/%s/detail", handle, handle)

	res_data, err := i.ProgramRquest(url)
	if err != nil {
		fmt.Println("intigriti 获取 target 失败", err)
		return
	}

	doc, err := html.Parse(strings.NewReader(string(res_data)))
	if err != nil {
		fmt.Println(err)
		return
	}

	container := i.FindByClass(doc, "domain-container")

	for _, item := range container {
		domain_endpoint := i.FindByClass(item, "domainEndpoint")
		domain_type := i.FindByClass(item, "domainType")
		impact_type := i.FindByClass(item, "impact")

		new_scope := models.IntigritiScope{
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

	return

}
