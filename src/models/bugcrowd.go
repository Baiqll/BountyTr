package models

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
