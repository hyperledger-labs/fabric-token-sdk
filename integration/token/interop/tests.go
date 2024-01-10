/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interop

import (
	"crypto"
	"encoding/base64"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop/views"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	pledge2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

func TestHTLCSingleNetwork(network *integration.Infrastructure) {
	// give some time to the nodes to get the public parameters
	time.Sleep(10 * time.Second)

	defaultTMSID := token.TMSID{}
	RegisterAuditor(network)

	IssueCash(network, "", "USD", 110, "alice")
	CheckBalanceWithLockedAndHolding(network, "alice", "", "USD", 110, 0, 0, -1)
	IssueCash(network, "", "USD", 10, "alice")
	CheckBalanceWithLockedAndHolding(network, "alice", "", "USD", 120, 0, 0, -1)

	h := ListIssuerHistory(network, "", "USD")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(120))).To(BeEquivalentTo(0))
	Expect(h.ByType("USD").Count()).To(BeEquivalentTo(h.Count()))

	h = ListIssuerHistory(network, "", "EUR")
	Expect(h.Count()).To(BeEquivalentTo(0))

	IssueCash(network, "", "EUR", 10, "bob")
	CheckBalanceWithLockedAndHolding(network, "bob", "", "EUR", 10, 0, 0, -1)
	IssueCash(network, "", "EUR", 10, "bob")
	CheckBalanceWithLockedAndHolding(network, "bob", "", "EUR", 20, 0, 0, -1)
	IssueCash(network, "", "EUR", 10, "bob")
	CheckBalanceWithLockedAndHolding(network, "bob", "", "EUR", 30, 0, 0, -1)

	h = ListIssuerHistory(network, "", "USD")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(120))).To(BeEquivalentTo(0))
	Expect(h.ByType("USD").Count()).To(BeEquivalentTo(h.Count()))

	h = ListIssuerHistory(network, "", "EUR")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(30))).To(BeEquivalentTo(0))
	Expect(h.ByType("EUR").Count()).To(BeEquivalentTo(h.Count()))

	CheckBalanceWithLockedAndHolding(network, "alice", "", "USD", 120, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "alice", "", "EUR", 0, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "bob", "", "EUR", 30, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "bob", "", "USD", 0, 0, 0, -1)

	Restart(network, "issuer", "auditor", "alice", "bob")
	RegisterAuditor(network)

	// htlc (lock, failing claim, reclaim)
	_, preImage, _ := HTLCLock(network, token.TMSID{}, "alice", "", "USD", 10, "bob", 10*time.Second, nil, crypto.SHA512)
	CheckBalanceWithLockedAndHolding(network, "alice", "", "USD", 110, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "alice", "", "EUR", 0, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "bob", "", "EUR", 30, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "bob", "", "USD", 0, 10, 0, -1)
	time.Sleep(15 * time.Second)
	CheckBalanceWithLockedAndHolding(network, "alice", "", "USD", 110, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "alice", "", "EUR", 0, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "bob", "", "EUR", 30, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "bob", "", "USD", 0, 0, 10, -1)
	HTLCClaim(network, defaultTMSID, "bob", "", preImage, "deadline elapsed")
	CheckBalanceWithLockedAndHolding(network, "alice", "", "USD", 110, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "alice", "", "EUR", 0, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "bob", "", "EUR", 30, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "bob", "", "USD", 0, 0, 10, -1)

	HTLCReclaimAll(network, "alice", "")
	CheckBalanceWithLockedAndHolding(network, "alice", "", "USD", 120, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "alice", "", "EUR", 0, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "bob", "", "EUR", 30, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "bob", "", "USD", 0, 0, 0, -1)

	// htlc (lock, claim)
	_, preImage, _ = HTLCLock(network, defaultTMSID, "alice", "", "USD", 20, "bob", 1*time.Hour, nil, crypto.SHA3_256)
	CheckBalanceWithLockedAndHolding(network, "alice", "", "USD", 100, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "alice", "", "EUR", 0, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "bob", "", "EUR", 30, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "bob", "", "USD", 0, 20, 0, -1)

	failedClaimTXID := HTLCClaim(network, defaultTMSID, "bob", "", preImage)
	CheckBalanceWithLockedAndHolding(network, "alice", "", "USD", 100, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "alice", "", "EUR", 0, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "bob", "", "EUR", 30, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "bob", "", "USD", 20, 0, 0, -1)

	// payment limit reached
	HTLCLock(network, defaultTMSID, "alice", "", "USD", uint64(views.Limit+10), "bob", 1*time.Hour, nil, crypto.SHA3_256, "payment limit reached")
	CheckBalanceWithLockedAndHolding(network, "alice", "", "USD", 100, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "alice", "", "EUR", 0, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "bob", "", "EUR", 30, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "bob", "", "USD", 20, 0, 0, -1)

	CheckPublicParams(network, defaultTMSID, "issuer", "auditor", "alice", "bob")
	CheckOwnerDB(network, defaultTMSID, nil, "issuer", "auditor", "alice", "bob")
	CheckAuditorDB(network, token.TMSID{}, "auditor", "", func(errs []string) error {
		if len(errs) != 2 {
			return errors.Errorf("expected 2 errors, got [%d][%v][%s]", len(errs), errs, failedClaimTXID)
		}
		for _, err := range errs {
			if strings.Contains(err, failedClaimTXID) {
				return errors.Errorf("[%s] does not contain [%s]", err, failedClaimTXID)
			}
		}
		return nil
	})

	// lock two times with the same hash, the second lock should fail
	_, _, hash := HTLCLock(network, defaultTMSID, "alice", "", "USD", 1, "bob", 1*time.Hour, nil, crypto.SHA3_256)
	failedLockTXID, _, _ := HTLCLock(network, defaultTMSID, "alice", "", "USD", 1, "bob", 1*time.Hour, hash, crypto.SHA3_256,
		fmt.Sprintf(
			"entry with transfer metadata key [%s] is already occupied by [%s]",
			htlc.LockKey(hash),
			base64.StdEncoding.EncodeToString(htlc.LockValue(hash)),
		),
	)
	HTLCLock(network, defaultTMSID, "alice", "", "USD", 1, "bob", 1*time.Hour, nil, crypto.SHA3_256)

	CheckPublicParams(network, defaultTMSID, "issuer", "auditor", "alice", "bob")
	CheckOwnerDB(network, defaultTMSID, nil, "issuer", "auditor", "alice", "bob")
	CheckAuditorDB(network, token.TMSID{}, "auditor", "", func(errs []string) error {
		fmt.Printf("Got errors [%v]", errs)
		if len(errs) != 6 {
			return errors.Errorf("expected 6 errors, got [%d]", len(errs))
		}
		for _, err := range errs[:2] {
			if strings.Contains(err, failedClaimTXID) {
				return errors.Errorf("[%s] does not contain [%s]", err, failedClaimTXID)
			}
		}
		firstError := fmt.Sprintf("transaction record [%s] is unknown for vault but not for the db [%s]", failedLockTXID, auditor.Pending)
		if errs[2] != firstError {
			return errors.Errorf("expected first error to be [%s], got [%s]", firstError, errs[0])
		}
		for _, err := range errs[2:] {
			if !strings.Contains(err, failedLockTXID) {
				return errors.Errorf("[%s] does not contain [%s]", err, failedLockTXID)
			}
		}
		return nil
	})
	PruneInvalidUnspentTokens(network, defaultTMSID, "issuer", "auditor", "alice", "bob")
	for _, name := range []string{"alice", "bob"} {
		IDs := ListVaultUnspentTokens(network, defaultTMSID, name)
		CheckIfExistsInVault(network, defaultTMSID, "auditor", IDs)
	}
}

