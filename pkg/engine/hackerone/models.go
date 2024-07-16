package hackerone


type Programs struct {
	Data []*ProgramsData `json:"data"`
	Links        *NextPrograms   `json:"links"`
}
type ProgramsData struct {
	Attributes *ProgramsAttributes `json:"attributes"`
}

type ProgramsAttributes struct {
	Handle       string `json:"handle"`
	State        string `json:"state"`
	OffersBounty bool   `json:"offers_bounties"`
}

type NextPrograms struct {
	Next string `json:"next"`
}


type Scope struct {
	Data []*ScopeData `json:"data"`
}

type ScopeData struct {
	Attributes *ScopeAttributes `json:"attributes"`
}

type ScopeAttributes struct {
	AssetType         string `json:"asset_type"`
	Identifier        string `json:"asset_identifier"`
	EligibleForBounty bool   `json:"eligible_for_bounty"` // 是否在范围内/是否有赏金
}


type ProgramsScope struct{

	Handle       string `json:"handle"`
	State        string `json:"state"`
	OffersBounty bool   `json:"offers_bounties"`
	Scope []*ScopeData `json:"scope"`
}