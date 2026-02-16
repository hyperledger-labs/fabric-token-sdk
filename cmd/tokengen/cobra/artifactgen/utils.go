/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package artifactgen

import (
	"io/fs"
	"os"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"gopkg.in/yaml.v2"
)

type Topologies struct {
	Topologies []api.Topology `yaml:"topologies,omitempty"`
}

func WriteTopologies(fileName string, topologies []api.Topology, perm fs.FileMode) error {
	raw, err := yaml.Marshal(&Topologies{Topologies: topologies})
	if err != nil {
		return errors.Wrap(err, "failed to marshal topologies")
	}
	if err := os.WriteFile(fileName, raw, perm); err != nil {
		return errors.Wrapf(err, "failed to write to [%s]", fileName)
	}

	return nil
}