func TestHTLCTwoNetworks(network *integration.Infrastructure) {
	// give some time to the nodes to get the public parameters
	time.Sleep(10 * time.Second)

	alpha := token.TMSID{Network: "alpha"}
	beta := token.TMSID{Network: "beta"}

	RegisterAuditor(network, token.WithTMSID(alpha))
	RegisterAuditor(network, token.WithTMSID(beta))

	IssueCashWithTMS(network, alpha, "issuer", "", "EUR", 30, "alice")
	CheckBalanceAndHolding(network, "alice", "", "EUR", 30, token.WithTMSID(alpha))

	IssueCashWithTMS(network, beta, "issuer", "", "USD", 30, "bob")
	CheckBalanceAndHolding(network, "bob", "", "USD", 30, token.WithTMSID(beta))

	_, preImage, hash := HTLCLock(network, alpha, "alice", "", "EUR", 10, "bob", 1*time.Hour, nil, 0)
	HTLCLock(network, beta, "bob", "", "USD", 10, "alice", 1*time.Hour, hash, 0)
	HTLCClaim(network, beta, "alice", "", preImage)
	HTLCClaim(network, alpha, "bob", "", preImage)

	CheckBalanceWithLockedAndHolding(network, "alice", "", "EUR", 20, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, "bob", "", "EUR", 10, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, "alice", "", "USD", 10, 0, 0, -1, token.WithTMSID(beta))
	CheckBalanceWithLockedAndHolding(network, "bob", "", "USD", 20, 0, 0, -1, token.WithTMSID(beta))

	// Try to claim again and get an error

	HTLCClaim(network, beta, "alice", "", preImage, "expected only one htlc script to match")
	HTLCClaim(network, alpha, "bob", "", preImage, "expected only one htlc script to match")

	CheckBalanceWithLockedAndHolding(network, "alice", "", "EUR", 20, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, "bob", "", "EUR", 10, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, "alice", "", "USD", 10, 0, 0, -1, token.WithTMSID(beta))
	CheckBalanceWithLockedAndHolding(network, "bob", "", "USD", 20, 0, 0, -1, token.WithTMSID(beta))

	// Try to claim without locking

	HTLCClaim(network, beta, "alice", "", nil, "expected only one htlc script to match")
	HTLCClaim(network, alpha, "bob", "", nil, "expected only one htlc script to match")

	CheckBalanceWithLockedAndHolding(network, "alice", "", "EUR", 20, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, "bob", "", "EUR", 10, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, "alice", "", "USD", 10, 0, 0, -1, token.WithTMSID(beta))
	CheckBalanceWithLockedAndHolding(network, "bob", "", "USD", 20, 0, 0, -1, token.WithTMSID(beta))

	CheckPublicParams(network, alpha, "issuer", "auditor", "alice", "bob")
	CheckPublicParams(network, beta, "issuer", "auditor", "alice", "bob")
	CheckOwnerDB(network, alpha, nil, "issuer", "auditor", "alice", "bob")
	CheckOwnerDB(network, beta, nil, "issuer", "auditor", "alice", "bob")
	CheckAuditorDB(network, alpha, "auditor", "", nil)
	CheckAuditorDB(network, beta, "auditor", "", nil)
	PruneInvalidUnspentTokens(network, alpha, "issuer", "auditor", "alice", "bob")
	PruneInvalidUnspentTokens(network, beta, "issuer", "auditor", "alice", "bob")
	for _, name := range []string{"alice", "bob"} {
		aIDs := ListVaultUnspentTokens(network, alpha, name)
		CheckIfExistsInVault(network, alpha, "auditor", aIDs)
		bIDs := ListVaultUnspentTokens(network, beta, name)
		CheckIfExistsInVault(network, beta, "auditor", bIDs)
	}

	// "alice" locks to "alice.id1", the deadline expires, "alice" reclaims, "alice.id1" checks the existence of an expired received locked token
	_, _, h := HTLCLock(network, alpha, "alice", "", "EUR", 10, "alice.id1", 10*time.Second, nil, 0)
	time.Sleep(10 * time.Second)
	HTLCReclaimByHash(network, "alice", "", h)
	HTLCCheckExistenceReceivedExpiredByHash(network, "alice", "alice.id1", h, false)
}

