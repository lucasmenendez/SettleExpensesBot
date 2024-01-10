package bot

type transactionSnapshot struct {
	Amount       float64  `json:"amount"`
	Payer        string   `json:"payer"`
	Participants []string `json:"participants"`
}

type sessionSnapshot struct {
	ID           int64                 `json:"id"`
	Transactions []transactionSnapshot `json:"transactions"`
	Expire       int64                 `json:"expire"`
}

type snapshotData struct {
	Sessions     []sessionSnapshot `json:"sessions"`
	AllowedUsers map[int64]string  `json:"allowed_users"`
}
