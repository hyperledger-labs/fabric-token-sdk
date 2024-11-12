/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interop

import (
	"crypto"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop/views"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	auditor2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/state/fabric"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

func TestHTLCSingleNetwork(network *integration.Infrastructure, sel *token2.ReplicaSelector) {
	alice := sel.Get("alice")
	bob := sel.Get("bob")
	issuer := sel.Get("issuer")
	auditor := sel.Get("auditor")

	// give some time to the nodes to get the public parameters
	time.Sleep(10 * time.Second)

	defaultTMSID := token.TMSID{}
	RegisterAuditor(network)

	IssueCash(network, "", "USD", 110, alice, alice)
	CheckBalanceWithLockedAndHolding(network, alice, "", "USD", 110, 0, 0, -1)
	IssueCash(network, "", "USD", 10, alice, alice)
	CheckBalanceWithLockedAndHolding(network, alice, "", "USD", 120, 0, 0, -1)

	h := ListIssuerHistory(network, "", "USD")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(120))).To(BeEquivalentTo(0))
	Expect(h.ByType("USD").Count()).To(BeEquivalentTo(h.Count()))

	h = ListIssuerHistory(network, "", "EUR")
	Expect(h.Count()).To(BeEquivalentTo(0))

	IssueCash(network, "", "EUR", 10, bob, bob)
	CheckBalanceWithLockedAndHolding(network, bob, "", "EUR", 10, 0, 0, -1)
	IssueCash(network, "", "EUR", 10, bob, bob)
	CheckBalanceWithLockedAndHolding(network, bob, "", "EUR", 20, 0, 0, -1)
	IssueCash(network, "", "EUR", 10, bob, bob)
	CheckBalanceWithLockedAndHolding(network, bob, "", "EUR", 30, 0, 0, -1)

	h = ListIssuerHistory(network, "", "USD")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(120))).To(BeEquivalentTo(0))
	Expect(h.ByType("USD").Count()).To(BeEquivalentTo(h.Count()))

	h = ListIssuerHistory(network, "", "EUR")
	Expect(h.Count() > 0).To(BeTrue())
	Expect(h.Sum(64).ToBigInt().Cmp(big.NewInt(30))).To(BeEquivalentTo(0))
	Expect(h.ByType("EUR").Count()).To(BeEquivalentTo(h.Count()))

	CheckBalanceWithLockedAndHolding(network, alice, "", "USD", 120, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, alice, "", "EUR", 0, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, bob, "", "EUR", 30, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, bob, "", "USD", 0, 0, 0, -1)

	Restart(network, issuer, auditor, alice, bob)
	RegisterAuditor(network)

	// htlc (lock, failing claim, reclaim)
	_, preImage, _ := HTLCLock(network, token.TMSID{}, alice, "", "USD", 10, bob, auditor, 10*time.Second, nil, crypto.SHA512)
	CheckBalanceWithLockedAndHolding(network, alice, "", "USD", 110, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, alice, "", "EUR", 0, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, bob, "", "EUR", 30, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, bob, "", "USD", 0, 10, 0, -1)
	time.Sleep(15 * time.Second)
	CheckBalanceWithLockedAndHolding(network, alice, "", "USD", 110, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, alice, "", "EUR", 0, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, bob, "", "EUR", 30, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, bob, "", "USD", 0, 0, 10, -1)
	htlcClaim(network, defaultTMSID, bob, "", preImage, auditor, "deadline elapsed")
	CheckBalanceWithLockedAndHolding(network, alice, "", "USD", 110, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, alice, "", "EUR", 0, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, bob, "", "EUR", 30, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, bob, "", "USD", 0, 0, 10, -1)

	HTLCReclaimAll(network, alice, "")
	CheckBalanceWithLockedAndHolding(network, alice, "", "USD", 120, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, alice, "", "EUR", 0, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, bob, "", "EUR", 30, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, bob, "", "USD", 0, 0, 0, -1)

	// htlc (lock, claim)
	_, preImage, _ = HTLCLock(network, defaultTMSID, alice, "", "USD", 20, bob, auditor, 1*time.Hour, nil, crypto.SHA3_256)
	CheckBalanceWithLockedAndHolding(network, alice, "", "USD", 100, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, alice, "", "EUR", 0, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, bob, "", "EUR", 30, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, bob, "", "USD", 0, 20, 0, -1)

	htlcClaim(network, defaultTMSID, bob, "", preImage, auditor)
	CheckBalanceWithLockedAndHolding(network, alice, "", "USD", 100, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, alice, "", "EUR", 0, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, bob, "", "EUR", 30, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, bob, "", "USD", 20, 0, 0, -1)

	// payment limit reached
	lockTxID, _, _ := HTLCLock(network, defaultTMSID, alice, "", "USD", uint64(views.Limit+10), bob, auditor, 1*time.Hour, nil, crypto.SHA3_256, "payment limit reached")
	CheckBalanceWithLockedAndHolding(network, alice, "", "USD", 100, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, alice, "", "EUR", 0, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, bob, "", "EUR", 30, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, bob, "", "USD", 20, 0, 0, -1)

	CheckPublicParams(network, defaultTMSID, issuer, auditor, alice, bob)
	<-time.After(30 * time.Second)
	CheckOwnerDB(network, defaultTMSID, nil, issuer, alice, bob)
	CheckAuditorDB(network, defaultTMSID, auditor, "", func(errs []string) error {
		if len(errs) != 1 {
			return errors.Errorf("expected 1 errors, got [%d][%v][%s]", len(errs), errs, lockTxID)
		}
		for _, err := range errs {
			if strings.Contains(err, lockTxID) {
				return errors.Errorf("[%s] does not contain [%s]", err, lockTxID)
			}
		}
		return nil
	})

	// lock two times with the same hash, the second lock should fail
	_, _, hash := HTLCLock(network, defaultTMSID, alice, "", "USD", 1, bob, auditor, 1*time.Hour, nil, crypto.SHA3_256)
	failedLockTXID, _, _ := HTLCLock(network, defaultTMSID, alice, "", "USD", 1, bob, auditor, 1*time.Hour, hash, crypto.SHA3_256,
		fmt.Sprintf(
			"entry with transfer metadata key [%s] is already occupied",
			htlc.LockKey(hash),
		),
	)
	HTLCLock(network, defaultTMSID, alice, "", "USD", 1, bob, auditor, 1*time.Hour, nil, crypto.SHA3_256)

	CheckPublicParams(network, defaultTMSID, issuer, auditor, alice, bob)
	CheckOwnerDB(network, defaultTMSID, nil, issuer, auditor, alice, bob)
	CheckAuditorDB(network, defaultTMSID, auditor, "", func(errs []string) error {
		// We should get here 3 errors:
		// - One from before;
		// - Two for failedLockTXID (one entry for the lock to Bob, the other relative to the rest to Alice)
		fmt.Printf("Got errors [%v]", errs)
		if len(errs) != 3 {
			return errors.Errorf("expected 3 errors, got [%d][%v][%s]", len(errs), errs, lockTxID)
		}
		for _, err := range errs[:1] {
			if strings.Contains(err, lockTxID) {
				return errors.Errorf("[%s] does not contain [%s]", err, lockTxID)
			}
		}
		firstError := fmt.Sprintf("transaction record [%s] is unknown for vault but not for the db [%s]", failedLockTXID, auditor2.TxStatusMessage[auditor2.Pending])
		if errs[1] != firstError {
			return errors.Errorf("expected first error to be [%s], got [%s]", firstError, errs[0])
		}
		for _, err := range errs[1:] {
			if !strings.Contains(err, failedLockTXID) {
				return errors.Errorf("[%s] does not contain [%s]", err, failedLockTXID)
			}
		}
		return nil
	})
	PruneInvalidUnspentTokens(network, defaultTMSID, issuer, auditor, alice, bob)
	for _, name := range []*token2.NodeReference{alice, bob} {
		IDs := ListVaultUnspentTokens(network, defaultTMSID, name)
		CheckIfExistsInVault(network, defaultTMSID, auditor, IDs)
	}
}

