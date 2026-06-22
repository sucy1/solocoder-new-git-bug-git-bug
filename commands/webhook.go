package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/entities/webhook"
)

func newWebhookCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Manage webhooks",
		Long: `Manage webhooks for bug events.

Webhooks are triggered when bugs are created, updated, or closed.
Webhook configuration is stored in git config.`,
		Example: `List webhooks:
  git bug webhook list

Add a webhook:
  git bug webhook add myhook https://example.com/webhook --events create,close

Remove a webhook:
  git bug webhook remove myhook`,
	}

	cmd.AddCommand(newWebhookListCommand(env))
	cmd.AddCommand(newWebhookAddCommand(env))
	cmd.AddCommand(newWebhookRemoveCommand(env))

	return cmd
}

func newWebhookListCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all webhooks",
		Aliases: []string{"ls"},
		PreRunE: execenv.LoadRepo(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runWebhookList(env)
		}),
	}

	return cmd
}

func runWebhookList(env *execenv.Env) error {
	webhooks, err := webhook.ListWebhooks(env.Repo.AnyConfig())
	if err != nil {
		return err
	}

	if len(webhooks) == 0 {
		env.Out.Println("No webhooks configured.")
		return nil
	}

	for _, wh := range webhooks {
		events := "*"
		if len(wh.Events) > 0 {
			events = joinEventsStr(wh.Events)
		}
		env.Out.Printf("%s:\n", wh.Name)
		env.Out.Printf("  URL:    %s\n", wh.URL)
		env.Out.Printf("  Events: %s\n", events)
		env.Out.Println()
	}

	return nil
}

type webhookAddOptions struct {
	events []string
}

func newWebhookAddCommand(env *execenv.Env) *cobra.Command {
	options := webhookAddOptions{}

	cmd := &cobra.Command{
		Use:   "add NAME URL",
		Short: "Add a new webhook",
		PreRunE: execenv.LoadRepo(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runWebhookAdd(env, options, args)
		}),
	}

	flags := cmd.Flags()
	flags.StringSliceVarP(&options.events, "events", "e", nil,
		"Events to trigger on. Valid values: create, update, close, *")
	cmd.RegisterFlagCompletionFunc("events",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return webhook.ValidEvents(), cobra.ShellCompDirectiveNoFileComp
		})

	return cmd
}

func runWebhookAdd(env *execenv.Env, opts webhookAddOptions, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: git bug webhook add NAME URL")
	}

	name := args[0]
	url := args[1]

	for _, e := range opts.events {
		if err := webhook.ValidateEvent(e); err != nil {
			return err
		}
	}

	err := webhook.AddWebhook(env.Repo.LocalConfig(), name, url, opts.events)
	if err != nil {
		return err
	}

	env.Out.Printf("Webhook '%s' added.\n", name)
	return nil
}

func newWebhookRemoveCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove NAME",
		Short:   "Remove a webhook",
		Aliases: []string{"rm"},
		PreRunE: execenv.LoadRepo(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runWebhookRemove(env, args)
		}),
	}

	return cmd
}

func runWebhookRemove(env *execenv.Env, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: git bug webhook remove NAME")
	}

	name := args[0]

	err := webhook.RemoveWebhook(env.Repo.LocalConfig(), name)
	if err != nil {
		return err
	}

	env.Out.Printf("Webhook '%s' removed.\n", name)
	return nil
}

func joinEventsStr(events []string) string {
	result := ""
	for i, e := range events {
		if i > 0 {
			result += ", "
		}
		result += e
	}
	return result
}
