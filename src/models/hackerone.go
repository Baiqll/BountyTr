package models

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
