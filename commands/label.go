package commands

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/entities/common"
)

func newLabelCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "label",
		Short: "List valid labels",
		Long: `List valid labels.

Note: in the future, a proper label policy could be implemented where valid labels are defined in a configuration file. Until that, the default behavior is to return the list of labels already used.`,
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runLabel(env)
		}),
	}

	cmd.AddCommand(newLabelColorCommand(env))

	return cmd
}

func runLabel(env *execenv.Env) error {
	labels := env.Backend.Bugs().ValidLabels()
	labelColors, err := common.ListLabelColors(env.Repo.LocalConfig())
	if err != nil {
		return err
	}

	for _, l := range labels {
		color := l.DefaultColor()
		if c, ok := labelColors[l]; ok {
			color = c
		}
		hex := fmt.Sprintf("#%02x%02x%02x", color.R, color.G, color.B)
		env.Out.Printf("%s\t%s\n", hex, l)
	}

	return nil
}

func newLabelColorCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "color",
		Short: "Manage label colors",
	}

	cmd.AddCommand(newLabelColorSetCommand(env))
	cmd.AddCommand(newLabelColorRmCommand(env))
	cmd.AddCommand(newLabelColorListCommand(env))

	return cmd
}

func newLabelColorSetCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set LABEL COLOR",
		Short: "Set the color of a label",
		Long: `Set the color of a label.

The color must be a valid RGB hex color (e.g., #ff0000, #f00, ff0000, f00).
Invalid colors will be rejected.`,
		Args: cobra.ExactArgs(2),
		PreRunE: execenv.LoadRepo(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLabelColorSet(env, args[0], args[1])
		},
	}

	return cmd
}

func runLabelColorSet(env *execenv.Env, labelStr, colorStr string) error {
	label := common.Label(labelStr)
	if err := label.Validate(); err != nil {
		return fmt.Errorf("invalid label: %w", err)
	}

	err := common.SetLabelColor(env.Repo.LocalConfig(), label, colorStr)
	if err != nil {
		return err
	}

	env.Out.Printf("Label %s color set to %s\n", label, colorStr)
	return nil
}

func newLabelColorRmCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm LABEL",
		Short: "Remove the custom color of a label",
		Args:  cobra.ExactArgs(1),
		PreRunE: execenv.LoadRepo(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLabelColorRm(env, args[0])
		},
	}

	return cmd
}

func runLabelColorRm(env *execenv.Env, labelStr string) error {
	label := common.Label(labelStr)
	err := common.RemoveLabelColor(env.Repo.LocalConfig(), label)
	if err != nil {
		return err
	}

	env.Out.Printf("Label %s color removed\n", label)
	return nil
}

func newLabelColorListCommand(env *execenv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all label color configurations",
		PreRunE: execenv.LoadRepo(env),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLabelColorList(env)
		},
	}

	return cmd
}

func runLabelColorList(env *execenv.Env) error {
	labelColors, err := common.ListLabelColors(env.Repo.LocalConfig())
	if err != nil {
		return err
	}

	if len(labelColors) == 0 {
		env.Out.Println("No custom label colors configured")
		return nil
	}

	labels := make([]common.Label, 0, len(labelColors))
	for l := range labelColors {
		labels = append(labels, l)
	}
	sort.Slice(labels, func(i, j int) bool {
		return string(labels[i]) < string(labels[j])
	})

	for _, l := range labels {
		color := labelColors[l]
		hex := fmt.Sprintf("#%02x%02x%02x", color.R, color.G, color.B)
		env.Out.Printf("%s\t%s\n", hex, l)
	}

	return nil
}
