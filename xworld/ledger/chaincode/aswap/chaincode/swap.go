package chaincode

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"schnorr"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

const (
	channelName       = "moochan"
	nftchaincodeName  = "nftsc"
	swapchaincodeName = "swapsc"
)

// Swap implements the HTLC interface.
//
// See lib/asset/htlc/HTLC
type SwapContract struct {
	contractapi.Contract
}

// Agreement represents a swap contract between an owner of tokens and
// a counterparty. The construct of an agreement captures the
// underlying token contract, the amount of tokens to be swapped and
// the image of a secret required to claim tokens. An agreement
// expires after a pre-agreed period of time.
type Agreement struct {
	// The address of the token owner and creator of an agreement.
	Owner string `json:"owner"`

	// The address of the counterparty in the agreement who is allowed
	// to claim tokens before the expiry.
	Counterparty string `json:"counterparty"`

	// The image of a secret required to claim tokens.
	Image string `json:"image"`

	// The amount of tokens to be swapped in the agreement.
	TokenId string `json:"token"`

	// The name of the token contract representing the tokens to be
	// swaped in the agreement.
	TokenContract string `json:"tokenContract"`

	// The time (wall clock) after which the agreement is considered to
	// have expired and tokens can be unlocked by the owner.
	Expiry int64 `json:"expiry"`
}

// Lock creates a new swap agreement between the token owner and a
// counterparty. The agreement includes the image of a known secret,
// the amount of tokens to swap, the name of the underlying token
// contract to invoke and an agreed upon lock time during which the
// invoker is unable to withdraw her tokens.
//
// The token owner must ensure an allowance to the amount specified in
// the agreement is made to the current contract's address. Invoking
// this function results in a transfer of funds from the owner's
// address to the current contract's address. The transfer is executed
// on the target contract by way of invoking the contract
// chaincode. The function returns the agreement ID.
func (ccs *SwapContract) Lock(ctx contractapi.TransactionContextInterface, counterparty string, image string, tokenid string, tokenContract string, lockTime int64) (string, error) {
	var agreement *Agreement
	var err error
	agreementID := newAgreementID(ctx)
	// Verify if agreement ID is unique
	if agreement, err = ccs.getAgreement(ctx, agreementID); err != nil {
		return "", err
	}
	if agreement != nil {
		return "", fmt.Errorf("Agreement %s already exists", agreementID)
	}
	// Create new agreement and write to ledger
	invoker, err := getIdentity(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get identity : %v", err)
	}

	t, _ := ctx.GetStub().GetTxTimestamp()
	expiry := t.GetSeconds() + lockTime
	agreement = &Agreement{
		Owner:         invoker,
		Counterparty:  counterparty,
		Image:         image,
		TokenId:       tokenid,
		TokenContract: tokenContract,
		Expiry:        expiry}
	if err = ccs.putAgreement(ctx, agreementID, agreement); err != nil {
		return "", err
	}
	// TODO: Invoke token contract to check if the contract has
	// implemented support for 'chaincode addresses'.

	// Invoke token contract to 'lock' tokens to custom (chaincode) address.
	args := argArray("TransferFrom", invoker, swapchaincodeName, tokenid)
	result := ctx.GetStub().InvokeChaincode(tokenContract, args, "")

	if result.Status != shim.OK {
		return "", fmt.Errorf("error transferring tokens in contract %s: %s", invoker, result.Message)
	}
	agreementEventBytes, err := hex.DecodeString(agreementID)
	if err != nil {
		return "", fmt.Errorf("failed to marshal transferEventBytes: %v", err)
	}

	err = ctx.GetStub().SetEvent("Agreement", agreementEventBytes)
	if err != nil {
		return "", fmt.Errorf("failed to SetEvent transferEventBytes %s: %v", agreementEventBytes, err)
	}
	return agreementID, nil
}

func getIdentity(ctx contractapi.TransactionContextInterface) (string, error) {
	identity, err := ctx.GetClientIdentity().GetID()
	return identity, err
	// creator, err := ctx.GetStub().GetCreator()
	// address := sha256.Sum256(creator)
	// if err != nil {
	// 	return "", err
	// }
	// return base64.StdEncoding.EncodeToString(address[:]), nil
}

// Unlock releases tokens locked by the invoker (owner) under a given
// agreement id. Tokens can only be released once the lock time has
// elapsed.
//
// Invoking this function results in a transfer of funds from the
// current contract's address to the owner's address. The transfer is
// executed on the target contract by way of invoking the contract
// chaincode.
func (ccs *SwapContract) Unlock(ctx contractapi.TransactionContextInterface, agreementID string) error {
	var agreement *Agreement
	var err error
	if agreement, err = ccs.getAgreement(ctx, agreementID); err != nil {
		return err
	}
	invoker, err := getIdentity(ctx)
	if err != nil {
		return fmt.Errorf("failed to get identity : %v", err)
	}
	if invoker != agreement.Owner {
		return fmt.Errorf("attempting to unlock tokens belonging to %s", agreement.Owner)
	}
	if agreement.Expiry > time.Now().Unix() {
		return fmt.Errorf("Agreement is set to expire on %s", time.Unix(agreement.Expiry, 0).Format(time.RFC850))
	}
	// Invoke token contract to 'unlock' tokens from custom (chaincode) address.
	args := argArray("Transfer", agreement.Owner, agreement.TokenId)
	result := ctx.GetStub().InvokeChaincode(agreement.TokenContract, args, "")
	if result.Status != shim.OK {
		return fmt.Errorf("error transferring tokens in contract %s: %s", agreement.TokenContract, result.Message)
	}
	return nil
}