func TestHTLCNoCrossClaimTwoNetworks(network *integration.Infrastructure) {
	// give some time to the nodes to get the public parameters
	time.Sleep(10 * time.Second)

	alpha := token.TMSID{Network: "alpha"}
	beta := token.TMSID{Network: "beta"}

	RegisterAuditor(network, token.WithTMSID(alpha))
	RegisterAuditor(network, token.WithTMSID(beta))

	IssueCashWithTMS(network, alpha, "issuer", "", "EUR", 30, "alice.id1")
	CheckBalanceAndHolding(network, "alice", "alice.id1", "EUR", 30, token.WithTMSID(alpha))

	IssueCashWithTMS(network, beta, "issuer", "", "USD", 30, "bob.id1")
	CheckBalanceAndHolding(network, "bob", "bob.id1", "USD", 30, token.WithTMSID(beta))

	aliceLockTxID, preImage, hash := HTLCLock(network, alpha, "alice", "alice.id1", "EUR", 10, "alice.id2", 30*time.Second, nil, 0)
	bobLockTxID, _, _ := HTLCLock(network, beta, "bob", "bob.id1", "USD", 10, "bob.id2", 30*time.Second, hash, 0)

	go func() { HTLCClaim(network, alpha, "alice", "alice.id2", preImage) }()
	go func() { HTLCClaim(network, beta, "bob", "bob.id2", preImage) }()
	Scan(network, "alice", hash, crypto.SHA256, "", token.WithTMSID(alpha))
	Scan(network, "alice", hash, crypto.SHA256, aliceLockTxID, token.WithTMSID(alpha))

	Scan(network, "bob", hash, crypto.SHA256, "", token.WithTMSID(beta))
	Scan(network, "bob", hash, crypto.SHA256, bobLockTxID, token.WithTMSID(beta))

	CheckBalanceWithLockedAndHolding(network, "alice", "alice.id1", "EUR", 20, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, "alice", "alice.id2", "EUR", 10, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, "bob", "bob.id1", "USD", 20, 0, 0, -1, token.WithTMSID(beta))
	CheckBalanceWithLockedAndHolding(network, "bob", "bob.id2", "USD", 10, 0, 0, -1, token.WithTMSID(beta))

	txID := IssueCashWithTMS(network, alpha, "issuer", "", "EUR", 30, "alice.id1")
	ScanWithError(network, "alice", hash, crypto.SHA256, txID, []string{"timeout reached"}, token.WithTMSID(alpha))

	CheckPublicParams(network, token.TMSID{}, "alice", "bob")
	CheckPublicParams(network, alpha, "issuer", "auditor")
	CheckPublicParams(network, beta, "issuer", "auditor")
	CheckOwnerDB(network, token.TMSID{}, nil, "auditor", "alice", "bob")
	CheckOwnerDB(network, alpha, nil, "issuer")
	CheckOwnerDB(network, beta, nil, "issuer")
	CheckAuditorDB(network, alpha, "auditor", "", nil)
	CheckAuditorDB(network, beta, "auditor", "", nil)
	PruneInvalidUnspentTokens(network, alpha, "issuer", "auditor")
	PruneInvalidUnspentTokens(network, beta, "issuer", "auditor")

	PruneInvalidUnspentTokens(network, alpha, "alice")
	aIDs := ListVaultUnspentTokens(network, alpha, "alice")
	CheckIfExistsInVault(network, alpha, "auditor", aIDs)

	PruneInvalidUnspentTokens(network, beta, "bob")
	bIDs := ListVaultUnspentTokens(network, beta, "bob")
	CheckIfExistsInVault(network, beta, "auditor", bIDs)
}

