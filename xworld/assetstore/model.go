package main

type PayPage struct {
	Merchant string
	Amount   string
	Image    string
}

type QR struct {
	Merchant string
	Amount   string
	SID      string
	Secret   string
}

type Agreement struct {
	Owner         string `json:"owner"`
	Counterparty  string `json:"counterparty"`
	Image         string `json:"image"`
	TokenId       string `json:"token"`
	TokenContract string `json:"tokenContract"`
	Expiry        int64  `json:"expiry"`
}



type Transfer struct {
	From    string `json:"from"`
	To      string `json:"to"`
	TokenId string `json:"tokenId"`
}