package bugcmd

import (
	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/entities/webhook"
)

func newBugStatusCloseCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "close [BUG_ID]",
		Short:   "Mark a bug as closed",
		PreRunE: execenv.LoadBackendEnsureUser(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runBugStatusClose(env, args)
		}),
		ValidArgsFunction: BugCompletion(env),
	}

	return cmd
}

func runBugStatusClose(env *execenv.Env, args []string) error {
	b, _, err := ResolveSelected(env.Backend, args)
	if err != nil {
		return err
	}

	_, err = b.Close()
	if err != nil {
		return err
	}

	err = b.Commit()
	if err != nil {
		return err
	}

	snap := b.Snapshot()
	_ = webhook.TriggerWebhooks(env.Repo.AnyConfig(), "close", b.Id().Human(), map[string]interface{}{
		"title":  snap.Title,
		"status": snap.Status.String(),
	})

	return nil
}
