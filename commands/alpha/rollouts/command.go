// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rollouts

import (
	"context"

	"github.com/GoogleContainerTools/kpt/commands/alpha/rollouts/get"
	"github.com/GoogleContainerTools/kpt/commands/alpha/rollouts/status"
	"github.com/spf13/cobra"
)

func NewCommand(ctx context.Context, version string) *cobra.Command {
	rolloutsCmd := &cobra.Command{
		Use:   "rollouts",
		Short: "[Alpha] " + "rollouts",
		Long:  "[Alpha] " + "rollouts",
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := cmd.Flags().GetBool("help")
			if err != nil {
				return err
			}
			if h {
				return cmd.Help()
			}
			return cmd.Usage()
		},
	}

	rolloutsCmd.AddCommand(
		get.NewCommand(ctx),
		status.NewCommand(ctx),
	)

	return rolloutsCmd
}
