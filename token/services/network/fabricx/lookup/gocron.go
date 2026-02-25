/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package lookup

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabricx/core/vault/queryservice"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/lookup"
)

var logger = logging.MustGetLogger()

// QueryService models the FabricX query service needed by the NSListenerManager
//
//go:generate counterfeiter -o mock/qs.go -fake-name QueryService . QueryService
type QueryService = queryservice.QueryService

// QueryServiceProvider is an alias for queryservice.Provider
//
//go:generate counterfeiter -o mock/qps.go -fake-name QueryServiceProvider . QueryServiceProvider
type QueryServiceProvider = queryservice.Provider

// Listener is an alias for lookup.Listener
//
//go:generate counterfeiter -o mock/ll.go -fake-name Listener . Listener
type Listener = lookup.Listener

// CronListenerManager is a lookup listener manager that uses gocron for task scheduling.
type CronListenerManager struct {
	queryService QueryService
	config       ConfigGetter
	scheduler    gocron.Scheduler

	mu   sync.RWMutex
	jobs map[Listener]gocron.Job
}

// NewCronListenerManager creates a new CronListenerManager.
// It initializes and starts the gocron scheduler.
func NewCronListenerManager(qs QueryService, config ConfigGetter) (*CronListenerManager, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, errors.Wrapf(err, "failed creating scheduler")
	}
	s.Start()

	return &CronListenerManager{
		queryService: qs,
		config:       config,
		scheduler:    s,
		jobs:         make(map[Listener]gocron.Job),
	}, nil
}

// PermanentLookupListenerSupported returns true if permanent lookup listeners are supported.
func (n *CronListenerManager) PermanentLookupListenerSupported() bool {
	return true
}

// AddPermanentLookupListener adds a permanent lookup listener for the given key.
// It schedules a recurring job that checks for state changes.
func (n *CronListenerManager) AddPermanentLookupListener(namespace string, key string, listener Listener) error {
	logger.Infof("AddPermanentLookupListener [%s:%s]", namespace, key)

	j, err := n.scheduler.NewJob(
		gocron.DurationJob(n.config.PermanentInterval()),
		gocron.NewTask(NewPermanentJob(namespace, key, listener, n.queryService)),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	if err != nil {
		return errors.Wrapf(err, "failed scheduling permanent lookup job")
	}

	n.mu.Lock()
	n.jobs[listener] = j
	n.mu.Unlock()

	return nil
}

// AddLookupListener adds a lookup listener for the given key.
// It schedules a recurring job that checks for the key until it is found or the deadline is reached.
// The job is removed once it is invoked.
func (n *CronListenerManager) AddLookupListener(namespace string, key string, listener lookup.Listener) error {
	logger.Infof("AddLookupListener [%s:%s]", namespace, key)

	deadline := time.Now().Add(n.config.OnceDeadline())

	// Use a pointer to the job so the task can remove it
	var j gocron.Job
	var err error

	task := gocron.NewTask(func() {
		logger.Infof("[KeyCheck] check for key [%s:%s]", namespace, key)
		v, err := n.queryService.GetState(namespace, key)
		if err == nil && v != nil && len(v.Raw) != 0 {
			logger.Infof("[KeyCheck] key [%s:%s] found, notify listener", namespace, key)
			listener.OnStatus(context.Background(), key, v.Raw)

			// Stop the job, no error is expected here
			_ = n.RemoveLookupListener(key, listener)

			return
		}

		if time.Now().After(deadline) {
			logger.Infof("[KeyCheck] key [%s:%s] not found, deadline reached", namespace, key)
			listener.OnError(context.Background(), key, errors.Errorf("key [%s:%s] not found", namespace, key))

			// Stop the job, no error is expected here
			_ = n.RemoveLookupListener(key, listener)

			return
		}
	})

	j, err = n.scheduler.NewJob(
		gocron.DurationJob(n.config.OnceInterval()),
		task,
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	if err != nil {
		return errors.Wrapf(err, "failed scheduling lookup job")
	}

	n.mu.Lock()
	n.jobs[listener] = j
	n.mu.Unlock()

	return nil
}

// RemoveLookupListener removes a lookup listener for the given key and stops its associated job.
func (n *CronListenerManager) RemoveLookupListener(id string, listener Listener) error {
	logger.Infof("RemoveLookupListener [%s]", id)
	n.mu.Lock()
	defer n.mu.Unlock()

	if j, ok := n.jobs[listener]; ok {
		err := n.scheduler.RemoveJob(j.ID())
		if err != nil {
			logger.Warningf("failed removing job [%s]: %s", j.ID(), err)
		}
		delete(n.jobs, listener)
	}

	return nil
}

// Stop stops the scheduler and waits for jobs to finish.
func (n *CronListenerManager) Stop() error {
	return n.scheduler.Shutdown()
}

// CronNSListenerManagerProvider is a provider for CronListenerManager.
type CronNSListenerManagerProvider struct {
	QueryServiceProvider QueryServiceProvider
	config               ConfigGetter
}

// NewCronNSListenerManagerProvider creates a new CronNSListenerManagerProvider.
func NewCronNSListenerManagerProvider(queryServiceProvider QueryServiceProvider, config ConfigGetter) lookup.ListenerManagerProvider {
	return &CronNSListenerManagerProvider{
		QueryServiceProvider: queryServiceProvider,
		config:               config,
	}
}

// NewManager creates a new ListenerManager for the given network and channel.
func (n *CronNSListenerManagerProvider) NewManager(network, channel string) (lookup.ListenerManager, error) {
	qs, err := n.QueryServiceProvider.Get(network, channel)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting query service")
	}

	return NewCronListenerManager(qs, n.config)
}

type PermanentJob struct {
	namespace string
	key       string
	listener  Listener

	queryService QueryService
	lastValue    []byte
}

func NewPermanentJob(namespace string, key string, listener Listener, queryService QueryService) *PermanentJob {
	return &PermanentJob{namespace: namespace, key: key, listener: listener, queryService: queryService}
}

func (j *PermanentJob) Run() {
	logger.Infof("[PermanentKeyCheck] check for key [%s:%s]", j.namespace, j.key)
	v, err := j.queryService.GetState(j.namespace, j.key)
	if err == nil && v != nil && len(v.Raw) != 0 {
		if !bytes.Equal(j.lastValue, v.Raw) {
			logger.Infof("[PermanentKeyCheck] key [%s:%s] found with new value, notify listener", j.namespace, j.key)
			j.listener.OnStatus(context.Background(), j.key, v.Raw)
			j.lastValue = v.Raw
		}
	}
}
