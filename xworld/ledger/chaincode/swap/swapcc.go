package main

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"github.com/dileban/atomic-swaps/fabric/lib/security"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

// SwapChaincode is ...
type SwapChaincode struct {
	swap SwapI
}

// CallerProps is a container for meta data from the remote client as
// well as the peer. This includes the arguments and identity of the
// client as well as callback pointers to the peer.
type CallerProps struct {
	args []string
	cert *x509.Certificate
	stub shim.ChaincodeStubInterface
}

// For use within handlers and the token implementation.
var caller *CallerProps

// Init is called during chaincode instantiation. No special
// initialization required.
func (ccs *SwapChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Invoke is called to update or query the state of the ledger. The
// arguments passed to Invoke by the remote client include:
//
//	0: The name of the function to Invoke. See 'HTLC'
//	   interface for list of function names that can be supplied.
//	1..N: A list of arguments for the function defined in the
//	   'HTLC' interface.
func (ccs *SwapChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	f, params := stub.GetFunctionAndParameters()
	ccs.swap = &Swap{}

	// Initialize caller props for use in handlers
	cert, _ := cid.GetX509Certificate(stub)
	caller = &CallerProps{args: params, cert: cert, stub: stub}

	// Dispatch to appropriate handler based on supplied func name
	// TODO: Handle potential panics
	v := reflect.ValueOf(ccs).MethodByName(f + "Handler").Call([]reflect.Value{})
	return v[0].Interface().(pb.Response)
}

// LockHandler creates a new swap agreement between the invoker
// (owner) and the counterparty. If the lock was successful, the
// handler raises the 'Locked' event and returns the ID of the new
// agreement.
func (ccs *SwapChaincode) LockHandler() pb.Response {
	// TODO: validate args
	counterparty := caller.args[0]
	image := caller.args[1]
	tokenId := caller.args[2]
	tokenContract := caller.args[3]
	lockTime := stringToInt64(caller.args[4])

	// Lock tokens by creating new swap agreement with counterparty
	agreementID, err := ccs.swap.Lock(counterparty, image, tokenId, tokenContract, lockTime)
	if err != nil {
		return shim.Error(fmt.Sprintf("Error creating agreement for counterparty %s: %v", counterparty, err))
	}
	owner := getInvokerAddress()
	expiry := getExpiryTime(lockTime)
	_ = caller.stub.SetEvent("Locked", newLockedEvent(agreementID, owner, counterparty, image, tokenId, expiry))
	return shim.Success([]byte(agreementID))
}

// UnlockHandler releases tokens locked by the invoker (owner) under a
// given agreement id if the lock time has elapsed. If the unlock was
// successful the handler raises the 'Unlocked' event and returns an
// empty payload.
func (ccs *SwapChaincode) UnlockHandler() pb.Response {
	// TODO: Validate args
	agreementID := caller.args[0]

	// Unlock owner's tokens if lock time has elapsed
	if err := ccs.swap.Unlock(agreementID); err != nil {
		return shim.Error(fmt.Sprintf("Failed to unlock tokens for agreement %s: %s", agreementID, err))
	}
	_ = caller.stub.SetEvent("Unlocked", newUnlockedEvent(agreementID))
	return shim.Success(nil)
}

// ClaimHandler allows the counterparty to claim tokens locked by the
// creator of an agreement given the provided secret is correct. If
// the claim was successful the handler raises the 'Claimed' event and
// returns an empty payload.
func (ccs *SwapChaincode) ClaimHandler() pb.Response {
	// TODO: Validate args
	agreementID := caller.args[0]
	secret := caller.args[1]

	// Claim locked tokens using secret
	if err := ccs.swap.Claim(agreementID, secret); err != nil {
		return shim.Error(fmt.Sprintf("Failed to claim tokens form agreement %s: %s", agreementID, err))
	}
	_ = caller.stub.SetEvent("Claimed", newClaimedEvent(agreementID))
	return shim.Success(nil)
}

// EchoHandler returns the address
func (tcc *SwapChaincode) EchoHandler() pb.Response {
	chaincodeAddress := getChaincodeAddress()
	return shim.Success([]byte(chaincodeAddress))
}

// newLockedEvent returns a byte array representing a chaincode
// event when tokens have been unlocked under an agreement.
func newLockedEvent(agreementID string, owner string, counterparty string,
	image string, tokenId string, expiry int64) []byte {
	t := Locked{AgreementID: agreementID, Owner: owner, CounterParty: counterparty,
		Image: image, TokenId: tokenId, Expiry: expiry}
	b, _ := json.Marshal(t)
	return b
}

// newUnlockedEvent returns a byte array representing a chaincode
// event when tokens from an agreement have been unlocked.
func newUnlockedEvent(agreementID string) []byte {
	t := Unlocked{AgreementID: agreementID}
	b, _ := json.Marshal(t)
	return b
}

// newClaimedEvent returns a byte array representing a chaincode
// event when tokens from an agreement have been claimed.
func newClaimedEvent(agreementID string) []byte {
	t := Claimed{AgreementID: agreementID}
	b, _ := json.Marshal(t)
	return b
}

// getInvokerAddress returns a hex-based address representing the
// invoker's public key.
func getInvokerAddress() string {
	cert := security.NewX509Certificate(caller.cert)
	return cert.GetAddress()
}

// getChaincodeAddress returns an address that represents the current
// chaincode. The format of this address is currently based on the
// chaincode ID.
func getChaincodeAddress() string {
	chaincodeID, _ := getChaincodeID()
	return "cc:" + chaincodeID
}

// getChaincodeID returns the name (hash) of the chaincode specified
// in the signed proposal request.
func getChaincodeID() (string, error) {
	var signedProposal *pb.SignedProposal
	var err error
	if signedProposal, err = caller.stub.GetSignedProposal(); err != nil {
		return "", err
	}

	proposal := &pb.Proposal{}
	err = proto.Unmarshal(signedProposal.ProposalBytes, proposal)
	if err != nil {
		return "", fmt.Errorf("could not unmarshal proposal: %v", err)
	}

	proposalPayload := &pb.ChaincodeProposalPayload{}
	err = proto.Unmarshal(proposal.Payload, proposalPayload)
	if err != nil {
		return "", fmt.Errorf("could not unmarshal chaincode proposal payload: %v", err)
	}

	cis := &pb.ChaincodeInvocationSpec{}
	err = proto.Unmarshal(proposalPayload.Input, cis)
	if err != nil {
		return "", fmt.Errorf("could not unmarshal chaincode invocation spec: %v", err)
	}

	if cis.ChaincodeSpec == nil {
		return "", fmt.Errorf("chaincode spec is nil")
	}

	if cis.ChaincodeSpec.ChaincodeId == nil {
		return "", fmt.Errorf("chaincode id is nil")
	}

	return cis.ChaincodeSpec.ChaincodeId.Name, nil
}

// getExpiryTime returns the (wall clock) time after which an
// agreement can be unlocked by the initiator. The expiry time is
// calculated using the client's transaction timestamp. This is
// deterministic and safe (as a counterparty can always inspect the
// expiry before proceeding with a swap).
func getExpiryTime(lockTime int64) int64 {
	t, _ := caller.stub.GetTxTimestamp()
	return t.GetSeconds() + lockTime
}

// // uint64ToBytes converts a string to an unsigned integer.
// func stringToUint64(s string) uint64 {
// 	i, _ := strconv.ParseUint(s, 10, 64)
// 	return i
// }

// int64ToBytes converts a string to an integer.
func stringToInt64(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

func main() {
	ccs := new(SwapChaincode)
	if err := shim.Start(ccs); err != nil {
		fmt.Printf("Error starting CrossChainSwap: %s", err)
	}
}
