package client

type BankingClient interface {
	DebitAccount(accountID string, amount float64, transferRef, tpin string) (*DebitResult, error)
	CreditAccount(accountID string, amount float64, transferRef string) (*CreditResult, error)
}