func TestFastExchange(network *integration.Infrastructure) {
	// give some time to the nodes to get the public parameters
	time.Sleep(10 * time.Second)

	alpha := token.TMSID{Network: "alpha"}
	beta := token.TMSID{Network: "beta"}

	RegisterAuditor(network, token.WithTMSID(alpha))
	RegisterAuditor(network, token.WithTMSID(beta))

	IssueCashWithTMS(network, alpha, "issuer", "", "EUR", 30, "alice")
	CheckBalance(network, "alice", "", "EUR", 30, token.WithTMSID(alpha))

	IssueCashWithTMS(network, beta, "issuer", "", "USD", 30, "bob")
	CheckBalance(network, "bob", "", "USD", 30, token.WithTMSID(beta))

	FastExchange(network, "alice", "bob", alpha, "EUR", 10, beta, "USD", 10, 1*time.Hour)

	CheckBalance(network, "alice", "", "EUR", 20, token.WithTMSID(alpha))
	CheckBalance(network, "bob", "", "EUR", 10, token.WithTMSID(alpha))

	CheckBalance(network, "alice", "", "USD", 10, token.WithTMSID(beta))
	CheckBalance(network, "bob", "", "USD", 20, token.WithTMSID(beta))
}

func TestAssetTransferWithTwoNetworks(network *integration.Infrastructure) {
	// give some time to the nodes to get the public parameters
	time.Sleep(10 * time.Second)

	alpha := token.TMSID{
		Network:   "alpha",
		Channel:   "testchannel",
		Namespace: "tns",
	}
	beta := token.TMSID{
		Network:   "beta",
		Channel:   "testchannel",
		Namespace: "tns",
	}

	alphaURL := pledge2.FabricURL(alpha)
	betaURL := pledge2.FabricURL(beta)

	RegisterAuditor(network, token.WithTMSID(alpha))
	RegisterAuditor(network, token.WithTMSID(beta))

	IssueCashWithTMS(network, alpha, "issuerAlpha", "", "USD", 50, "alice")
	IssueCashWithTMS(network, alpha, "issuerAlpha", "", "EUR", 10, "alice")
	CheckBalance(network, "alice", "", "USD", 50)
	CheckBalance(network, "alice", "", "EUR", 10)

	// Pledge + Claim
	txid, pledgeid := Pledge(network, "alice", "", "USD", 50, "bob", "issuerAlpha", betaURL, time.Minute*1, token.WithTMSID(alpha))
	CheckBalance(network, "alice", "", "USD", 0)
	CheckBalance(network, "bob", "", "USD", 0)

	PledgeIDExists(network, "alice", pledgeid, txid, token.WithTMSID(alpha))

	Claim(network, "bob", "issuerBeta", &token2.ID{TxId: txid, Index: 0})
	CheckBalance(network, "alice", "", "USD", 0)
	CheckBalance(network, "bob", "", "USD", 50)

	time.Sleep(time.Minute * 1)
	RedeemWithTMS(network, "issuerAlpha", &token2.ID{TxId: txid, Index: 0}, token.WithTMSID(alpha))
	CheckBalance(network, "alice", "", "USD", 0)
	CheckBalance(network, "bob", "", "USD", 50)

	// Pledge + Reclaim

	IssueCashWithTMS(network, alpha, "issuerAlpha", "", "USD", 50, "alice")
	CheckBalance(network, "alice", "", "USD", 50, token.WithTMSID(alpha))
	txid, _ = Pledge(network, "alice", "", "USD", 50, "bob", "issuerAlpha", betaURL, time.Second*10, token.WithTMSID(alpha))

	time.Sleep(time.Second * 15)
	Reclaim(network, "alice", "", txid, token.WithTMSID(alpha))
	CheckBalance(network, "alice", "", "USD", 50, token.WithTMSID(alpha))
	CheckBalance(network, "bob", "", "USD", 50)

	RedeemWithTMSAndError(network, "issuerAlpha", &token2.ID{TxId: txid, Index: 0}, token.WithTMSID(alpha))
	CheckBalance(network, "alice", "", "USD", 50, token.WithTMSID(alpha))
	CheckBalance(network, "bob", "", "USD", 50)

	ScanPledgeIDWithError(network, "alice", pledgeid, txid, []string{"timeout reached"}, token.WithTMSID(alpha))

	// Try to reclaim again

	ReclaimWithError(network, "alice", "", txid, token.WithTMSID(alpha))
	CheckBalance(network, "alice", "", "USD", 50, token.WithTMSID(alpha))
	CheckBalance(network, "bob", "", "USD", 50)

	// Try to claim after reclaim

	ClaimWithError(network, "bob", "issuerBeta", &token2.ID{TxId: txid, Index: 0})
	CheckBalance(network, "alice", "", "USD", 50, token.WithTMSID(alpha))
	CheckBalance(network, "bob", "", "USD", 50)

	// Try to reclaim after claim

	txid, _ = Pledge(network, "alice", "", "USD", 10, "bob", "issuerAlpha", betaURL, time.Minute*1, token.WithTMSID(alpha))
	CheckBalance(network, "alice", "", "USD", 40)
	CheckBalance(network, "bob", "", "USD", 50)

	Claim(network, "bob", "issuerBeta", &token2.ID{TxId: txid, Index: 0})
	CheckBalance(network, "alice", "", "USD", 40)
	CheckBalance(network, "bob", "", "USD", 60)

	time.Sleep(time.Minute * 1)

	ReclaimWithError(network, "alice", "", txid, token.WithTMSID(alpha))
	CheckBalance(network, "alice", "", "USD", 40)
	CheckBalance(network, "bob", "", "USD", 60)

	// Try to claim after claim

	txid, pledgeid = Pledge(network, "bob", "", "USD", 5, "alice", "issuerBeta", alphaURL, time.Minute*1, token.WithTMSID(beta))
	CheckBalance(network, "alice", "", "USD", 40)
	CheckBalance(network, "bob", "", "USD", 55)

	PledgeIDExists(network, "bob", pledgeid, txid, token.WithTMSID(beta))

	Claim(network, "alice", "issuerAlpha", &token2.ID{TxId: txid, Index: 0})
	CheckBalance(network, "alice", "", "USD", 45)
	CheckBalance(network, "bob", "", "USD", 55)

	ClaimWithError(network, "alice", "issuerAlpha", &token2.ID{TxId: txid, Index: 0})
	CheckBalance(network, "alice", "", "USD", 45)
	CheckBalance(network, "bob", "", "USD", 55)

	time.Sleep(1 * time.Minute)
	RedeemWithTMS(network, "issuerBeta", &token2.ID{TxId: txid, Index: 0}, token.WithTMSID(beta))
	CheckBalance(network, "alice", "", "USD", 45)
	CheckBalance(network, "bob", "", "USD", 55)

	// Try to redeem again
	RedeemWithTMSAndError(network, "issuerBeta", &token2.ID{TxId: txid, Index: 0}, token.WithTMSID(beta), "failed to retrieve pledged token during redeem")
	CheckBalance(network, "alice", "", "USD", 45)
	CheckBalance(network, "bob", "", "USD", 55)

	// Try to claim or reclaim without pledging

	ClaimWithError(network, "alice", "issuerAlpha", &token2.ID{TxId: "", Index: 0})
	CheckBalance(network, "alice", "", "USD", 45)
	CheckBalance(network, "bob", "", "USD", 55)

	ReclaimWithError(network, "alice", "", "", token.WithTMSID(alpha))
	CheckBalance(network, "alice", "", "USD", 45)
	CheckBalance(network, "bob", "", "USD", 55)

	// Fast Pledge + Claim
	FastTransferPledgeClaim(network, "alice", "", "USD", 10, "bob", "issuerAlpha", betaURL, time.Minute*1, token.WithTMSID(alpha))
	CheckBalance(network, "alice", "", "USD", 35)
	CheckBalance(network, "bob", "", "USD", 65)

	// Fast Pledge + Reclaim
	FastTransferPledgeReclaim(network, "alice", "", "USD", 10, "bob", "issuerAlpha", betaURL, time.Second*5, token.WithTMSID(alpha))
	CheckBalance(network, "alice", "", "USD", 35)
	CheckBalance(network, "bob", "", "USD", 65)
}
