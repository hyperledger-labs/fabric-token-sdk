/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interactive

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/disabled"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Additional fakes used only in this file
// ---------------------------------------------------------------------------

// errorQueryEngine returns an error from UnspentTokensIterator.
type errorQueryEngine struct {
	err error
}

func (e *errorQueryEngine) UnspentTokensIterator(_ context.Context) (*token2.UnspentTokensIterator, error) {
	return nil, e.err
}

// alwaysFailViewManager returns the same error on every InitiateView call.
type alwaysFailViewManager struct {
	mu   sync.Mutex
	err  error
	seen int
}

func (f *alwaysFailViewManager) InitiateView(_ view.View) (interface{}, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seen++

	return nil, f.err
}

func (f *alwaysFailViewManager) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.seen
}

// wrongTypeViewManager succeeds but returns a string instead of map[*token.ID][]byte.
type wrongTypeViewManager struct{}

func (w *wrongTypeViewManager) InitiateView(_ view.View) (interface{}, error) {
	return "not-a-map", nil
}

// failingCertificationStorage delegates Exists to the inner fake but always
// fails on Store.
type failingCertificationStorage struct {
	*fakeCertificationStorage
	storeErr error
}

func (f *failingCertificationStorage) Store(_ context.Context, _ map[*token.ID][]byte) error {
	return f.storeErr
}

// fakeSubscriber records which topics were subscribed to.
type fakeSubscriber struct {
	mu     sync.Mutex
	topics []string
}

func (f *fakeSubscriber) Subscribe(topic string, _ events.Listener) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.topics = append(f.topics, topic)
}

func (f *fakeSubscriber) Unsubscribe(_ string, _ events.Listener) {}

// sliceUnspentIterator is a driver.UnspentTokensIterator backed by a slice.
type sliceUnspentIterator struct {
	items []*token.UnspentToken
	pos   int
}

func (s *sliceUnspentIterator) Next() (*token.UnspentToken, error) {
	if s.pos >= len(s.items) {
		return nil, nil
	}
	item := s.items[s.pos]
	s.pos++

	return item, nil
}

func (s *sliceUnspentIterator) Close() {}

// populatedQueryEngine returns an iterator over a fixed set of unspent tokens.
type populatedQueryEngine struct {
	items []*token.UnspentToken
}

func (p *populatedQueryEngine) UnspentTokensIterator(_ context.Context) (*token2.UnspentTokensIterator, error) {
	return &token2.UnspentTokensIterator{UnspentTokensIterator: &sliceUnspentIterator{items: p.items}}, nil
}

// unknownTopicEvent carries a valid TokenMessage but an unregistered topic so
// OnReceive hits the "not subscribed" early-return path.
type unknownTopicEvent struct{}

func (e *unknownTopicEvent) Topic() string { return "unregistered-topic" }
func (e *unknownTopicEvent) Message() interface{} {
	return tokens.TokenMessage{TxID: "tx-unk", Index: 0}
}

// wrongMsgTypeEvent has the right topic but a payload that cannot be cast to
// tokens.TokenMessage, exercising the first early-return path in OnReceive.
type wrongMsgTypeEvent struct{}

func (e *wrongMsgTypeEvent) Topic() string        { return tokens.AddToken }
func (e *wrongMsgTypeEvent) Message() interface{} { return 42 }

// ---------------------------------------------------------------------------
// NewCertificationClient — notifier path
// ---------------------------------------------------------------------------

// TestNewCertificationClient_WithNotifier verifies that when a Subscriber is
// provided the client registers itself for the expected topics.
func TestNewCertificationClient_WithNotifier(t *testing.T) {
	sub := &fakeSubscriber{}
	storage := newFakeCertificationStorage()

	cc := NewCertificationClient(
		context.Background(),
		"test-network",
		"ch", "ns",
		&fakeQueryEngine{},
		storage,
		&fakeViewManager{},
		[]view.Identity{view.Identity([]byte("c"))},
		sub,
		DefaultMaxAttempts, DefaultWaitTime,
		DefaultBatchSize, DefaultBufferSize,
		DefaultFlushInterval, DefaultWorkers,
		DefaultResponseTimeout,
		&disabled.Provider{},
	)

	assert.NotNil(t, cc)
	sub.mu.Lock()
	defer sub.mu.Unlock()
	assert.Contains(t, sub.topics, tokens.AddToken, "expected AddToken topic to be subscribed")
}

// ---------------------------------------------------------------------------
// RequestCertification
// ---------------------------------------------------------------------------

