package asn1

import (
	"encoding/asn1"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Rectangle struct {
	Length int
	Height int
}

func (a *Rectangle) Serialize() ([]byte, error) {
	return asn1.Marshal(*a)
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

func TestMarshal(t *testing.T) {
	a := &Rectangle{
		Length: 5,
		Height: 9,
	}
	var square *Square
	raw, err := Marshal[Serializer](a, square)
	assert.NoError(t, err)

	a1 := &Rectangle{}
	var square1 *Square
	err = Unmarshal[Serializer](raw, a1, square1)
	assert.NoError(t, err)
	assert.Equal(t, a, a1)
	assert.Equal(t, square, square1)
}
