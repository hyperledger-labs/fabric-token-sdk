/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mailman

import (
	"fmt"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/testutils"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	NumClients = 32
	NumJobs    = 10000
	NumTokens  = 1000
)

func TestMail(t *testing.T) {

	qs := setupKVS()

	mailman := NewMailman()
	mailman.Start()

	// let's add all our tokens to mailman
	// we do this by sending "updates" instead of using the NewMailman constructor
	prePopulate(mailman, qs)

	iter := &UnspentTokenIterator{
		qs:      qs,
		mailman: mailman,
	}

	done := make(chan bool, NumClients)
	jobs := make(chan int, NumJobs)
	// create clients
	for w := 0; w < NumClients; w++ {
		go userRunner(jobs, iter, done)
	}

	for j := 1; j <= NumJobs; j++ {
		jobs <- j
	}
	close(jobs)

	for w := 0; w < NumClients; w++ {
		<-done
	}

	mailman.Stop()

	fmt.Printf("done\n")
}

func setupKVS() *testutils.MockQueryService {
	wallet := &token2.Owner{Raw: []byte("wallet0")}

	qs := testutils.NewMockQueryService()
	for i := 0; i < NumTokens; i++ {
		t := &token2.UnspentToken{
			Id:       &token2.ID{TxId: fmt.Sprintf("%d", i), Index: 0},
			Owner:    wallet,
			Type:     testutils.TokenType,
			Quantity: "1",
		}

		k := fmt.Sprintf("etoken.%s.%s.%s.%d", string(wallet.Raw), testutils.TokenType, t.Id.TxId, t.Id.Index)
		qs.Add(k, t)
	}
	qs.WarmupCache(string(wallet.Raw), testutils.TokenType)

	return qs
}

func userRunner(jobs <-chan int, iter *UnspentTokenIterator, done chan<- bool) {
	for range jobs {
		err := retry(testutils.SelectorNumRetries, testutils.LockSleepTimeout, func() error {
			t, err := iter.Next()
			if err != nil {
				return err
			}

			// if we got a token ... do something with the tokenID and then release again ....
			if t != nil {
				time.Sleep(7 * time.Millisecond)
				iter.mailman.Update(update{
					op:      Unlock,
					tokenID: *t.Id,
				})
			}
			return nil
		})
		if err != nil {
			fmt.Printf("got error: %v\n", err)
			continue
		}
	}
	done <- true
}

func prePopulate(mm *Mailman, qs *testutils.MockQueryService) {
	var updates []update
	it, _ := qs.UnspentTokensIterator()
	for {
		ut, err := it.Next()
		if err != nil || ut == nil {
			break
		}
		updates = append(updates, update{op: Add, tokenID: *ut.Id})
	}
	it.Close()
	mm.Update(updates...)
}

func retry(attempts int, sleep time.Duration, f func() error) (err error) {
	for i := 0; i < attempts; i++ {
		if i > 0 {
			time.Sleep(sleep)
			sleep *= 2
		}

		err = f()
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("no luck after %d attempts: last error: %v", attempts, err)
}