func TestHTLCTwoNetworks(network *integration.Infrastructure, sel *token2.ReplicaSelector) {
	issuer := sel.Get("issuer")
	alice := sel.Get("alice")
	bob := sel.Get("bob")
	auditor := sel.Get("auditor")

	// give some time to the nodes to get the public parameters
	time.Sleep(10 * time.Second)

	alpha := token.TMSID{Network: "alpha"}
	beta := token.TMSID{Network: "beta"}

	RegisterAuditor(network, token.WithTMSID(alpha))
	RegisterAuditor(network, token.WithTMSID(beta))

	IssueCashWithTMS(network, alpha, issuer, "", "EUR", 30, alice, auditor)
	CheckBalanceAndHolding(network, alice, "", "EUR", 30, token.WithTMSID(alpha))

	IssueCashWithTMS(network, beta, issuer, "", "USD", 30, bob, auditor)
	CheckBalanceAndHolding(network, bob, "", "USD", 30, token.WithTMSID(beta))

	_, preImage, hash := HTLCLock(network, alpha, alice, "", "EUR", 10, bob, auditor, 1*time.Hour, nil, 0)
	HTLCLock(network, beta, bob, "", "USD", 10, alice, auditor, 1*time.Hour, hash, 0)
	htlcClaim(network, beta, alice, "", preImage, auditor)
	htlcClaim(network, alpha, bob, "", preImage, auditor)

	CheckBalanceWithLockedAndHolding(network, alice, "", "EUR", 20, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, bob, "", "EUR", 10, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, alice, "", "USD", 10, 0, 0, -1, token.WithTMSID(beta))
	CheckBalanceWithLockedAndHolding(network, bob, "", "USD", 20, 0, 0, -1, token.WithTMSID(beta))

	// Try to claim again and get an error

	htlcClaim(network, beta, alice, "", preImage, auditor, "expected only one htlc script to match")
	htlcClaim(network, alpha, bob, "", preImage, auditor, "expected only one htlc script to match")

	CheckBalanceWithLockedAndHolding(network, alice, "", "EUR", 20, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, bob, "", "EUR", 10, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, alice, "", "USD", 10, 0, 0, -1, token.WithTMSID(beta))
	CheckBalanceWithLockedAndHolding(network, bob, "", "USD", 20, 0, 0, -1, token.WithTMSID(beta))

	// Try to claim without locking

	htlcClaim(network, beta, alice, "", nil, auditor, "expected only one htlc script to match")
	htlcClaim(network, alpha, bob, "", nil, auditor, "expected only one htlc script to match")

	CheckBalanceWithLockedAndHolding(network, alice, "", "EUR", 20, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, bob, "", "EUR", 10, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, alice, "", "USD", 10, 0, 0, -1, token.WithTMSID(beta))
	CheckBalanceWithLockedAndHolding(network, bob, "", "USD", 20, 0, 0, -1, token.WithTMSID(beta))

	CheckPublicParams(network, alpha, issuer, auditor, alice, bob)
	CheckPublicParams(network, beta, issuer, auditor, alice, bob)
	CheckOwnerDB(network, alpha, nil, issuer, auditor, alice, bob)
	CheckOwnerDB(network, beta, nil, issuer, auditor, alice, bob)
	CheckAuditorDB(network, alpha, sel.Get("auditor"), "", nil)
	CheckAuditorDB(network, beta, sel.Get("auditor"), "", nil)
	PruneInvalidUnspentTokens(network, alpha, issuer, auditor, alice, bob)
	PruneInvalidUnspentTokens(network, beta, issuer, auditor, alice, bob)
	for _, name := range []*token2.NodeReference{alice, bob} {
		aIDs := ListVaultUnspentTokens(network, alpha, name)
		CheckIfExistsInVault(network, alpha, auditor, aIDs)
		bIDs := ListVaultUnspentTokens(network, beta, name)
		CheckIfExistsInVault(network, beta, auditor, bIDs)
	}

	// "alice" locks to "alice.id1", the deadline expires, "alice" reclaims, "alice.id1" checks the existence of an expired received locked token
	_, _, h := HTLCLock(network, alpha, alice, "", "EUR", 10, sel.Get("alice.id1"), auditor, 10*time.Second, nil, 0)
	time.Sleep(10 * time.Second)
	HTLCReclaimByHash(network, alpha, alice, "", h)
	HTLCCheckExistenceReceivedExpiredByHash(network, alpha, alice, "alice.id1", h, false)
}

