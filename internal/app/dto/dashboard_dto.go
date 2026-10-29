package dto

// AdminSummaryResponse — ringkasan jumlah data platform untuk dashboard admin.
type AdminSummaryResponse struct {
	Users struct {
		Active    int `json:"active"`
		Inactive  int `json:"inactive"`
		Suspended int `json:"suspended"`
		Admins    int `json:"admins"`
	} `json:"users"`
	Companies struct {
		Active    int `json:"active"`
		Suspended int `json:"suspended"`
		Closed    int `json:"closed"`
	} `json:"companies"`
	Solutions struct {
		Active   int `json:"active"`
		Inactive int `json:"inactive"`
	} `json:"solutions"`
	ActiveClients int `json:"active_clients"`
}
