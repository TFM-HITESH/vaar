/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package cli

import (
	"fmt"

	applicationdiff "github.com/envaar/vaar/internal/application/diff"
	diffoutput "github.com/envaar/vaar/internal/output/diff"
	"github.com/spf13/cobra"
)

func newDiffCmd() *cobra.Command {
	var jsonOutput bool
	var quiet bool
	service := applicationdiff.New()

	cmd := &cobra.Command{
		Use:   "diff <left> <right>",
		Short: "Compare dotenv key presence",
		Args:  exactDiffArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonOutput && quiet {
				return NewToolError("--quiet and --json cannot be used together", nil)
			}

			leftPath := args[0]
			rightPath := args[1]

			result, err := service.Run(cmd.Context(), applicationdiff.Options{
				LeftPath:  leftPath,
				RightPath: rightPath,
			})
			if err != nil {
				return err
			}

			different := result.HasDifferences()

			if jsonOutput {
				data, err := diffoutput.JSON(result)
				if err != nil {
					return NewToolError("rendering diff JSON output failed", err)
				}
				if err := writeDiffLine(cmd, string(data)); err != nil {
					return err
				}
			} else if !quiet {
				if err := writeDiffLine(cmd, diffoutput.Text(result)); err != nil {
					return err
				}
			}

			if different {
				return ExitError{Code: ExitFindings}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Render key differences as JSON")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "Suppress normal output and use exit status only")

	return cmd
}

func exactDiffArgs(_ *cobra.Command, args []string) error {
	if len(args) != 2 {
		return NewToolError("diff requires exactly two dotenv files", nil)
	}
	return nil
}

func writeDiffLine(cmd *cobra.Command, line string) error {
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), line); err != nil {
		return NewToolError("writing diff output failed", err)
	}
	return nil
}
