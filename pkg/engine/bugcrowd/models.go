package bugcrowd

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