func TestHTLCNoCrossClaimTwoNetworks(network *integration.Infrastructure, sel *token2.ReplicaSelector) {
	issuer := sel.Get("issuer")
	alice := sel.Get("alice")
	bob := sel.Get("bob")
	auditor := sel.Get("auditor")

	// give some time to the nodes to get the public parameters
	time.Sleep(10 * time.Second)

	alpha := token.TMSID{Network: "alpha"}
	beta := token.TMSID{Network: "beta"}

	RegisterAuditor(network, token.WithTMSID(alpha))
	RegisterAuditor(network, token.WithTMSID(beta))

	IssueCashWithTMS(network, alpha, issuer, "", "EUR", 30, sel.Get("alice.id1"), auditor)
	CheckBalanceAndHolding(network, alice, "alice.id1", "EUR", 30, token.WithTMSID(alpha))

	IssueCashWithTMS(network, beta, issuer, "", "USD", 30, sel.Get("bob.id1"), auditor)
	CheckBalanceAndHolding(network, bob, "bob.id1", "USD", 30, token.WithTMSID(beta))

	aliceLockTxID, preImage, hash := HTLCLock(network, alpha, alice, "alice.id1", "EUR", 10, sel.Get("alice.id2"), auditor, 30*time.Second, nil, 0)
	bobLockTxID, _, _ := HTLCLock(network, beta, bob, "bob.id1", "USD", 10, sel.Get("bob.id2"), auditor, 30*time.Second, hash, 0)

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		Expect(func() {
			htlcClaim(network, alpha, alice, "alice.id2", preImage, auditor)
		}).ToNot(Panic())
	}()
	go func() {
		defer wg.Done()
		Expect(func() {
			htlcClaim(network, beta, bob, "bob.id2", preImage, auditor)
		}).ToNot(Panic())
	}()
	scan(network, alice, hash, crypto.SHA256, "", false, token.WithTMSID(alpha))
	scan(network, alice, hash, crypto.SHA256, aliceLockTxID, false, token.WithTMSID(alpha))

	scan(network, bob, hash, crypto.SHA256, "", false, token.WithTMSID(beta))
	scan(network, bob, hash, crypto.SHA256, bobLockTxID, false, token.WithTMSID(beta))

	// wait the completion of the claims above
	wg.Wait()

	CheckBalanceWithLockedAndHolding(network, alice, "alice.id1", "EUR", 20, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, alice, "alice.id2", "EUR", 10, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, bob, "bob.id1", "USD", 20, 0, 0, -1, token.WithTMSID(beta))
	CheckBalanceWithLockedAndHolding(network, bob, "bob.id2", "USD", 10, 0, 0, -1, token.WithTMSID(beta))

	txID := IssueCashWithTMS(network, alpha, issuer, "", "EUR", 30, sel.Get("alice.id1"), auditor)

	scan(network, bob, hash, crypto.SHA256, bobLockTxID, true, token.WithTMSID(beta))
	start := time.Now()
	scanWithError(network, alice, hash, crypto.SHA256, txID, []string{"context done"}, true, token.WithTMSID(alpha))
	Expect(time.Since(start)).To(BeNumerically("<", time.Second*30), "scan should be canceled on last tx, before timeout")
	scanWithError(network, alice, hash, crypto.SHA256, txID, []string{"timeout reached"}, false, token.WithTMSID(alpha))

	CheckPublicParams(network, token.TMSID{}, alice, bob)
	CheckPublicParams(network, alpha, issuer, auditor)
	CheckPublicParams(network, beta, issuer, auditor)
	CheckOwnerDB(network, token.TMSID{}, nil, auditor, alice, bob)
	CheckOwnerDB(network, alpha, nil, issuer)
	CheckOwnerDB(network, beta, nil, issuer)
	CheckAuditorDB(network, alpha, auditor, "", nil)
	CheckAuditorDB(network, beta, auditor, "", nil)
	PruneInvalidUnspentTokens(network, alpha, issuer, auditor)
	PruneInvalidUnspentTokens(network, beta, issuer, auditor)

	PruneInvalidUnspentTokens(network, alpha, alice)
	aIDs := ListVaultUnspentTokens(network, alpha, alice)
	CheckIfExistsInVault(network, alpha, sel.Get("auditor"), aIDs)

	PruneInvalidUnspentTokens(network, beta, bob)
	bIDs := ListVaultUnspentTokens(network, beta, bob)
	CheckIfExistsInVault(network, beta, sel.Get("auditor"), bIDs)
}

