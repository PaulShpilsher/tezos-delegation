package api

type DelegationDto struct {
	Timestamp string `json:"timestamp"`
	Amount    int64  `json:"amount"`
	Delegator string `json:"delegator"`
	Level     string `json:"level"`
}

type GetDelegationsResponse struct {
	Data []DelegationDto `json:"data"`
}
