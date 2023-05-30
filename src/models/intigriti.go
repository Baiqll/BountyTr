package models

type Bounty struct {
	Value    float64 `json:"value"`
	Currency string  `json:"currency"`
}

type IntigritiScope struct {
	Type        string `json:"type"`
	Endpoint    string `json:"endpoint"`
	Description string `json:"description"`
}

type IntigritiTarget struct {
	InScope    []IntigritiScope `json:"in_scope"`
	OutOfScope []IntigritiScope `json:"out_of_scope"`
}

type Intigriti struct {
	Name                 string          `json:"name"`
	CompanyHandle        string          `json:"company_handle"`
	Handle               string          `json:"handle"`
	Url                  string          `json:"url"`
	Status               string          `json:"status"`
	ConfidentialityLevel string          `json:"confidentiality_level"`
	MinBounty            Bounty          `json:"min_bounty"`
	MaxBounty            Bounty          `json:"max_bounty"`
	Targets              IntigritiTarget `json:"targets"`
}
