/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rng

import (
	"crypto/rand"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/chacha20"
)

// SecureRNG is a high-throughput, concurrency-safe random number generator
// backed by the ChaCha20 stream cipher.
//
// SECURITY GUARANTEES:
//   - Thread Safety: Safe for concurrent use by multiple goroutines.
//   - Forward Secrecy: Periodic reseeding ensures future key compromises do not reveal past outputs.
//   - Backward Secrecy: NOT guaranteed (compromise reveals output since last reseed).
//
// WARNING: VM SNAPSHOTS & FORKS
// This RNG lives in userspace. If the process is forked or the VM snapshotted,
// the state will be duplicated, leading to identical random streams.
type SecureRNG struct {
	Pool sync.Pool

	// ReseedInterval forces a re-key after a set duration.
	ReseedInterval time.Duration

	// ReseedVolume forces a re-key after generating a set volume of data.
	ReseedVolume uint64
}

// CipherWrapper holds the state for a single ChaCha20 stream.
type CipherWrapper struct {
	cipher     *chacha20.Cipher
	bytesRead  atomic.Uint64 // Atomic to prevent data races on counter
	createTime time.Time
}

// NewSecureRNG initializes a new RNG with safe defaults (1 Minute / 32GB).
func NewSecureRNG() *SecureRNG {
	return NewSecureRNGWith(1*time.Minute, 32*1024*1024*1024)
}

// NewSecureRNGWith allows custom reseed parameters.
func NewSecureRNGWith(interval time.Duration, volume uint64) *SecureRNG {
	return &SecureRNG{
		ReseedInterval: interval,
		ReseedVolume:   volume,
		Pool: sync.Pool{
			New: func() any {
				// Return nil on error; handled in Read
				c, err := newCipherWrapper()
				if err != nil {
					return nil
				}

				return c
			},
		},
	}
}

func newCipherWrapper() (*CipherWrapper, error) {
	var key [32]byte
	if _, err := rand.Reader.Read(key[:]); err != nil {
		return nil, fmt.Errorf("RNG seed failed: %w", err)
	}

	// We use a zero nonce because the Key is fresh and unique.
	// RFC 7539 permits this as long as the (Key, Nonce) pair is unique.
	nonce := make([]byte, chacha20.NonceSize)
	c, err := chacha20.NewUnauthenticatedCipher(key[:], nonce)
	if err != nil {
		return nil, err
	}

	return &CipherWrapper{
		cipher:     c,
		createTime: time.Now(),
	}, nil
}

// Read fills p with random bytes. It handles pool retrieval, reseeding, and cleanup.
func (r *SecureRNG) Read(p []byte) (int, error) {
	item := r.Pool.Get()

	var w *CipherWrapper
	var err error

	if item == nil {
		// Pool failed to allocate or was empty and New() failed
		w, err = newCipherWrapper()
		if err != nil {
			return 0, err
		}
	} else {
		w = item.(*CipherWrapper)
	}

	// Check reseed conditions (Time or Volume)
	if time.Since(w.createTime) > r.ReseedInterval || w.bytesRead.Load() > r.ReseedVolume {
		// Discard old wrapper. We generate a fresh one.
		w.cipher = nil // Help GC

		w, err = newCipherWrapper()
		if err != nil {
			// CRITICAL: Do not return the old 'w' to the pool.
			// It is expired and we failed to replace it. Drop it.
			return 0, err
		}
	}

	// Zero the buffer before XORing (Standard ChaCha20 practice)
	clear(p)

	w.cipher.XORKeyStream(p, p)
	w.bytesRead.Add(uint64(len(p)))

	r.Pool.Put(w)

	return len(p), nil
}