func TestFastExchange(network *integration.Infrastructure, sel *token2.ReplicaSelector) {
	issuer := sel.Get("issuer")
	alice := sel.Get("alice")
	bob := sel.Get("bob")
	auditor := sel.Get("auditor")

	// give some time to the nodes to get the public parameters
	time.Sleep(10 * time.Second)

	alpha := token.TMSID{Network: "alpha"}
	beta := token.TMSID{Network: "beta"}

	RegisterAuditor(network, token.WithTMSID(alpha))
	RegisterAuditor(network, token.WithTMSID(beta))

	IssueCashWithTMS(network, alpha, issuer, "", "EUR", 30, alice, auditor)
	CheckBalance(network, sel.Get("alice"), "", "EUR", 30, token.WithTMSID(alpha))

	IssueCashWithTMS(network, beta, issuer, "", "USD", 30, bob, auditor)
	CheckBalance(network, bob, "", "USD", 30, token.WithTMSID(beta))

	fastExchange(network, alice, bob, alpha, "EUR", 10, beta, "USD", 10, 1*time.Hour)

	CheckBalance(network, sel.Get("alice"), "", "EUR", 20, token.WithTMSID(alpha))
	Eventually(CheckBalanceReturnError).WithArguments(network, sel.Get("bob"), "", "EUR", uint64(10), token.WithTMSID(alpha)).WithTimeout(1 * time.Minute).WithPolling(15 * time.Second).Should(Succeed())

	CheckBalance(network, sel.Get("alice"), "", "USD", 10, token.WithTMSID(beta))
	Eventually(CheckBalanceReturnError).WithArguments(network, sel.Get("bob"), "", "USD", uint64(20), token.WithTMSID(beta)).WithTimeout(1 * time.Minute).WithPolling(15 * time.Second).Should(Succeed())
}

