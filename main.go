package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/edsrzf/mmap-go"

	"golang.org/x/sys/unix"
)

var bugcrowdurl = "https://raw.githubusercontent.com/arkadiyt/bounty-targets-data/main/data/bugcrowd_data.json"
var hackeroneurl = "https://raw.githubusercontent.com/arkadiyt/bounty-targets-data/main/data/hackerone_data.json"

var source_path = filepath.Join(user_home_dir(), ".config/bountytr/")

type HackeroneScope struct {
	AssetIdentifier           string `json:"asset_identifier"`
	AssetType                 string `json:"asset_type"`
	AvailabilityRequirement   string `json:"availability_requirement"`
	ConfdentialityRequirement string `json:"confidentiality_requirement"`
	EligibleForBounty         bool   `json:"eligible_for_bounty"`
	EligibleForSubmission     bool   `json:"eligible_for_submission"`
	Instruction               string `json:"instruction"`
	IntegrityRequirement      string `json:"integrity_requirement"`
	MaxSeverity               string `json:"max_severity"`
}

type HackeroneTarget struct {
	InScope    []HackeroneScope `json:"in_scope"`
	OutOfScope []HackeroneScope `json:"out_of_scope"`
}

type Hackerone struct {
	AllowsBountySplitting bool `json:"allows_bounty_splitting"`
	// AverageTimeToBountyAwarded        string          `json:"average_time_to_bounty_awarded"`
	AverageTimeToFirstProgramResponse float32 `json:"average_time_to_first_program_response"`
	// AverageTimeToReportResolved       string          `json:"average_time_to_report_resolved"`
	Handle                       string          `json:"phandle"`
	ManagedProgram               bool            `json:"managed_program"`
	OffersBounties               bool            `json:"offers_bounties"`
	OffersSwag                   bool            `json:"offers_swag"`
	Name                         string          `json:"name"`
	ResponseEfficiencyPercentage int64           `json:"response_efficiency_percentage"`
	SubmissionState              string          `json:"submission_state"`
	Url                          string          `json:"url"`
	Website                      string          `json:"website"`
	Targets                      HackeroneTarget `json:"targets"`
}

type BugcrowdScope struct {
	Type   string `json:"type"`
	Target string `json:"target"`
}

type BugcrowdTarget struct {
	InScope    []BugcrowdScope `json:"in_scope"`
	OutOfScope []BugcrowdScope `json:"out_of_scope"`
}

type Bugcrowd struct {
	Name              string         `json:"name"`
	Url               string         `json:"url"`
	AllowsDisclosure  bool           `json:"allows_disclosure"`
	ManagedByBugcrowd bool           `json:"managed_by_bugcrowd"`
	SafeHarbor        string         `json:"safe_harbor"`
	MaxPayout         int64          `json:"max_payout"`
	Targets           BugcrowdTarget `json:"targets"`
}

type Bounty interface {
	HackeroneTarget | BugcrowdTarget
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
func bugcrowd(source_targets map[string]bool) (new_targets []string) {

	var body = BountyTarget(bugcrowdurl)

	// 解析JSON数据
	var targets []Bugcrowd
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
					if !source_targets[domain] {
						fmt.Println(domain)
						new_targets = append(new_targets, domain)
					}
				}
			}

		}

	}

	return

}

func hackerone(source_targets map[string]bool) (new_targets []string) {

	var body = BountyTarget(hackeroneurl)

	// 解析JSON数据
	var targets []Hackerone
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
					if !source_targets[domain] {
						fmt.Println(domain)
						new_targets = append(new_targets, domain)
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
	fmt.Println(string(banner))

	var cycle_time int64

	// now := time.Now().Format("2006-01-02 15:04:05")

	flag.Int64Var(&cycle_time, "t", 30, "监控周期(分钟)")

	// 解析命令行参数写入注册的flag里
	flag.Parse()

	fmt.Println("[*] Starting tracker", "... ")

	os.MkdirAll(source_path, os.ModePerm)

	// 启动定时任务
	tasks := []*Task{
		NewTask("tracker", time.Duration(cycle_time)*time.Minute, func() {

			// 读取源目标文件
			source_targets := read_file_to_map(filepath.Join(source_path, "domain.txt"))

			// 获取新增赏金目标
			new_hackerone_targets := hackerone(source_targets)
			new_bugcrowd_targets := bugcrowd(source_targets)

			// 保存新增目标
			save_targets_to_file(filepath.Join(source_path, "domain.txt"), append(new_hackerone_targets, new_bugcrowd_targets...))
		}),
	}

	for _, task := range tasks {
		go task.Run()
	}

	// 等待任务结束
	select {}

}

// // 目标对比
// func compared_target(latest_targets []string) {
// 	// 获取新资产目标，对比出新增资产

// }

func in(target string, str_array []string) bool {
	sort.Strings(str_array)
	index := sort.SearchStrings(str_array, target)
	if index < len(str_array) && str_array[index] == target {
		return true
	}
	return false
}

func domain_match(url string) []string {

	domain_rege := regexp.MustCompile(`[a-zA-Z0-9][-a-zA-Z0-9]{0,62}(\.[a-zA-Z0-9][-a-zA-Z0-9]{0,62})+`)

	return domain_rege.FindAllString(url, -1)
}

func read_file_to_map(filename string) map[string]bool {
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
	usr, err := user.Current()
	if err != nil {
		fmt.Println("Could not get user home directory:", err)
	}
	return usr.HomeDir
}