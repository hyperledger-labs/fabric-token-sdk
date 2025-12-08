package bls12381ext

import (
	"sync"

	bls12381 "github.com/consensys/gnark-crypto/ecc/bls12-381"
)

// G1Jacs is a shared *bls12381.G1Jac{} memory pool
var G1Jacs g1JacPool

var _g1JacPool = sync.Pool{
	New: func() interface{} {
		return new(bls12381.G1Jac)
	},
}

type g1JacPool struct{}

func (g1JacPool) Get() *bls12381.G1Jac {
	return _g1JacPool.Get().(*bls12381.G1Jac)
}

func (g1JacPool) Put(v *bls12381.G1Jac) {
	if v == nil {
		panic("g1JacPool called with nil value")
	}
	_g1JacPool.Put(v)
}