func TestAssetTransferWithTwoNetworks(network *integration.Infrastructure, sel *token2.ReplicaSelector) {
	issuer := sel.Get("issuer")
	alice := sel.Get("alice")
	bob := sel.Get("bob")
	auditor := sel.Get("auditor")

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

	alphaURL := fabric.FabricURL(alpha)
	betaURL := fabric.FabricURL(beta)

	RegisterAuditor(network, token.WithTMSID(alpha))
	RegisterAuditor(network, token.WithTMSID(beta))

	IssueCashWithTMS(network, alpha, "issuerAlpha", "", "USD", 50, "alice")
	IssueCashWithTMS(network, alpha, "issuerAlpha", "", "EUR", 10, "alice")
	CheckBalance(network, alice, "", "USD", 50)
	CheckBalance(network, alice, "", "EUR", 10)

	// Pledge + Claim
	txid, pledgeid := Pledge(network, alice, "", "USD", 50, "bob", "issuerAlpha", betaURL, time.Minute*1, token.WithTMSID(alpha))
	CheckBalance(network, alice, "", "USD", 0)
	CheckBalance(network, "bob", "", "USD", 0)

	PledgeIDExists(network, alice, pledgeid, txid, token.WithTMSID(alpha))

	Claim(network, "bob", "issuerBeta", &token2.ID{TxId: txid, Index: 0})
	CheckBalance(network, alice, "", "USD", 0)
	CheckBalance(network, "bob", "", "USD", 50)

	time.Sleep(time.Minute * 1)
	RedeemWithTMS(network, "issuerAlpha", &token2.ID{TxId: txid, Index: 0}, token.WithTMSID(alpha))
	CheckBalance(network, alice, "", "USD", 0)
	CheckBalance(network, "bob", "", "USD", 50)

	// Pledge + Reclaim

	IssueCashWithTMS(network, alpha, "issuerAlpha", "", "USD", 50, alice)
	CheckBalance(network, alice, "", "USD", 50, token.WithTMSID(alpha))
	txid, _ = Pledge(network, alice, "", "USD", 50, "bob", "issuerAlpha", betaURL, time.Second*10, token.WithTMSID(alpha))

	time.Sleep(time.Second * 15)
	Reclaim(network, alice, "", txid, token.WithTMSID(alpha))
	CheckBalance(network, alice, "", "USD", 50, token.WithTMSID(alpha))
	CheckBalance(network, "bob", "", "USD", 50)

	RedeemWithTMSAndError(network, "issuerAlpha", &token2.ID{TxId: txid, Index: 0}, token.WithTMSID(alpha))
	CheckBalance(network, alice, "", "USD", 50, token.WithTMSID(alpha))
	CheckBalance(network, "bob", "", "USD", 50)

	ScanPledgeIDWithError(network, alice, pledgeid, txid, []string{"timeout reached"}, token.WithTMSID(alpha))

	// Try to reclaim again

	ReclaimWithError(network, alice, "", txid, token.WithTMSID(alpha))
	CheckBalance(network, alice, "", "USD", 50, token.WithTMSID(alpha))
	CheckBalance(network, "bob", "", "USD", 50)

	// Try to claim after reclaim

	ClaimWithError(network, "bob", "issuerBeta", &token2.ID{TxId: txid, Index: 0})
	CheckBalance(network, alice, "", "USD", 50, token.WithTMSID(alpha))
	CheckBalance(network, "bob", "", "USD", 50)

	// Try to reclaim after claim

	txid, _ = Pledge(network, alice, "", "USD", 10, "bob", "issuerAlpha", betaURL, time.Minute*1, token.WithTMSID(alpha))
	CheckBalance(network, alice, "", "USD", 40)
	CheckBalance(network, "bob", "", "USD", 50)

	Claim(network, "bob", "issuerBeta", &token2.ID{TxId: txid, Index: 0})
	CheckBalance(network, alice, "", "USD", 40)
	CheckBalance(network, "bob", "", "USD", 60)

	time.Sleep(time.Minute * 1)

	ReclaimWithError(network, alice, "", txid, token.WithTMSID(alpha))
	CheckBalance(network, alice, "", "USD", 40)
	CheckBalance(network, "bob", "", "USD", 60)

	// Try to claim after claim

	txid, pledgeid = Pledge(network, "bob", "", "USD", 5, alice, "issuerBeta", alphaURL, time.Minute*1, token.WithTMSID(beta))
	CheckBalance(network, alice, "", "USD", 40)
	CheckBalance(network, "bob", "", "USD", 55)

	PledgeIDExists(network, "bob", pledgeid, txid, token.WithTMSID(beta))

	Claim(network, alice, "issuerAlpha", &token2.ID{TxId: txid, Index: 0})
	CheckBalance(network, alice, "", "USD", 45)
	CheckBalance(network, "bob", "", "USD", 55)

	ClaimWithError(network, alice, "issuerAlpha", &token2.ID{TxId: txid, Index: 0})
	CheckBalance(network, alice, "", "USD", 45)
	CheckBalance(network, "bob", "", "USD", 55)

	time.Sleep(1 * time.Minute)
	RedeemWithTMS(network, "issuerBeta", &token2.ID{TxId: txid, Index: 0}, token.WithTMSID(beta))
	CheckBalance(network, alice, "", "USD", 45)
	CheckBalance(network, "bob", "", "USD", 55)

	// Try to redeem again
	RedeemWithTMSAndError(network, "issuerBeta", &token2.ID{TxId: txid, Index: 0}, token.WithTMSID(beta), "failed to retrieve pledged token during redeem")
	CheckBalance(network, alice, "", "USD", 45)
	CheckBalance(network, "bob", "", "USD", 55)

	// Try to claim or reclaim without pledging

	ClaimWithError(network, alice, "issuerAlpha", &token2.ID{TxId: "", Index: 0})
	CheckBalance(network, alice, "", "USD", 45)
	CheckBalance(network, "bob", "", "USD", 55)

	ReclaimWithError(network, alice, "", "", token.WithTMSID(alpha))
	CheckBalance(network, alice, "", "USD", 45)
	CheckBalance(network, "bob", "", "USD", 55)

	// Fast Pledge + Claim
	FastTransferPledgeClaim(network, alice, "", "USD", 10, "bob", "issuerAlpha", betaURL, time.Minute*1, token.WithTMSID(alpha))
	CheckBalance(network, alice, "", "USD", 35)
	CheckBalance(network, "bob", "", "USD", 65)

	// Fast Pledge + Reclaim
	FastTransferPledgeReclaim(network, alice, "", "USD", 10, "bob", "issuerAlpha", betaURL, time.Second*5, token.WithTMSID(alpha))
	CheckBalance(network, alice, "", "USD", 35)
	CheckBalance(network, "bob", "", "USD", 65)
}
