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
	htlcClaim(network, defaultTMSID, "bob", "", preImage, "deadline elapsed")
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

	htlcClaim(network, defaultTMSID, "bob", "", preImage)
	CheckBalanceWithLockedAndHolding(network, "alice", "", "USD", 100, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "alice", "", "EUR", 0, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "bob", "", "EUR", 30, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "bob", "", "USD", 20, 0, 0, -1)

	// payment limit reached
	lockTxID, _, _ := HTLCLock(network, defaultTMSID, "alice", "", "USD", uint64(views.Limit+10), "bob", 1*time.Hour, nil, crypto.SHA3_256, "payment limit reached")
	CheckBalanceWithLockedAndHolding(network, "alice", "", "USD", 100, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "alice", "", "EUR", 0, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "bob", "", "EUR", 30, 0, 0, -1)
	CheckBalanceWithLockedAndHolding(network, "bob", "", "USD", 20, 0, 0, -1)

	CheckPublicParams(network, defaultTMSID, "issuer", "auditor", "alice", "bob")
	CheckOwnerDB(network, defaultTMSID, nil, "issuer", "alice", "bob")
	CheckAuditorDB(network, defaultTMSID, "auditor", "", func(errs []string) error {
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
	_, _, hash := HTLCLock(network, defaultTMSID, "alice", "", "USD", 1, "bob", 1*time.Hour, nil, crypto.SHA3_256)
	failedLockTXID, _, _ := HTLCLock(network, defaultTMSID, "alice", "", "USD", 1, "bob", 1*time.Hour, hash, crypto.SHA3_256,
		fmt.Sprintf(
			"entry with transfer metadata key [%s] is already occupied by [%s]",
			htlc.LockKey(hash),
			base64.StdEncoding.EncodeToString(htlc.LockValue(hash)),
		),
	)
	HTLCLock(network, defaultTMSID, "alice", "", "USD", 1, "bob", 1*time.Hour, nil, crypto.SHA3_256)

	CheckPublicParams(network, defaultTMSID, sel.All("issuer", "auditor", "alice", "bob")...)
	CheckOwnerDB(network, defaultTMSID, nil, sel.All("issuer", "auditor", "alice", "bob")...)
	CheckAuditorDB(network, defaultTMSID, sel.Get("auditor"), "", func(errs []string) error {
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
		firstError := fmt.Sprintf("transaction record [%s] is unknown for vault but not for the db [%s]", failedLockTXID, auditor.TxStatusMessage[auditor.Pending])
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
	PruneInvalidUnspentTokens(network, defaultTMSID, sel.All("issuer", "auditor", "alice", "bob")...)
	for _, name := range []string{"alice", "bob"} {
		IDs := ListVaultUnspentTokens(network, defaultTMSID, name)
		CheckIfExistsInVault(network, defaultTMSID, sel.Get("auditor"), IDs)
	}
}

func TestHTLCTwoNetworks(network *integration.Infrastructure, sel *token2.ReplicaSelector) {
	// give some time to the nodes to get the public parameters
	time.Sleep(10 * time.Second)

	alpha := token.TMSID{Network: "alpha"}
	beta := token.TMSID{Network: "beta"}

	RegisterAuditor(network, token.WithTMSID(alpha))
	RegisterAuditor(network, token.WithTMSID(beta))

	IssueCashWithTMS(network, alpha, sel.Get("issuer"), "", "EUR", 30, "alice")
	CheckBalanceAndHolding(network, sel.Get("alice"), "", "EUR", 30, token.WithTMSID(alpha))

	IssueCashWithTMS(network, beta, sel.Get("issuer"), "", "USD", 30, "bob")
	CheckBalanceAndHolding(network, sel.Get("bob"), "", "USD", 30, token.WithTMSID(beta))

	_, preImage, hash := HTLCLock(network, alpha, sel.Get("alice"), "", "EUR", 10, "bob", 1*time.Hour, nil, 0)
	HTLCLock(network, beta, sel.Get("bob"), "", "USD", 10, "alice", 1*time.Hour, hash, 0)
	htlcClaim(network, beta, sel.Get("alice"), "", preImage)
	htlcClaim(network, alpha, sel.Get("bob"), "", preImage)

	CheckBalanceWithLockedAndHolding(network, sel.Get("alice"), "", "EUR", 20, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, sel.Get("bob"), "", "EUR", 10, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, sel.Get("alice"), "", "USD", 10, 0, 0, -1, token.WithTMSID(beta))
	CheckBalanceWithLockedAndHolding(network, sel.Get("bob"), "", "USD", 20, 0, 0, -1, token.WithTMSID(beta))

	// Try to claim again and get an error

	htlcClaim(network, beta, sel.Get("alice"), "", preImage, "expected only one htlc script to match")
	htlcClaim(network, alpha, sel.Get("bob"), "", preImage, "expected only one htlc script to match")

	CheckBalanceWithLockedAndHolding(network, sel.Get("alice"), "", "EUR", 20, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, sel.Get("bob"), "", "EUR", 10, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, sel.Get("alice"), "", "USD", 10, 0, 0, -1, token.WithTMSID(beta))
	CheckBalanceWithLockedAndHolding(network, sel.Get("bob"), "", "USD", 20, 0, 0, -1, token.WithTMSID(beta))

	// Try to claim without locking

	htlcClaim(network, beta, sel.Get("alice"), "", nil, "expected only one htlc script to match")
	htlcClaim(network, alpha, sel.Get("bob"), "", nil, "expected only one htlc script to match")

	CheckBalanceWithLockedAndHolding(network, sel.Get("alice"), "", "EUR", 20, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, sel.Get("bob"), "", "EUR", 10, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, sel.Get("alice"), "", "USD", 10, 0, 0, -1, token.WithTMSID(beta))
	CheckBalanceWithLockedAndHolding(network, sel.Get("bob"), "", "USD", 20, 0, 0, -1, token.WithTMSID(beta))

	CheckPublicParams(network, alpha, sel.All("issuer", "auditor", "alice", "bob")...)
	CheckPublicParams(network, beta, sel.All("issuer", "auditor", "alice", "bob")...)
	CheckOwnerDB(network, alpha, nil, sel.All("issuer", "auditor", "alice", "bob")...)
	CheckOwnerDB(network, beta, nil, sel.All("issuer", "auditor", "alice", "bob")...)
	CheckAuditorDB(network, alpha, sel.Get("auditor"), "", nil)
	CheckAuditorDB(network, beta, sel.Get("auditor"), "", nil)
	PruneInvalidUnspentTokens(network, alpha, sel.All("issuer", "auditor", "alice", "bob")...)
	PruneInvalidUnspentTokens(network, beta, sel.All("issuer", "auditor", "alice", "bob")...)
	for _, name := range sel.All("alice", "bob") {
		aIDs := ListVaultUnspentTokens(network, alpha, name)
		CheckIfExistsInVault(network, alpha, sel.Get("auditor"), aIDs)
		bIDs := ListVaultUnspentTokens(network, beta, name)
		CheckIfExistsInVault(network, beta, sel.Get("auditor"), bIDs)
	}

	// "alice" locks to "alice.id1", the deadline expires, "alice" reclaims, "alice.id1" checks the existence of an expired received locked token
	_, _, h := HTLCLock(network, alpha, sel.Get("alice"), "", "EUR", 10, "alice.id1", 10*time.Second, nil, 0)
	time.Sleep(10 * time.Second)
	HTLCReclaimByHash(network, alpha, sel.Get("alice"), "", h)
	HTLCCheckExistenceReceivedExpiredByHash(network, alpha, sel.Get("alice"), "alice.id1", h, false)
}

func TestHTLCNoCrossClaimTwoNetworks(network *integration.Infrastructure, sel *token2.ReplicaSelector) {
	// give some time to the nodes to get the public parameters
	time.Sleep(10 * time.Second)

	alpha := token.TMSID{Network: "alpha"}
	beta := token.TMSID{Network: "beta"}

	RegisterAuditor(network, token.WithTMSID(alpha))
	RegisterAuditor(network, token.WithTMSID(beta))

	IssueCashWithTMS(network, alpha, sel.Get("issuer"), "", "EUR", 30, "alice.id1")
	CheckBalanceAndHolding(network, sel.Get("alice"), "alice.id1", "EUR", 30, token.WithTMSID(alpha))

	IssueCashWithTMS(network, beta, sel.Get("issuer"), "", "USD", 30, "bob.id1")
	CheckBalanceAndHolding(network, sel.Get("bob"), "bob.id1", "USD", 30, token.WithTMSID(beta))

	aliceLockTxID, preImage, hash := HTLCLock(network, alpha, sel.Get("alice"), "alice.id1", "EUR", 10, "alice.id2", 30*time.Second, nil, 0)
	bobLockTxID, _, _ := HTLCLock(network, beta, sel.Get("bob"), "bob.id1", "USD", 10, "bob.id2", 30*time.Second, hash, 0)

	go func() { htlcClaim(network, alpha, sel.Get("alice"), "alice.id2", preImage) }()
	go func() { htlcClaim(network, beta, sel.Get("bob"), "bob.id2", preImage) }()
	scan(network, sel.Get("alice"), hash, crypto.SHA256, "", false, token.WithTMSID(alpha))
	scan(network, sel.Get("alice"), hash, crypto.SHA256, aliceLockTxID, false, token.WithTMSID(alpha))

	scan(network, sel.Get("bob"), hash, crypto.SHA256, "", false, token.WithTMSID(beta))
	scan(network, sel.Get("bob"), hash, crypto.SHA256, bobLockTxID, false, token.WithTMSID(beta))

	CheckBalanceWithLockedAndHolding(network, sel.Get("alice"), "alice.id1", "EUR", 20, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, sel.Get("alice"), "alice.id2", "EUR", 10, 0, 0, -1, token.WithTMSID(alpha))
	CheckBalanceWithLockedAndHolding(network, sel.Get("bob"), "bob.id1", "USD", 20, 0, 0, -1, token.WithTMSID(beta))
	CheckBalanceWithLockedAndHolding(network, sel.Get("bob"), "bob.id2", "USD", 10, 0, 0, -1, token.WithTMSID(beta))

	txID := IssueCashWithTMS(network, alpha, sel.Get("issuer"), "", "EUR", 30, "alice.id1")

	scan(network, sel.Get("bob"), hash, crypto.SHA256, bobLockTxID, true, token.WithTMSID(beta))
	start := time.Now()
	scanWithError(network, sel.Get("alice"), hash, crypto.SHA256, txID, []string{"context done"}, true, token.WithTMSID(alpha))
	Expect(time.Since(start)).To(BeNumerically("<", time.Second*30), "scan should be canceled on last tx, before timeout")
	scanWithError(network, sel.Get("alice"), hash, crypto.SHA256, txID, []string{"timeout reached"}, false, token.WithTMSID(alpha))

	CheckPublicParams(network, token.TMSID{}, sel.All("alice", "bob")...)
	CheckPublicParams(network, alpha, sel.All("issuer", "auditor")...)
	CheckPublicParams(network, beta, sel.All("issuer", "auditor")...)
	CheckOwnerDB(network, token.TMSID{}, nil, sel.All("auditor", "alice", "bob")...)
	CheckOwnerDB(network, alpha, nil, sel.All("issuer")...)
	CheckOwnerDB(network, beta, nil, sel.All("issuer")...)
	CheckAuditorDB(network, alpha, sel.Get("auditor"), "", nil)
	CheckAuditorDB(network, beta, sel.Get("auditor"), "", nil)
	PruneInvalidUnspentTokens(network, alpha, sel.All("issuer", "auditor")...)
	PruneInvalidUnspentTokens(network, beta, sel.All("issuer", "auditor")...)

	PruneInvalidUnspentTokens(network, alpha, sel.All("alice")...)
	aIDs := ListVaultUnspentTokens(network, alpha, sel.Get("alice"))
	CheckIfExistsInVault(network, alpha, sel.Get("auditor"), aIDs)

	PruneInvalidUnspentTokens(network, beta, sel.All("bob")...)
	bIDs := ListVaultUnspentTokens(network, beta, sel.Get("bob"))
	CheckIfExistsInVault(network, beta, sel.Get("auditor"), bIDs)
}

func TestFastExchange(network *integration.Infrastructure, sel *token2.ReplicaSelector) {
	// give some time to the nodes to get the public parameters
	time.Sleep(10 * time.Second)

	alpha := token.TMSID{Network: "alpha"}
	beta := token.TMSID{Network: "beta"}

	RegisterAuditor(network, token.WithTMSID(alpha))
	RegisterAuditor(network, token.WithTMSID(beta))

	IssueCashWithTMS(network, alpha, sel.Get("issuer"), "", "EUR", 30, "alice")
	CheckBalance(network, sel.Get("alice"), "", "EUR", 30, token.WithTMSID(alpha))

	IssueCashWithTMS(network, beta, sel.Get("issuer"), "", "USD", 30, "bob")
	CheckBalance(network, sel.Get("bob"), "", "USD", 30, token.WithTMSID(beta))

	fastExchange(network, sel.Get("alice"), "bob", alpha, "EUR", 10, beta, "USD", 10, 1*time.Hour)

	CheckBalance(network, sel.Get("alice"), "", "EUR", 20, token.WithTMSID(alpha))
	CheckBalance(network, sel.Get("bob"), "", "EUR", 10, token.WithTMSID(alpha))

	CheckBalance(network, sel.Get("alice"), "", "USD", 10, token.WithTMSID(beta))
	CheckBalance(network, sel.Get("bob"), "", "USD", 20, token.WithTMSID(beta))
}
