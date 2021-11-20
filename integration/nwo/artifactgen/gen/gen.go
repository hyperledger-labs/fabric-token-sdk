/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package gen

import (
	"fmt"
	"io/ioutil"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/weaver"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"

	"gopkg.in/yaml.v2"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
)

type Topology struct {
	Type string `yaml:"type,omitempty"`
}

type Topologies struct {
	Topologies []Topology `yaml:"topologies,omitempty"`
}

type T struct {
	Topologies []interface{} `yaml:"topologies,omitempty"`
}

var topologyFile string
var output string
var port int

// Cmd returns the Cobra Command for Version
func Cmd() *cobra.Command {
	// Set the flags on the node start command.
	flags := cobraCommand.Flags()
	flags.StringVarP(&topologyFile, "topology", "t", "", "topology file in yaml format")
	flags.StringVarP(&output, "output", "o", "./testdata", "output folder")
	flags.IntVarP(&port, "port", "p", 20000, "host starting port")

	return cobraCommand
}

var cobraCommand = &cobra.Command{
	Use:   "artifacts",
	Short: "Gen artifacts.",
	Long:  `Read topology from file and generates artifacts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("trailing args detected")
		}
		// Parsing of the command line is done so silence cmd usage
		cmd.SilenceUsage = true
		return gen(args)
	},
}

// gen read topology and generates artifacts
func gen(args []string) error {
	if len(topologyFile) == 0 {
		return errors.Errorf("expecting topology file path")
	}
	raw, err := ioutil.ReadFile(topologyFile)
	if err != nil {
		return errors.Wrapf(err, "failed reading topology file [%s]", topologyFile)
	}
	names := &Topologies{}
	if err := yaml.Unmarshal(raw, names); err != nil {
		return errors.Wrapf(err, "failed unmarshalling topology file [%s]", topologyFile)
	}

	t := &T{}
	if err := yaml.Unmarshal(raw, t); err != nil {
		return errors.Wrapf(err, "failed unmarshalling topology file [%s]", topologyFile)
	}
	t2 := []api.Topology{}
	for i, topology := range names.Topologies {
		switch topology.Type {
		case fabric.TopologyName:
			top := fabric.NewDefaultTopology()
			r, err := yaml.Marshal(t.Topologies[i])
			if err != nil {
				return errors.Wrapf(err, "failed remarshalling topology configuration [%s]", topologyFile)
			}
			if err := yaml.Unmarshal(r, top); err != nil {
				return errors.Wrapf(err, "failed unmarshalling topology file [%s]", topologyFile)
			}
			t2 = append(t2, top)
		case fsc.TopologyName:
			top := fsc.NewTopology()
			r, err := yaml.Marshal(t.Topologies[i])
			if err != nil {
				return errors.Wrapf(err, "failed remarshalling topology configuration [%s]", topologyFile)
			}
			if err := yaml.Unmarshal(r, top); err != nil {
				return errors.Wrapf(err, "failed unmarshalling topology file [%s]", topologyFile)
			}
			t2 = append(t2, top)
		case weaver.TopologyName:
			top := weaver.NewTopology()
			r, err := yaml.Marshal(t.Topologies[i])
			if err != nil {
				return errors.Wrapf(err, "failed remarshalling topology configuration [%s]", topologyFile)
			}
			if err := yaml.Unmarshal(r, top); err != nil {
				return errors.Wrapf(err, "failed unmarshalling topology file [%s]", topologyFile)
			}
			t2 = append(t2, top)
		case token.TopologyName:
			top := token.NewTopology()
			r, err := yaml.Marshal(t.Topologies[i])
			if err != nil {
				return errors.Wrapf(err, "failed remarshalling topology configuration [%s]", topologyFile)
			}
			if err := yaml.Unmarshal(r, top); err != nil {
				return errors.Wrapf(err, "failed unmarshalling topology file [%s]", topologyFile)
			}
			t2 = append(t2, top)
		}
	}

	network, err := integration.New(port, output, t2...)
	if err != nil {
		return errors.Wrapf(err, "cannot instantate integration infrastructure")
	}
	network.RegisterPlatformFactory(token.NewPlatformFactory())
	network.Generate()

	return nil
}
