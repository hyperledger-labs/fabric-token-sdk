package pp

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/pp"
)

// Marshal marshals the passed public parameters
func Marshal(pp *pp.PublicParameters) ([]byte, error) {
	return json.Marshal(pp)
}

func Unmarshal(raw []byte) (*pp.PublicParameters, error) {
	pp := &pp.PublicParameters{}
	if err := json.Unmarshal(raw, pp); err != nil {
		return nil, err
	}
	return pp, nil
}
