package bugcmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/commands/execenv"
)

type batchOptions struct {
	status    string
	addLabels []string
	rmLabels  []string
}

func newBugBatchCommand(env *execenv.Env) *cobra.Command {
	options := batchOptions{}

	cmd := &cobra.Command{
		Use:   "batch",
		Short: "Batch operate on multiple bugs",
		Long: `Batch operate on multiple bugs by reading bug IDs from stdin, one per line.

Supported operations: set status, add labels, remove labels.
Partial failures will not interrupt the batch; all results are reported.`,
		Example: `Close multiple bugs:
  git bug batch --status closed < bug_ids.txt

Add labels to multiple bugs:
  git bug batch --label bug --label urgent < bug_ids.txt

Remove labels and close:
  git bug batch --status closed --rm-label wip < bug_ids.txt`,
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugBatch(env, options)
		}),
	}

	flags := cmd.Flags()
	flags.StringVar(&options.status, "status", "",
		"Set status. Valid values are [open, closed]")
	cmd.RegisterFlagCompletionFunc("status",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return []string{"open", "closed"}, cobra.ShellCompDirectiveNoFileComp
		})
	flags.StringSliceVarP(&options.addLabels, "label", "l", nil,
		"Labels to add")
	cmd.RegisterFlagCompletionFunc("label", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	})
	flags.StringSliceVar(&options.rmLabels, "rm-label", nil,
		"Labels to remove")
	cmd.RegisterFlagCompletionFunc("rm-label", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

type batchResult struct {
	bugID   string
	success bool
	message string
}

func runBugBatch(env *execenv.Env, opts batchOptions) error {
	if opts.status == "" && len(opts.addLabels) == 0 && len(opts.rmLabels) == 0 {
		return fmt.Errorf("no operation specified. Use --status, --label, or --rm-label")
	}

	var bugIDs []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			bugIDs = append(bugIDs, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stdin: %w", err)
	}

	if len(bugIDs) == 0 {
		return fmt.Errorf("no bug IDs provided on stdin")
	}

	results := make([]batchResult, 0, len(bugIDs))
	successCount := 0
	failCount := 0

	for _, idStr := range bugIDs {
		result := processBugBatch(env, idStr, opts)
		results = append(results, result)
		if result.success {
			successCount++
		} else {
			failCount++
		}
	}

	for _, r := range results {
		status := "OK"
		if !r.success {
			status = "FAIL"
		}
		env.Out.Printf("[%s] %s: %s\n", status, r.bugID, r.message)
	}

	env.Out.Printf("\nSummary: %d succeeded, %d failed, %d total\n", successCount, failCount, len(bugIDs))

	if failCount > 0 {
		return fmt.Errorf("%d operation(s) failed", failCount)
	}

	return nil
}

func processBugBatch(env *execenv.Env, idStr string, opts batchOptions) batchResult {
	b, err := env.Backend.Bugs().ResolvePrefix(idStr)
	if err != nil {
		return batchResult{
			bugID:   idStr,
			success: false,
			message: fmt.Sprintf("failed to resolve bug: %v", err),
		}
	}

	bugID := b.Id().Human()
	var messages []string

	if opts.status != "" {
		switch opts.status {
		case "open":
			_, err := b.Open()
			if err != nil {
				return batchResult{
					bugID:   bugID,
					success: false,
					message: fmt.Sprintf("failed to open: %v", err),
				}
			}
			messages = append(messages, "status set to open")
		case "closed":
			_, err := b.Close()
			if err != nil {
				return batchResult{
					bugID:   bugID,
					success: false,
					message: fmt.Sprintf("failed to close: %v", err),
				}
			}
			messages = append(messages, "status set to closed")
		default:
			return batchResult{
				bugID:   bugID,
				success: false,
				message: fmt.Sprintf("invalid status: %s", opts.status),
			}
		}
	}

	if len(opts.addLabels) > 0 || len(opts.rmLabels) > 0 {
		changes, _, err := b.ChangeLabels(opts.addLabels, opts.rmLabels)
		if err != nil {
			return batchResult{
				bugID:   bugID,
				success: false,
				message: fmt.Sprintf("failed to change labels: %v", err),
			}
		}
		for _, change := range changes {
			messages = append(messages, change.String())
		}
	}

	err = b.Commit()
	if err != nil {
		return batchResult{
			bugID:   bugID,
			success: false,
			message: fmt.Sprintf("failed to commit: %v", err),
		}
	}

	return batchResult{
		bugID:   bugID,
		success: true,
		message: strings.Join(messages, "; "),
	}
}
