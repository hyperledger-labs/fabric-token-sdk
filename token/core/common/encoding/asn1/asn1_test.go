/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package asn1

import (
	"encoding/asn1"
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type MathContainer struct {
	Zr      *math.Zr
	G1      *math.G1
	G2      *math.G2
	ZrArray []*math.Zr
	G1Array []*math.G1
}

func NewRandomMathContainer(curve *math.Curve) (*MathContainer, error) {
	rand, err := curve.Rand()
	if err != nil {
		return nil, err
	}
	return &MathContainer{
		Zr: curve.NewRandomZr(rand),
		G1: curve.NewG1(),
		G2: curve.NewG2(),
		ZrArray: []*math.Zr{
			curve.NewRandomZr(rand),
			curve.NewRandomZr(rand),
		},
		G1Array: []*math.G1{
			curve.NewG1(),
			curve.NewG1(),
		},
	}, nil
}

func (a *MathContainer) Serialize() ([]byte, error) {
	zrArray, err := NewElementArray(a.ZrArray)
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize ZrArray")
	}
	g1Array, err := NewElementArray(a.G1Array)
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize G1Array")
	}
	return MarshalMath(a.Zr, a.G1, a.G2, zrArray, g1Array)
}

func (a *MathContainer) Deserialize(bytes []byte) error {
	unmarshaller, err := NewUnmarshaller(bytes)
	if err != nil {
		return errors.Wrap(err, "failed to deserialize")
	}
	a.Zr, err = unmarshaller.NextZr()
	if err != nil {
		return errors.Wrap(err, "failed to deserialize zr")
	}
	a.G1, err = unmarshaller.NextG1()
	if err != nil {
		return errors.Wrap(err, "failed to deserialize g1")
	}
	a.G2, err = unmarshaller.NextG2()
	if err != nil {
		return errors.Wrap(err, "failed to deserialize g2")
	}
	a.ZrArray, err = unmarshaller.NextZrArray()
	if err != nil {
		return errors.Wrap(err, "failed to deserialize zrArray")
	}
	a.G1Array, err = unmarshaller.NextG1Array()
	if err != nil {
		return errors.Wrap(err, "failed to deserialize g1Array")
	}
	zr, err := unmarshaller.NextZr()
	if zr != nil {
		return errors.Wrap(err, "no more values expected")
	}
	if err != nil {
		return errors.Wrap(err, "no error expected")
	}
	g1, err := unmarshaller.NextG1()
	if g1 != nil {
		return errors.Wrap(err, "no more values expected")
	}
	if err != nil {
		return errors.Wrap(err, "no error expected")
	}
	g2, err := unmarshaller.NextG2()
	if g2 != nil {
		return errors.Wrap(err, "no more values expected")
	}
	if err != nil {
		return errors.Wrap(err, "no error expected")
	}
	g1A, err := unmarshaller.NextG1Array()
	if g1A != nil {
		return errors.Wrap(err, "no more values expected")
	}
	if err != nil {
		return errors.Wrap(err, "no error expected")
	}
	zrA, err := unmarshaller.NextZrArray()
	if zrA != nil {
		return errors.Wrap(err, "no more values expected")
	}
	if err != nil {
		return errors.Wrap(err, "no error expected")
	}

	return nil
}

type Rectangle struct {
	Length int
	Height int
}

func (a *Rectangle) Serialize() ([]byte, error) {
	return MarshalStd(*a)
}

func (a *Rectangle) Deserialize(bytes []byte) error {
	_, err := asn1.Unmarshal(bytes, a)
	return err
}

type Square struct {
	Length int
}

func (s *Square) Serialize() ([]byte, error) {
	return asn1.Marshal(*s)
}

func (s *Square) Deserialize(bytes []byte) error {
	_, err := asn1.Unmarshal(bytes, s)
	return err
}

type Failure struct{}

func (a *Failure) Serialize() ([]byte, error) {
	return nil, errors.New("failure serialization")
}

func (a *Failure) Deserialize(bytes []byte) error {
	return errors.New("failure deserialization")
}

func TestMarshal(t *testing.T) {
	_, err := Marshal[Serializer](&Failure{})
	assert.Error(t, err)
	assert.EqualError(t, err, "failed to serialize value: failure serialization")

	a := &Rectangle{
		Length: 5,
		Height: 9,
	}
	var square *Square
	raw, err := Marshal[Serializer](a, square)
	assert.NoError(t, err)

	a1 := &Rectangle{}
	var square1 *Square
	// test failures
	err = Unmarshal[Serializer]([]byte{0, 1, 2})
	assert.Error(t, err)
	assert.EqualError(t, err, "failed to unmarshal values: asn1: structure error: tags don't match (16 vs {class:0 tag:0 length:1 isCompound:false}) {optional:false explicit:false application:false private:false defaultValue:<nil> tag:<nil> stringType:0 timeType:0 set:false omitEmpty:false} Values @2")
	err = Unmarshal[Serializer](raw, a1)
	assert.Error(t, err)
	assert.EqualError(t, err, "number of values does not match number of values")
	err = Unmarshal[Serializer](raw, &Failure{}, square1)
	assert.Error(t, err)
	assert.EqualError(t, err, "failed to deserialize value [0 of 2]: failure deserialization")

	// success
	err = Unmarshal[Serializer](raw, a1, square1)
	assert.NoError(t, err)
	assert.Equal(t, a, a1)
	assert.Equal(t, square, square1)

	err = Unmarshal[Serializer](raw, a1, &Failure{})
	assert.NoError(t, err) // This is because at marshalling time, square was nil
}

func TestUnmarshaller(t *testing.T) {
	curve := math.Curves[math.BN254]
	p, err := NewRandomMathContainer(curve)
	assert.NoError(t, err)
	raw, err := p.Serialize()
	assert.NoError(t, err)

	p1 := &MathContainer{}
	// some errors
	err = p1.Deserialize([]byte{0, 1, 2})
	assert.Error(t, err)
	assert.EqualError(t, err, "failed to deserialize: failed to unmarshal values: asn1: structure error: tags don't match (16 vs {class:0 tag:0 length:1 isCompound:false}) {optional:false explicit:false application:false private:false defaultValue:<nil> tag:<nil> stringType:0 timeType:0 set:false omitEmpty:false} Values @2")
	// success
	err = p1.Deserialize(raw)
	assert.NoError(t, err)
	assert.Equal(t, p, p1)
}

func TestArray(t *testing.T) {
	r1 := &Rectangle{
		Length: 5,
		Height: 9,
	}
	r2 := &Rectangle{
		Length: 1,
		Height: 2,
	}
	a1, err := NewArray([]*Rectangle{r1, r2})
	assert.NoError(t, err)
	raw, err := a1.Serialize()
	assert.NoError(t, err)

	a2, err := NewArrayWithNew[*Rectangle](func() *Rectangle {
		return &Rectangle{}
	})
	assert.NoError(t, a2.Deserialize(raw))
	assert.Equal(t, a1.Values, a2.Values)
}
