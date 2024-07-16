package intigriti

type Bounty struct {
	Value    float64 `json:"value"`
	Currency string  `json:"currency"`
}

type IntigritiScope struct {
	Type        string `json:"type"`
	Endpoint    string `json:"endpoint"`
	Impact      string `json:"impact"`
	Description string `json:"description"`
}

type IntigritiTarget struct {
	InScope    []IntigritiScope `json:"in_scope"`
	OutOfScope []IntigritiScope `json:"out_of_scope"`
}

type Intigriti struct {
	Name                 string          `json:"name"`
	CompanyHandle        string          `json:"companyHandle"`
	Handle               string          `json:"handle"`
	Url                  string          `json:"url"`
	Status               string          `json:"status"`
	ConfidentialityLevel int64           `json:"confidentialityLevel"`
	MinBounty            Bounty          `json:"minBounty"`
	MaxBounty            Bounty          `json:"maxBounty"`
	Targets              IntigritiTarget `json:"targets"`
}
