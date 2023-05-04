package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
)

var bugcrowdurl = "https://raw.githubusercontent.com/arkadiyt/bounty-targets-data/main/data/bugcrowd_data.json"
var hackeroneurl = "https://raw.githubusercontent.com/arkadiyt/bounty-targets-data/main/data/hackerone_data.json"

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
func bugcrowd() {

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
					fmt.Println(domain)
				}
			}

		}

	}

}

func hackerone() {

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
					fmt.Println(domain)
				}

			}
		}
	}
}

func main() {

	fmt.Println("New Bounty Target ....")
	hackerone()
	bugcrowd()

}

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
