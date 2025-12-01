/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package testdata

//go:generate idemixgen ca-keygen --output ./bls12_381_bbs/ca --curve BLS12_381_BBS --aries
//go:generate idemixgen signerconfig --ca-input ./bls12_381_bbs/ca --output ./bls12_381_bbs/idemix --admin -u example.com -e alice -r 150 --curve BLS12_381_BBS --aries

//go:generate idemixgen ca-keygen --output ./bn254/ca --curve BN254
//go:generate idemixgen signerconfig --ca-input ./bn254/ca --output ./bn254/idemix --admin -u example.com -e alice -r 150 --curve BN254
