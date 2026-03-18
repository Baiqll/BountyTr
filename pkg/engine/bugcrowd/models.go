package bugcrowd

type APIResponse struct {
	Engagements []APIProgram `json:"engagements"`
	Links       struct {
		Next string `json:"next"`
	} `json:"links"`
}

type APIProgram struct {
	Name       string `json:"name"`
	BriefURL   string `json:"briefUrl"`
	IsPrivate  bool   `json:"isPrivate"`
}

type APITarget struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	Category string `json:"category"`
	URI      string `json:"uri"`
}

type ProgramWithScope struct {
	Handle string `json:"handle"`
	Name   string `json:"name"`
	URL    string `json:"url"`
	Targets struct {
		InScope []APITarget `json:"in_scope"`
	} `json:"targets"`
}

type BugcrowdScope struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	IpAddress   string `json:"ipAddress"`
	Url         string `json:"uri"`
}

type BugcrowdTarget struct {
	InScope    []BugcrowdScope `json:"in_scope"`
	OutOfScope []BugcrowdScope `json:"out_of_scope"`
}

type Bugcrowd struct {
	Name          string         `json:"name"`
	Url           string         `json:"url"`
	Participation string         `json:"participation"`
	ProgramUrl    string         `json:"program_url"`
	InvitedStatus string         `json:"invited_status"`
	MinRewards    int64          `json:"min_rewards"`
	MaxRewards    int64          `json:"max_rewards"`
	Targets       BugcrowdTarget `json:"targets"`
}
