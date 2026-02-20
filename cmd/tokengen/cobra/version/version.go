/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package version

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/spf13/cobra"
)

const ProgramName = "tokengen"

var Commit, Time, Modified = func() (string, string, string) {
	if info, ok := debug.ReadBuildInfo(); ok {
		var revision string
		var time string
		var modified string
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				revision = setting.Value
			}
			if setting.Key == "vcs.time" {
				time = setting.Value
			}
			if setting.Key == "vcs.modified" {
				modified = setting.Value
			}
		}

		return revision, time, modified
	}

	return "", "", ""
}()

// Cmd returns the Cobra Command for Version
func Cmd() *cobra.Command {
	return cobraCommand
}

var cobraCommand = &cobra.Command{
	Use:   "version",
	Short: "Print tokengen version.",
	Long:  `Print current version of tokengen.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return errors.New("trailing args detected")
		}
		// Parsing of the command line is done so silence cmd usage
		cmd.SilenceUsage = true
		_, err := fmt.Fprint(cmd.OutOrStdout(), GetInfo())

		return err
	},
}

// GetInfo returns version information for the peer
func GetInfo() string {
	return fmt.Sprintf("%s\n"+
		" Version: (%s, %s, %s)\n "+
		" Go version: %s\n"+
		" OS/Arch: %s\n",
		ProgramName, Commit, Time, Modified,
		runtime.Version(),
		fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
}