// TestRequestCertification_AllAlreadyCertified verifies that when every
// supplied token is already certified, the ViewManager is never called.
func TestRequestCertification_AllAlreadyCertified(t *testing.T) {
	vm := &fakeViewManager{}
	storage := newFakeCertificationStorage()

	id1 := &token.ID{TxId: "already-1", Index: 0}
	id2 := &token.ID{TxId: "already-2", Index: 0}
	storage.data["already-1"] = []byte("cert-1")
	storage.data["already-2"] = []byte("cert-2")

	cc := newTestClient(t, vm, storage, 10, 1*time.Second, 1)

	err := cc.RequestCertification(context.Background(), id1, id2)
	require.NoError(t, err)
	assert.Equal(t, 0, vm.callCount(), "ViewManager must not be called when all tokens are already certified")
}

// TestRequestCertification_PartialAlreadyCertified verifies that only the
// uncertified subset is passed to the ViewManager.
func TestRequestCertification_PartialAlreadyCertified(t *testing.T) {
	vm := &fakeViewManager{}
	storage := newFakeCertificationStorage()

	idCertified := &token.ID{TxId: "already", Index: 0}
	storage.data["already"] = []byte("cert-already")

	idNew := &token.ID{TxId: "new-tx", Index: 0}

	cc := newTestClient(t, vm, storage, 10, 1*time.Second, 1)

	err := cc.RequestCertification(context.Background(), idCertified, idNew)
	require.NoError(t, err)
	assert.Equal(t, 1, vm.callCount(), "ViewManager should be called once for the uncertified token")
	assert.True(t, storage.Exists(context.Background(), idNew), "new token should be certified after the call")
	assert.True(t, storage.Exists(context.Background(), idCertified), "pre-certified token should still be present")
}

// TestRequestCertification_RetriesExhausted verifies that when the
// ViewManager keeps failing, RequestCertification returns an error after
// exhausting maxAttempts.
func TestRequestCertification_RetriesExhausted(t *testing.T) {
	certErr := errors.New("certifier down")
	vm := &alwaysFailViewManager{err: certErr}
	storage := newFakeCertificationStorage()

	cc := NewCertificationClient(
		context.Background(),
		"test-network",
		"ch", "ns",
		&fakeQueryEngine{},
		storage,
		vm,
		[]view.Identity{view.Identity([]byte("c"))},
		nil,
		2,                  // maxAttempts=2
		1*time.Millisecond, // short retry gap
		10, 1000, 1*time.Second, 1,
		DefaultResponseTimeout,
		&disabled.Provider{},
	)

	id := &token.ID{TxId: "retry-tx", Index: 0}
	err := cc.RequestCertification(context.Background(), id)

	require.Error(t, err)
	require.ErrorIs(t, err, certErr)
	assert.Equal(t, 2, vm.callCount(), "ViewManager should be called exactly maxAttempts times")
}

