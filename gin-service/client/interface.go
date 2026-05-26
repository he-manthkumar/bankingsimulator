package client

type BankingClient interface {
	SettleTransfer(fromAccount, toAccount string, amount float64, transferMode, tpin string) (*SettleTransferResponse, error)
}