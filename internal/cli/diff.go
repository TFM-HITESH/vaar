/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/envaar/vaar/internal/diff"
	"github.com/envaar/vaar/internal/fs"
	diffoutput "github.com/envaar/vaar/internal/output/diff"
	"github.com/spf13/cobra"
)

func newDiffCmd() *cobra.Command {
	var jsonOutput bool
	var quiet bool

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

			leftData, err := readDiffFile(leftPath)
			if err != nil {
				return err
			}

			rightData, err := readDiffFile(rightPath)
			if err != nil {
				return err
			}

			result, err := diff.Compare(leftPath, leftData, rightPath, rightData)
			if err != nil {
				return NewToolError("comparing dotenv files", err)
			}

			different := result.HasDifferences()

			if jsonOutput {
				if err := writeDiffJSON(cmd, result); err != nil {
					return err
				}
			} else if !quiet {
				if err := writeDiffText(cmd, result); err != nil {
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

func readDiffFile(path string) ([]byte, error) {
	data, err := fs.ReadFile(path)
	if err != nil {
		switch {
		case errors.Is(err, fs.ErrIsDirectory):
			return nil, NewToolError(fmt.Sprintf("%s is a directory, expected a dotenv file", path), nil)
		case errors.Is(err, fs.ErrNotRegularFile):
			return nil, NewToolError(fmt.Sprintf("%s is not a regular file, expected a dotenv file", path), nil)
		case errors.Is(err, os.ErrNotExist):
			return nil, NewToolError(fmt.Sprintf("reading %s: file does not exist", path), nil)
		default:
			return nil, NewToolError(fmt.Sprintf("reading %s", path), err)
		}
	}
	return data, nil
}

func writeDiffText(cmd *cobra.Command, result diff.Result) error {
	return writeDiffLine(cmd, diffoutput.Text(result))
}

func writeDiffJSON(cmd *cobra.Command, result diff.Result) error {
	data, err := diffoutput.JSON(result)
	if err != nil {
		return NewToolError("rendering diff JSON output failed", err)
	}

	return writeDiffLine(cmd, string(data))
}

func writeDiffLine(cmd *cobra.Command, line string) error {
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), line); err != nil {
		return NewToolError("writing diff output failed", err)
	}
	return nil
}
