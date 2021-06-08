/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package bn256

import (
	"sync"
	"testing"

	"github.com/consensys/gurvy/bn256"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
)

func TestG1MUM(t *testing.T) {
	g := G1Gen()
	bytes, err := g.MarshalJSON()
	assert.NoError(err)
	h := &G1{}
	h.UnmarshalJSON(bytes)
	assert.Equal(g.Equals(h), true)
	assert.Equal(g, h)

}

func TestG1Add(t *testing.T) {
	rng, _ := GetRand()
	r := RandModOrder(rng)
	s := RandModOrder(rng)
	a := G1Gen()
	c := &G1{}
	c.Copy(a)
	g := a.Mul(r)
	assert.Equal(a, c)
	h := a.Mul(s)
	assert.Equal(a, c)

	assert.Equal(g.Add(h), g)
	assert.Equal(g, a.Mul(ModAdd(r, s, Order))) // fixme this should be equal
	assert.Equal(g, a.Mul(r.Plus(s)))
	assert.Equal((*bn256.G1Affine)(g).IsInSubGroup(), true)
	assert.Equal((*bn256.G1Affine)(g.Mul(Order)).IsInfinity(), true) // fixme this should be true
}

func TestG2MUM(t *testing.T) {
	g := G2Gen()

	bytes, err := g.MarshalJSON()
	assert.NoError(err)
	h := &G2{}
	h.UnmarshalJSON(bytes)
	assert.Equal(g, h)
}

func TestG2Add(t *testing.T) {
	rng, _ := GetRand()
	r := RandModOrder(rng)
	s := RandModOrder(rng)
	a := G2Gen()
	c := &G2{}
	c.Copy(a)
	g := a.Mul(r)
	assert.Equal(a, c)
	h := a.Mul(s)
	assert.Equal(a, c)

	assert.Equal(g.Add(h), g)
	assert.Equal(g, a.Mul(ModAdd(r, s, Order))) // fixme this should be equal
	assert.Equal(g, a.Mul(r.Plus(s)))
	assert.Equal((*bn256.G2Affine)(g).IsInSubGroup(), true)
	assert.Equal((*bn256.G2Affine)(g.Mul(Order)).IsInfinity(), true) // fixme this should be true

}

func TestZrMUM(t *testing.T) {
	g := NewZrInt(15)
	bytes, err := g.MarshalJSON()
	assert.NoError(err)
	h := &Zr{}
	h.UnmarshalJSON(bytes)
	assert.Equal(h.IsZero(), false)
	assert.Equal(h.Cmp(NewZrInt(13)) != 0, true)
	assert.Equal(g, h)
}

func TestDataRaceG1(t *testing.T) {
	g := G1Gen()
	bytes := g.Bytes()

	wg := &sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go loadG1(wg, bytes)
	}
	wg.Wait()
}

func loadG1(wg *sync.WaitGroup, bytes []byte) {
	for i := 0; i < 100; i++ {
		g1, err := NewG1FromBytes(bytes)
		assert.NoError(err)
		g1.Bytes()
	}
	wg.Done()
}
