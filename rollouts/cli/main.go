// Copyright 2022 Google LLC
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

package main

import (
	"context"
	"os"

	"github.com/GoogleContainerTools/kpt/rollouts/cli/get"
	"github.com/GoogleContainerTools/kpt/rollouts/cli/status"
	"github.com/spf13/cobra"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	ctx := context.Background()

	rolloutsCmd := &cobra.Command{
		Use:   "cli",
		Short: "cli ",
		Long:  "cli",
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
	if err := rolloutsCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}