// Claim allows the counterparty to claim tokens from the agreement
// setup by the creator. The counterparty must provide the correct
// agreement id and secret to claim her tokens.
//
// Invoking this function results in a transfer of funds from the
// current contract's address to the counterparty's address. The
// transfer is executed on the target contract by way of invoking the
// contract chaincode.
func (ccs *SwapContract) Claim(ctx contractapi.TransactionContextInterface, agreementID string, secret string) (string, error) {
	var agreement *Agreement
	var err error
	if agreement, err = ccs.getAgreement(ctx, agreementID); err != nil {
		return "", err
	}
	invoker, err := getIdentity(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get identity : %v", err)
	}

	if invoker != agreement.Counterparty {
		return "", fmt.Errorf("attempting to claim tokens belonging to %s by %s", agreement.Counterparty, invoker)
	}

	if agreement.Expiry < time.Now().Unix() {
		return "", fmt.Errorf("Agreement expired on %s", time.Unix(agreement.Expiry, 0).Format(time.RFC850))
	}
	im, err := imageOf(secret)
	if err != nil {
		return "", err
	}
	if im != agreement.Image {
		return "", fmt.Errorf("SHA256 of secret '%s' does not match image '%s' received '%s' for agreement '%s'", secret, agreement.Image, im, agreementID)
	}
	// Invoke token contract to 'unlock' tokens from custom (chaincode) address.

	args := argArray("TransferFrom", swapchaincodeName, agreement.Counterparty, agreement.TokenId)
	result := ctx.GetStub().InvokeChaincode(agreement.TokenContract, args, "")
	if result.Status != shim.OK {
		return "", fmt.Errorf("error transferring tokens in contract %s: %s", agreement.TokenContract, result.String())
	}
	// Emit the ApprovalForAll event
	data, _ := hex.DecodeString(secret)
	err = ctx.GetStub().SetEvent("Claim", data)
	if err != nil {
		return "", fmt.Errorf("failed to SetEvent Claim: %v", err)
	}
	return "ok", nil
}

func (ccs *SwapContract) Details(ctx contractapi.TransactionContextInterface, agreementId string) (string, error) {

	agreement, err := ccs.getAgreement(ctx, agreementId)

	if err != nil {
		return "", err
	}

	return agreement.TokenContract, nil
}

// getAgreement returns the agreement with the specified ID from the ledger.
func (ccs *SwapContract) getAgreement(ctx contractapi.TransactionContextInterface, agreementID string) (*Agreement, error) {
	var b []byte
	var err error
	if b, err = ctx.GetStub().GetState(agreementID); err != nil {
		return nil, err
	}
	var agreement Agreement
	if b == nil {
		return nil, nil
	}
	if err = json.Unmarshal(b, &agreement); err != nil {
		return nil, err
	}
	return &agreement, nil
}

// putAgreement writes the given agreement to the ledger.
func (ccs *SwapContract) putAgreement(ctx contractapi.TransactionContextInterface, agreementID string, agreement *Agreement) error {
	b, err := json.Marshal(&agreement)
	if err != nil {
		return err
	}
	if err = ctx.GetStub().PutState(agreementID, b); err != nil {
		return err
	}
	return nil
}

// newAgreementID creates a unique agreement ID.
func newAgreementID(ctx contractapi.TransactionContextInterface) string {
	// The transaction ID is unique per transaction, per client.
	// This will serve as a good agreement ID.
	return ctx.GetStub().GetTxID()
}

// imageOf returns the SHA256 hex representation of a given string.
func imageOf(secret string) (string, error) {
	data, err := hex.DecodeString(secret)
	if err != nil {
		return "", err
	}
	r := schnorr.NewScalar([32]byte(data))

	R := r.G().ToBytes()
	return hex.EncodeToString(R[:]), nil
}

// argArray returns a slice over byte array, each element representing a
// byte representation of a string.
func argArray(s ...string) [][]byte {
	args := make([][]byte, len(s))
	for i, v := range s {
		args[i] = []byte(v)
	}
	return args
}

// // getChaincodeAddress returns an address that represents the current
// // chaincode. The format of this address is currently based on the
// // chaincode ID.
// // getChaincodeID returns the name (hash) of the chaincode specified
// // in the signed proposal request.
// func getChaincodeAddress(ctx contractapi.TransactionContextInterface) (string, error) {

// 	var signedProposal *prop.SignedProposal
// 	var err error
// 	if signedProposal, err = ctx.GetStub().GetSignedProposal(); err != nil {
// 		return "", err
// 	}

// 	proposal := &prop.Proposal{}
// 	err = proto.Unmarshal(signedProposal.ProposalBytes, proposal)
// 	if err != nil {
// 		return "", fmt.Errorf("could not unmarshal proposal: %v", err)
// 	}

// 	proposalPayload := &prop.ChaincodeProposalPayload{}
// 	err = proto.Unmarshal(proposal.Payload, proposalPayload)
// 	if err != nil {
// 		return "", fmt.Errorf("could not unmarshal chaincode proposal payload: %v", err)
// 	}

// 	cis := &prop.ChaincodeInvocationSpec{}
// 	err = proto.Unmarshal(proposalPayload.Input, cis)
// 	if err != nil {
// 		return "", fmt.Errorf("could not unmarshal chaincode invocation spec: %v", err)
// 	}

// 	if cis.ChaincodeSpec == nil {
// 		return "", fmt.Errorf("chaincode spec is nil")
// 	}

// 	if cis.ChaincodeSpec.ChaincodeId == nil {
// 		return "", fmt.Errorf("chaincode id is nil")
// 	}
// 	return "cc:" + cis.ChaincodeSpec.ChaincodeId.Name, nil
// }
