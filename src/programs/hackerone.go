package programs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/baiqll/bountytr/src/models"
	"github.com/tidwall/gjson"
)

type HackeroneTry struct {
	Url      string             `json:"url"`
	Programs []models.Hackerone `json:"programs"`
}

func NewHackeroneTry(url string) *HackeroneTry {

	return &HackeroneTry{
		Url:      url,
		Programs: []models.Hackerone{},
	}
}

func (h HackeroneTry) ProgramGraphql(data *bytes.Buffer) (body []byte, err error) {

	// 请求JSON数据
	resp, err := http.Post("https://hackerone.com/graphql", "application/json", data)
	if err != nil {
		return
	}

	// 读取响应体
	body, err = ioutil.ReadAll(resp.Body)

	return
}

func (h HackeroneTry) Program() (programs []models.Hackerone) {
	/*
		获取项目列表
	*/
	data := bytes.NewBufferString(`{"operationName":"DirectoryQuery","variables":{"where":{"_and":[{"_or":[{"offers_bounties":{"_eq":true}},{"external_program":{"offers_rewards":{"_eq":true}}}]},{"_or":[{"submission_state":{"_eq":"open"}},{"submission_state":{"_eq":"api_only"}},{"external_program":{}}]},{"_not":{"external_program":{}}},{"_or":[{"_and":[{"state":{"_neq":"sandboxed"}},{"state":{"_neq":"soft_launched"}}]},{"external_program":{}}]}]},"first":1000,"secureOrderBy":{"launched_at":{"_direction":"DESC"}},"product_area":"directory","product_feature":"programs"},"query":"query DirectoryQuery($cursor: String, $secureOrderBy: FiltersTeamFilterOrder, $where: FiltersTeamFilterInput) {\n  me {\n    id\n    edit_unclaimed_profiles\n    __typename\n  }\n  teams(first: 1000, after: $cursor, secure_order_by: $secureOrderBy, where: $where) {\n    pageInfo {\n      endCursor\n      hasNextPage\n      __typename\n    }\n    edges {\n      node {\n        id\n        bookmarked\n        ...TeamTableResolvedReports\n        ...TeamTableAvatarAndTitle\n        ...TeamTableLaunchDate\n        ...TeamTableMinimumBounty\n        ...TeamTableAverageBounty\n        ...BookmarkTeam\n        __typename\n      }\n      __typename\n    }\n    __typename\n  }\n}\n\nfragment TeamTableResolvedReports on Team {\n  id\n  resolved_report_count\n  __typename\n}\n\nfragment TeamTableAvatarAndTitle on Team {\n  id\n  profile_picture(size: medium)\n  name\n  handle\n  submission_state\n  triage_active\n  publicly_visible_retesting\n  state\n  allows_bounty_splitting\n  external_program {\n    id\n    __typename\n  }\n  ...TeamLinkWithMiniProfile\n  __typename\n}\n\nfragment TeamLinkWithMiniProfile on Team {\n  id\n  handle\n  name\n  __typename\n}\n\nfragment TeamTableLaunchDate on Team {\n  id\n  launched_at\n  __typename\n}\n\nfragment TeamTableMinimumBounty on Team {\n  id\n  currency\n  base_bounty\n  __typename\n}\n\nfragment TeamTableAverageBounty on Team {\n  id\n  currency\n  average_bounty_lower_amount\n  average_bounty_upper_amount\n  __typename\n}\n\nfragment BookmarkTeam on Team {\n  id\n  bookmarked\n  __typename\n}\n"}`)

	res_data, err := h.ProgramGraphql(data)
	if err != nil {
		fmt.Println("hackerone Program 请求失败", err)
		return
	}

	result := gjson.GetBytes(res_data, "data.teams.edges.#.node")

	var new_programs []models.Hackerone

	err = json.Unmarshal([]byte(result.Raw), &new_programs)
	if err != nil {
		fmt.Println("hackerone Program 解析失败", err)
		return
	}

	for _, item := range new_programs {

		item.Url = "https://hackerone.com/" + item.Handle

		if item.SubmissionState != "open" {
			continue
		}

		item.Targets.InScope, item.Targets.OutOfScope = h.Scope(item.Handle)

		h.Programs = append(h.Programs, item)

	}

	programs = h.Programs

	return

}

func (h HackeroneTry) Scope(handle string) (in_scopes []models.HackeroneScope, out_scopes []models.HackeroneScope) {
	/*
		获取项目赏金目标
	*/

	data := bytes.NewBufferString(fmt.Sprintf(`{"operationName":"TeamAssets","variables":{"handle":"%s","product_area":"team_profile","product_feature":"overview"},"query":"query TeamAssets($handle: String!) {\n  me {\n    id\n    membership(team_handle: $handle) {\n      id\n      permissions\n      __typename\n    }\n    __typename\n  }\n  team(handle: $handle) {\n    id\n    handle\n    structured_scope_versions(archived: false) {\n      max_updated_at\n      __typename\n    }\n    in_scope_assets: structured_scopes(\n      archived: false\n      eligible_for_submission: true\n    ) {\n      edges {\n        node {\n          id\n          asset_type\n          asset_identifier\n          instruction\n          max_severity\n          eligible_for_bounty\n          labels(first: 100) {\n            edges {\n              node {\n                id\n                name\n                __typename\n              }\n              __typename\n            }\n            __typename\n          }\n          __typename\n        }\n        __typename\n      }\n      __typename\n    }\n    out_scope_assets: structured_scopes(\n      archived: false\n      eligible_for_submission: false\n    ) {\n      edges {\n        node {\n          id\n          asset_type\n          asset_identifier\n          instruction\n          __typename\n        }\n        __typename\n      }\n      __typename\n    }\n    __typename\n  }\n}\n"}`, handle))

	res_data, err := h.ProgramGraphql(data)
	if err != nil {
		fmt.Println("hackerone Scope 获取失败", err)
		return
	}

	in_result := gjson.GetBytes(res_data, "data.team.in_scope_assets.edges.#.node")
	out_result := gjson.GetBytes(res_data, "data.team.out_scope_assets.edges.#.node")

	err = json.Unmarshal([]byte(in_result.Raw), &in_scopes)
	if err != nil {
		fmt.Println("hackerone Scope 解析失败", err)
		return
	}

	err = json.Unmarshal([]byte(out_result.Raw), &out_scopes)

	if err != nil {
		fmt.Println(err)
	}

	return

}
