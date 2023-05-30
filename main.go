package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dlclark/regexp2"

	"github.com/edsrzf/mmap-go"

	"github.com/baiqll/bountytr/src/models"

	"github.com/projectdiscovery/dnsx/libs/dnsx"
	"golang.org/x/sys/unix"
)

var bugcrowdurl = "https://raw.githubusercontent.com/arkadiyt/bounty-targets-data/main/data/bugcrowd_data.json"
var hackeroneurl = "https://raw.githubusercontent.com/arkadiyt/bounty-targets-data/main/data/hackerone_data.json"
var intigritiurl = "https://raw.githubusercontent.com/arkadiyt/bounty-targets-data/main/data/intigriti_data.json"

var blacklist = []string{
	".gov",
	".edu",
	".json",
	".[0-9.]+$",
}

var source_path = filepath.Join(user_home_dir(), ".config/bountytr/")

type Bounty interface {
	models.HackeroneTarget | models.BugcrowdTarget | models.IntigritiTarget
}

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

func BountyTarget(url string) []byte {

	// 请求JSON数据
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return body

}

func bugcrowd(source_targets map[string]bool, fail_targets map[string]bool) (error_targets []string, new_targets []string) {

	var body = BountyTarget(bugcrowdurl)

	// 解析JSON数据
	var targets []models.Bugcrowd
	err := json.Unmarshal(body, &targets)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, target := range targets {

		// 判断是否有赏金
		if target.MaxPayout <= 0 {
			continue
		}

		for _, scope := range target.Targets.InScope {

			// 只打印 Web 目标
			if in(scope.Type, []string{"api", "website"}) {
				for _, domain := range domain_match(scope.Target) {
					if !source_targets[domain] && !in(domain, new_targets) && !fail_targets[domain] {
						if domain_valid(domain) && !strings.Contains(scope.Target, `\*`) {
							fmt.Println(domain)
							new_targets = append(new_targets, domain)
						} else {
							error_targets = append(error_targets, domain)
						}
					}
				}
			}
		}
	}

	return
}

func hackerone(source_targets map[string]bool, fail_targets map[string]bool) (error_targets []string, new_targets []string) {

	var body = BountyTarget(hackeroneurl)

	// 解析JSON数据
	var targets []models.Hackerone
	err := json.Unmarshal(body, &targets)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, target := range targets {

		// 判断是否有赏金
		if !target.OffersBounties {
			continue
		}

		for _, scope := range target.Targets.InScope {

			// 只打印 Web 目标
			if in(scope.AssetType, []string{"URL", "WILDCARD"}) {
				for _, domain := range domain_match(scope.AssetIdentifier) {
					if !source_targets[domain] && !in(domain, new_targets) && !fail_targets[domain] {
						if domain_valid(domain) && !strings.Contains(scope.AssetIdentifier, `\*`) {
							fmt.Println(domain)
							new_targets = append(new_targets, domain)
						} else {
							error_targets = append(error_targets, domain)
						}
					}
				}
			}

			// 其他
			if in(scope.AssetType, []string{"OTHER"}) {
				for _, domain := range domain_match(scope.AssetIdentifier) {
					if !source_targets[domain] && !in(domain, new_targets) && !fail_targets[domain] {
						if !domain_valid(domain) && !strings.Contains(scope.AssetIdentifier, `\*`) {
							fmt.Println(domain)
							new_targets = append(new_targets, domain)
						} else {
							error_targets = append(error_targets, domain)
						}
					}
				}
				for _, domain := range domain_match(scope.Instruction) {
					if !source_targets[domain] && !in(domain, new_targets) && !fail_targets[domain] {
						if domain_valid(domain) && !strings.Contains(scope.Instruction, `\*`) {
							fmt.Println(domain)
							new_targets = append(new_targets, domain)
						} else {
							error_targets = append(error_targets, domain)
						}
					}
				}
			}
		}
	}

	return
}

func intigriti(source_targets map[string]bool, fail_targets map[string]bool) (error_targets []string, new_targets []string) {

	var body = BountyTarget(intigritiurl)

	// 解析JSON数据
	var targets []models.Intigriti
	err := json.Unmarshal(body, &targets)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, target := range targets {

		// 判断是否有赏金
		if target.MaxBounty.Value <= 0 {
			continue
		}

		for _, scope := range target.Targets.InScope {

			// 只打印 Web 目标
			if in(scope.Type, []string{"url"}) {
				for _, domain := range domain_match(scope.Endpoint) {
					if !source_targets[domain] && !in(domain, new_targets) && !fail_targets[domain] {
						if domain_valid(domain) && !strings.Contains(scope.Endpoint, `\*`) {
							fmt.Println(domain)
							new_targets = append(new_targets, domain)
						} else {
							error_targets = append(error_targets, domain)
						}
					}
				}
			}
		}
	}

	return
}

