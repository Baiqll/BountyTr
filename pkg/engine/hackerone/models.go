package hackerone

type APIResponse struct {
	Data  []*APIProgram `json:"data"`
	Links struct {
		Next string `json:"next"`
	} `json:"links"`
}

type APIProgram struct {
	ID         string `json:"id"`
	Attributes ProgramWithScope `json:"attributes"`
}


type APIScope struct {
	Data []struct {
		Attributes ScopeTarget`json:"attributes"`
	} `json:"data"`
}


type ProgramWithScope struct {
	Handle                     string   `json:"handle"`
	Name                       string   `json:"name"`
	URL                        string   `json:"url"`
	State                      string   `json:"state"`
	OffersBounties             bool     `json:"offers_bounties"`
	OffersSwag                 bool     `json:"offers_swag"`
	SubmissionState            string   `json:"submission_state"`
	ManagedProgram             bool     `json:"managed_program"`
	AllowsBountySplitting      bool     `json:"allows_bounty_splitting"`
	ResponseEfficiencyPercent  int      `json:"response_efficiency_percentage"`
	AvgTimeToBountyAwarded     *float64 `json:"average_time_to_bounty_awarded"`
	AvgTimeToFirstResponse     *float64 `json:"average_time_to_first_program_response"`
	AvgTimeToReportResolved    *float64 `json:"average_time_to_report_resolved"`
	Targets struct {
		InScope []ScopeTarget `json:"in_scope"`
	} `json:"targets"`
}

// ScopeTarget scope 目标详情
type ScopeTarget struct {
	AssetType                  string  `json:"asset_type"`
	AssetIdentifier            string  `json:"asset_identifier"`
	EligibleForBounty          bool    `json:"eligible_for_bounty"`
	EligibleForSubmission      bool    `json:"eligible_for_submission"`
	Instruction                string  `json:"instruction,omitempty"`
	MaxSeverity                string  `json:"max_severity,omitempty"`
	AvailabilityRequirement    *string `json:"availability_requirement,omitempty"`
	ConfidentialityRequirement *string `json:"confidentiality_requirement,omitempty"`
	IntegrityRequirement       *string `json:"integrity_requirement,omitempty"`
}