// TestRequestCertification_ContextCancelled verifies that cancelling the
// context during a retry wait causes RequestCertification to return
// context.DeadlineExceeded promptly instead of waiting for the full retry gap.
func TestRequestCertification_ContextCancelled(t *testing.T) {
	vm := &alwaysFailViewManager{err: errors.New("always fails")}
	storage := newFakeCertificationStorage()

	cc := NewCertificationClient(
		context.Background(),
		"test-network",
		"ch", "ns",
		&fakeQueryEngine{},
		storage,
		vm,
		[]view.Identity{view.Identity([]byte("c"))},
		nil,
		10,             // many attempts — context must cancel first
		10*time.Second, // long wait between retries
		10, 1000, 1*time.Second, 1,
		DefaultResponseTimeout,
		&disabled.Provider{},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	id := &token.ID{TxId: "ctx-cancel-tx", Index: 0}
	err := cc.RequestCertification(ctx, id)

	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

// TestRequestCertification_InvalidReturnType verifies the error path when the
// ViewManager returns a type that cannot be cast to map[*token.ID][]byte.
func TestRequestCertification_InvalidReturnType(t *testing.T) {
	storage := newFakeCertificationStorage()

	cc := NewCertificationClient(
		context.Background(),
		"test-network",
		"ch", "ns",
		&fakeQueryEngine{},
		storage,
		&wrongTypeViewManager{},
		[]view.Identity{view.Identity([]byte("c"))},
		nil,
		1, 1*time.Millisecond,
		10, 1000, 1*time.Second, 1,
		DefaultResponseTimeout,
		&disabled.Provider{},
	)

	id := &token.ID{TxId: "bad-type-tx", Index: 0}
	err := cc.RequestCertification(context.Background(), id)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type")
}

// TestRequestCertification_StoreError verifies that an error from
// CertificationStorage.Store is propagated back to the caller.
func TestRequestCertification_StoreError(t *testing.T) {
	vm := &fakeViewManager{}
	storeErr := errors.New("storage is unavailable")
	storage := &failingCertificationStorage{
		fakeCertificationStorage: newFakeCertificationStorage(),
		storeErr:                 storeErr,
	}

	cc := NewCertificationClient(
		context.Background(),
		"test-network",
		"ch", "ns",
		&fakeQueryEngine{},
		storage,
		vm,
		[]view.Identity{view.Identity([]byte("c"))},
		nil,
		1, 1*time.Millisecond,
		10, 1000, 1*time.Second, 1,
		DefaultResponseTimeout,
		&disabled.Provider{},
	)

	id := &token.ID{TxId: "store-err-tx", Index: 0}
	err := cc.RequestCertification(context.Background(), id)

	require.Error(t, err)
	require.ErrorIs(t, err, storeErr)
}

// TestRequestCertification_RetryThenSuccess verifies that a transient error on
// the first attempt is retried and the subsequent success is accepted.
func TestRequestCertification_RetryThenSuccess(t *testing.T) {
	transientErr := errors.New("transient error")
	// fakeViewManager.failErr is cleared after the first failure, so the
	// second attempt succeeds automatically.
	vm := &fakeViewManager{failErr: transientErr}
	storage := newFakeCertificationStorage()

	cc := NewCertificationClient(
		context.Background(),
		"test-network",
		"ch", "ns",
		&fakeQueryEngine{},
		storage,
		vm,
		[]view.Identity{view.Identity([]byte("c"))},
		nil,
		3, // 3 attempts available
		1*time.Millisecond,
		10, 1000, 1*time.Second, 1,
		DefaultResponseTimeout,
		&disabled.Provider{},
	)

	id := &token.ID{TxId: "retry-ok-tx", Index: 0}
	err := cc.RequestCertification(context.Background(), id)

	require.NoError(t, err)
	assert.Equal(t, 2, vm.callCount(), "ViewManager called twice: once failed, once succeeded")
	assert.True(t, storage.Exists(context.Background(), id), "token should be certified after successful retry")
}

// ---------------------------------------------------------------------------
// OnReceive edge cases
// ---------------------------------------------------------------------------

// TestOnReceive_WrongMessageType verifies that an event whose Message() cannot
// be cast to tokens.TokenMessage is silently dropped (no panic, no enqueue).
func TestOnReceive_WrongMessageType(t *testing.T) {
	vm := &fakeViewManager{}
	storage := newFakeCertificationStorage()
	cc := newTestClient(t, vm, storage, 10, 1*time.Second, 1)

	assert.NotPanics(t, func() {
		cc.OnReceive(&wrongMsgTypeEvent{})
	})
	assert.Empty(t, cc.tokens, "wrong-type event must not enqueue a token")
}

// TestOnReceive_UnknownTopic verifies that an event with an unregistered topic
// is silently dropped even when the message type is correct.
func TestOnReceive_UnknownTopic(t *testing.T) {
	vm := &fakeViewManager{}
	storage := newFakeCertificationStorage()
	cc := newTestClient(t, vm, storage, 10, 1*time.Second, 1)

	assert.NotPanics(t, func() {
		cc.OnReceive(&unknownTopicEvent{})
	})
	assert.Empty(t, cc.tokens, "unknown-topic event must not enqueue a token")
}

// ---------------------------------------------------------------------------
// Scan
// ---------------------------------------------------------------------------

// TestScan_EmptyVault verifies that Scan on an empty vault succeeds without
// calling the ViewManager.
func TestScan_EmptyVault(t *testing.T) {
	vm := &fakeViewManager{}
	storage := newFakeCertificationStorage()

	cc := newTestClient(t, vm, storage, 10, 1*time.Second, 1)

	err := cc.Scan()
	require.NoError(t, err)
	assert.Equal(t, 0, vm.callCount(), "empty vault should not trigger any certification requests")
}

// TestScan_UncertifiedTokens verifies that Scan picks up tokens from the vault
// and requests their certification.
func TestScan_UncertifiedTokens(t *testing.T) {
	vm := &fakeViewManager{}
	storage := newFakeCertificationStorage()

	unspent := []*token.UnspentToken{
		{Id: token.ID{TxId: "vault-tx-1", Index: 0}},
		{Id: token.ID{TxId: "vault-tx-2", Index: 0}},
	}

	cc := NewCertificationClient(
		context.Background(),
		"test-network",
		"ch", "ns",
		&populatedQueryEngine{items: unspent},
		storage,
		vm,
		[]view.Identity{view.Identity([]byte("c"))},
		nil,
		2, 1*time.Millisecond,
		10, 1000, 1*time.Second, 1,
		DefaultResponseTimeout,
		&disabled.Provider{},
	)

	err := cc.Scan()
	require.NoError(t, err)

	for _, ut := range unspent {
		id := &ut.Id
		assert.True(t, storage.Exists(context.Background(), id), "token %s should be certified after Scan", ut.Id.TxId)
	}
}

// TestScan_AlreadyCertified verifies that tokens already present in storage
// are skipped — the ViewManager is not called for them.
func TestScan_AlreadyCertified(t *testing.T) {
	vm := &fakeViewManager{}
	storage := newFakeCertificationStorage()

	// Pre-certify the vault token.
	storage.data["pre-certified"] = []byte("existing-cert")

	unspent := []*token.UnspentToken{
		{Id: token.ID{TxId: "pre-certified", Index: 0}},
	}

	cc := NewCertificationClient(
		context.Background(),
		"test-network",
		"ch", "ns",
		&populatedQueryEngine{items: unspent},
		storage,
		vm,
		[]view.Identity{view.Identity([]byte("c"))},
		nil,
		2, 1*time.Millisecond,
		10, 1000, 1*time.Second, 1,
		DefaultResponseTimeout,
		&disabled.Provider{},
	)

	err := cc.Scan()
	require.NoError(t, err)
	assert.Equal(t, 0, vm.callCount(), "already-certified token should not trigger a ViewManager call")
}

// ---------------------------------------------------------------------------
// processBatch
// ---------------------------------------------------------------------------

// TestProcessBatch_EmptyBatch_TriggersVaultScan verifies that an empty batch
// causes processBatch to fall through to a vault scan, certifying any
// uncertified tokens it finds.
func TestProcessBatch_EmptyBatch_TriggersVaultScan(t *testing.T) {
	vm := &fakeViewManager{}
	storage := newFakeCertificationStorage()

	unspent := []*token.UnspentToken{
		{Id: token.ID{TxId: "vault-via-empty-batch", Index: 0}},
	}

	cc := NewCertificationClient(
		context.Background(),
		"test-network",
		"ch", "ns",
		&populatedQueryEngine{items: unspent},
		storage,
		vm,
		[]view.Identity{view.Identity([]byte("c"))},
		nil,
		2, 1*time.Millisecond,
		10, 1000, 1*time.Second, 1,
		DefaultResponseTimeout,
		&disabled.Provider{},
	)

	// Invoke directly — no Start() needed.
	cc.processBatch([]*token.ID{})

	id := &unspent[0].Id
	assert.True(t, storage.Exists(context.Background(), id),
		"vault token should be certified after empty-batch scan")
}

// TestProcessBatch_CertificationFailure_PushesBackToBuffer verifies that when
// the ViewManager fails, the tokens in the batch are pushed back into the
// tokens channel for a future retry rather than silently dropped.
func TestProcessBatch_CertificationFailure_PushesBackToBuffer(t *testing.T) {
	certErr := errors.New("certification service unavailable")
	vm := &alwaysFailViewManager{err: certErr}
	storage := newFakeCertificationStorage()

	cc := NewCertificationClient(
		context.Background(),
		"test-network",
		"ch", "ns",
		&fakeQueryEngine{},
		storage,
		vm,
		[]view.Identity{view.Identity([]byte("c"))},
		nil,
		1, // maxAttempts=1 — fail immediately, no retry loop
		1*time.Millisecond,
		10,
		100, // bufferSize=100, enough room for push-back
		1*time.Second, 1,
		DefaultResponseTimeout,
		&disabled.Provider{},
	)
	// Do NOT Start() — we call processBatch directly so the tokens channel
	// is not being consumed by the accumulator.

	id := &token.ID{TxId: "pushback-tx", Index: 0}
	cc.processBatch([]*token.ID{id})

	assert.Len(t, cc.tokens, 1, "failed token should be pushed back to the tokens channel")
}

// ---------------------------------------------------------------------------
// Driver construction
// ---------------------------------------------------------------------------

// TestNewDriver_Construction verifies that NewDriver initialises an empty
// client map and a nil CertificationService ready for lazy initialisation.
func TestNewDriver_Construction(t *testing.T) {
	mp := &disabled.Provider{}

	d := NewDriver(nil, nil, nil, &fakeViewManager{}, &ResponderRegistryMock{}, mp)

	assert.NotNil(t, d)
	assert.NotNil(t, d.CertificationClients, "CertificationClients map should be initialised")
	assert.Empty(t, d.CertificationClients, "no clients should exist before any TMS is registered")
	assert.Nil(t, d.CertificationService, "CertificationService should be nil until first use")
	assert.Equal(t, mp, d.MetricsProvider)
}

// ---------------------------------------------------------------------------
// Scan — error paths
// ---------------------------------------------------------------------------

// TestScan_QueryEngineError verifies that an error from UnspentTokensIterator
// is propagated back to the caller.
func TestScan_QueryEngineError(t *testing.T) {
	queryErr := errors.New("vault iterator failure")
	vm := &fakeViewManager{}
	storage := newFakeCertificationStorage()

	cc := NewCertificationClient(
		context.Background(),
		"net", "ch", "ns",
		&errorQueryEngine{err: queryErr},
		storage,
		vm,
		[]view.Identity{view.Identity([]byte("c"))},
		nil,
		1, 1*time.Millisecond,
		10, 1000, 1*time.Second, 1,
		DefaultResponseTimeout,
		&disabled.Provider{},
	)

	err := cc.Scan()
	require.Error(t, err)
	require.ErrorIs(t, err, queryErr)
}

// TestScan_RequestCertificationError verifies that when RequestCertification
// fails for uncertified vault tokens, Scan propagates the error.
func TestScan_RequestCertificationError(t *testing.T) {
	certErr := errors.New("certification service down")
	vm := &alwaysFailViewManager{err: certErr}
	storage := newFakeCertificationStorage()

	unspent := []*token.UnspentToken{
		{Id: token.ID{TxId: "scan-fail-tx", Index: 0}},
	}

	cc := NewCertificationClient(
		context.Background(),
		"net", "ch", "ns",
		&populatedQueryEngine{items: unspent},
		storage,
		vm,
		[]view.Identity{view.Identity([]byte("c"))},
		nil,
		1, 1*time.Millisecond,
		10, 1000, 1*time.Second, 1,
		DefaultResponseTimeout,
		&disabled.Provider{},
	)

	err := cc.Scan()
	require.Error(t, err)
	require.ErrorIs(t, err, certErr)
}

// ---------------------------------------------------------------------------
// processBatch — remaining edge cases
// ---------------------------------------------------------------------------

// TestProcessBatch_EmptyBatch_ScanError verifies that when the vault scan
// triggered by an empty batch returns an error, processBatch logs and returns
// without panicking.
func TestProcessBatch_EmptyBatch_ScanError(t *testing.T) {
	vm := &fakeViewManager{}
	storage := newFakeCertificationStorage()

	cc := NewCertificationClient(
		context.Background(),
		"net", "ch", "ns",
		&errorQueryEngine{err: errors.New("vault error")},
		storage,
		vm,
		[]view.Identity{view.Identity([]byte("c"))},
		nil,
		1, 1*time.Millisecond,
		10, 1000, 1*time.Second, 1,
		DefaultResponseTimeout,
		&disabled.Provider{},
	)

	require.NotPanics(t, func() {
		cc.processBatch([]*token.ID{})
	})
}

// TestProcessBatch_FailureBufferFull_DropsTokens verifies that when the
// push-back buffer is already full after a certification failure, the token
// is dropped rather than causing a deadlock.
func TestProcessBatch_FailureBufferFull_DropsTokens(t *testing.T) {
	certErr := errors.New("certifier unavailable")
	vm := &alwaysFailViewManager{err: certErr}
	storage := newFakeCertificationStorage()

	cc := NewCertificationClient(
		context.Background(),
		"net", "ch", "ns",
		&fakeQueryEngine{},
		storage,
		vm,
		[]view.Identity{view.Identity([]byte("c"))},
		nil,
		1, 1*time.Millisecond,
		1,
		1, // bufferSize=1 — filled below so push-back overflows
		1*time.Second, 1,
		DefaultResponseTimeout,
		&disabled.Provider{},
	)

	// Fill the buffer to capacity so push-back hits the default (drop) branch.
	cc.tokens <- &token.ID{TxId: "pre-fill", Index: 0}

	done := make(chan struct{})
	go func() {
		cc.processBatch([]*token.ID{{TxId: "fail-tx", Index: 0}})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("processBatch deadlocked when buffer was full after failure")
	}
}