func main() {

	var banner = `

         __                      __        __      
        / /_  ____  __  ______  / /___  __/ /______
       / __ \/ __ \/ / / / __ \/ __/ / / / __/ ___/
      / /_/ / /_/ / /_/ / / / / /_/ /_/ / /_/ /    
     /_.___/\____/\__,_/_/ /_/\__/\__, /\__/_/     
                                 /____/       v1.0       

   
	Keep track of bounty targets
    `

	var cycle_time int64
	var silent bool

	flag.Int64Var(&cycle_time, "t", 0, "监控周期(分钟)")
	flag.BoolVar(&silent, "silent", false, "是否静默状态")

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
				run(silent)
			}),
		}
		for _, task := range tasks {
			go task.Run()
		}
		// 等待任务结束
		select {}
	} else {
		run(silent)
	}
}

func run(silent bool) {

	if !silent {
		now := time.Now().Format("2006-01-02 15:04:05")

		fmt.Println("[*] Date:", now)
	}

	// 读取源目标文件
	source_targets := read_file_to_map(filepath.Join(source_path, "domain.txt"))
	fail_targets := read_file_to_map(filepath.Join(source_path, "faildomain.txt"))

	// 获取新增赏金目标
	new_hackerone_fail_targets, new_hackerone_targets := hackerone(source_targets, fail_targets)
	new_bugcrowd_fail_targets, new_bugcrowd_targets := bugcrowd(source_targets, fail_targets)
	new_intigriti_fail_targets, new_intigriti_targets := intigriti(source_targets, fail_targets)

	// 保存新增目标
	save_targets_to_file(filepath.Join(source_path, "domain.txt"), append(append(new_hackerone_targets, new_bugcrowd_targets...), new_intigriti_targets...))

	save_targets_to_file(filepath.Join(source_path, "faildomain.txt"), append(append(new_hackerone_fail_targets, new_bugcrowd_fail_targets...), new_intigriti_fail_targets...))

}

func new_goal_reminder(new_targets []string) {

}

func in(target string, str_array []string) bool {
	// 判断字符串是否 存在于字符串数组内
	sort.Strings(str_array)
	index := sort.SearchStrings(str_array, target)
	if index < len(str_array) && str_array[index] == target {
		return true
	}
	return false
}

func domain_match(url string) []string {
	// 提取域名

	// 黑名单正则
	var black_pattern []string
	for _, black := range blacklist {

		black_pattern = append(black_pattern, fmt.Sprintf(".*%s", black))
	}

	// 特殊过滤
	// black_pattern = append(black_pattern, filterlist...)
	pattern := fmt.Sprintf(`(?!%s)[a-zA-Z0-9][-a-zA-Z0-9]{0,62}(\.[a-zA-Z0-9][-a-zA-Z0-9]{0,62})+`, strings.Join(black_pattern, "|"))

	domain_rege := regexp2.MustCompile(pattern, 0)
	// domain_rege := regexp.MustCompile(`^(?!.*gov|.*edu)[a-zA-Z0-9][-a-zA-Z0-9]{0,62}(\.[a-zA-Z0-9][-a-zA-Z0-9]{0,62})+`)

	// return dedupe_from_list(domain_rege.FindAllString(url, -1))
	return dedupe_from_list(regexp2FindAllString(domain_rege, url))
}

func read_file_to_map(filename string) map[string]bool {
	// 读取文件到 map

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}
	size := info.Size()

	if size == 0 {
		if _, err := file.WriteString("\n"); err != nil {
			log.Fatal(err)
		}
	}

	hash := make(map[string]bool)
	reader := bufio.NewReader(file)
	mm, err := mmap.Map(file, unix.PROT_READ, 0)
	if err != nil {
		log.Fatal(err)
	}
	defer mm.Unmap()

	for i := 0; int64(i) < size; i++ {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		hash[strings.Replace(line, "\n", "", -1)] = true
	}
	return hash
}

func save_targets_to_file(filename string, targets []string) {

	// 保存目标到文件内

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, target := range targets {
		writer.WriteString(target + "\n")
	}
	writer.Flush()
	file.Sync()

}
func user_home_dir() string {
	// 获取 $home 路径
	usr, err := user.Current()
	if err != nil {
		fmt.Println("Could not get user home directory:", err)
	}
	return usr.HomeDir
}

func dedupe_from_list(source []string) []string {
	// 列表去重
	var new_list []string

	dedupe_set := make(map[string]bool)
	for _, v := range source {
		dedupe_set[v] = true
	}

	for k := range dedupe_set {

		new_list = append(new_list, k)
	}

	return new_list
}

func regexp2FindAllString(re *regexp2.Regexp, s string) []string {
	// 正则匹配提取
	var matches []string
	m, _ := re.FindStringMatch(s)
	for m != nil {
		matches = append(matches, m.String())
		m, _ = re.FindNextMatch(m)
	}
	return matches
}

func domain_valid(domain string) bool {
	// 检查domain是否符合url格式
	_, err := url.Parse("http://" + domain)
	if err != nil {
		return false
	}

	// DNS 查询
	dnsClient, _ := dnsx.New(dnsx.DefaultOptions)

	// DNS 查询 A 记录
	result, _ := dnsClient.Lookup(domain)

	if len(result) == 0 {
		return false
	}

	return true
}
