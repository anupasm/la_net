package main

// SimpleToken interface is modeled after Ethereum's ERC20 standard.
//
// See: https://github.com/ethereum/EIPs/blob/master/EIPS/eip-20.md
type TokenI interface {

	// OwnerOf returns the owner of the token.
	OwnerOf(tokenId string) (string, error)

	// Mint returns the token balance of the specified owner.
	Mint(tokenId string, tokenURI string) error

	// BalanceOf returns the token balance of the specified owner.
	BalanceOf(owner string) (int, error)

	// Approve will allow 'spender' to transfer 'amount' tokens from
	// the invoker (owner) by calling TransferFrom. Calling Approve
	// multiple times overwrites the previous approved amount.
	Approve(operator string, tokenId string) (bool, error)

	// TransferFrom allows the invoker to transfer up to 'amount'
	// tokens from the owner's ('from') account to the receiver's
	// ('to') account. The invoker is allowed to call TransferFrom
	// multiple times as long as there are sufficient funds.
	TransferFrom(from string, to string, tokenId string) (bool, error)

	// Transfer allows the invoker to transfer his token to receiver's
	// ('to') account.
	Transfer(to string, tokenId string) (bool, error)

	// Allowance returns the amount of tokens approved by an owner for
	// spending by a given 'spender'.
	GetApproved(tokenId string) (string, error)
}
