package commands

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entity"
)

type reportOptions struct {
	format string
}

type Report struct {
	TotalBugs      int               `json:"total_bugs"`
	StatusCounts   map[string]int    `json:"status_counts"`
	LabelCounts    map[string]int    `json:"label_counts"`
	AvgResolveTime float64           `json:"avg_resolve_time_seconds"`
	TopReporters   []ReporterCount   `json:"top_reporters"`
}

type ReporterCount struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Count  int    `json:"count"`
}

func NewReportCommand(env *execenv.Env) *cobra.Command {
	options := reportOptions{}

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate project statistics report",
		Long: `Generate a project statistics report.

Includes:
- Bug distribution by status
- Bug distribution by label
- Average resolve time
- Top 10 most active reporters`,
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runReport(env, options)
		}),
	}

	flags := cmd.Flags()
	flags.StringVarP(&options.format, "format", "f", "text",
		"Output format. Valid values are [text, json]")
	cmd.RegisterFlagCompletionFunc("format",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return []string{"text", "json"}, cobra.ShellCompDirectiveNoFileComp
		})

	return cmd
}

func runReport(env *execenv.Env, opts reportOptions) error {
	allIds := env.Backend.Bugs().AllIds()

	statusCounts := make(map[string]int)
	labelCounts := make(map[string]int)
	reporterCounts := make(map[entity.Id]int)
	var totalResolveTime float64
	var closedCount int

	for _, id := range allIds {
		b, err := env.Backend.Bugs().Resolve(id)
		if err != nil {
			return err
		}

		snap := b.Snapshot()

		statusCounts[snap.Status.String()]++

		for _, label := range snap.Labels {
			labelCounts[label.String()]++
		}

		authorId := snap.Author.Id()
		reporterCounts[authorId]++

		if snap.Status == common.ClosedStatus {
			resolveTime, ok := calculateResolveTime(snap)
			if ok {
				totalResolveTime += resolveTime
				closedCount++
			}
		}
	}

	var avgResolveTime float64
	if closedCount > 0 {
		avgResolveTime = totalResolveTime / float64(closedCount)
	}

	topReporters := calculateTopReporters(env, reporterCounts, 10)

	report := Report{
		TotalBugs:      len(allIds),
		StatusCounts:   statusCounts,
		LabelCounts:    labelCounts,
		AvgResolveTime: avgResolveTime,
		TopReporters:   topReporters,
	}

	switch opts.format {
	case "json":
		return printReportJSON(env, report)
	case "text":
		return printReportText(env, report)
	default:
		return fmt.Errorf("unknown format: %s", opts.format)
	}
}

func calculateResolveTime(snap *bug.Snapshot) (float64, bool) {
	var firstCloseTime time.Time
	found := false

	for _, item := range snap.Timeline {
		if statusItem, ok := item.(*bug.SetStatusTimelineItem); ok {
			if statusItem.Status == common.ClosedStatus {
				firstCloseTime = statusItem.UnixTime.Time()
				found = true
				break
			}
		}
	}

	if !found {
		return 0, false
	}

	resolveTime := firstCloseTime.Sub(snap.CreateTime)
	return resolveTime.Seconds(), true
}

func calculateTopReporters(env *execenv.Env, counts map[entity.Id]int, topN int) []ReporterCount {
	type pair struct {
		id    entity.Id
		count int
	}

	pairs := make([]pair, 0, len(counts))
	for id, count := range counts {
		pairs = append(pairs, pair{id, count})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].count > pairs[j].count
	})

	result := make([]ReporterCount, 0, topN)
	for i := 0; i < len(pairs) && i < topN; i++ {
		id := pairs[i].id
		name := id.Human()
		excerpt, err := env.Backend.Identities().ResolveExcerpt(id)
		if err == nil {
			name = excerpt.DisplayName()
		}
		result = append(result, ReporterCount{
			Id:    id.Human(),
			Name:  name,
			Count: pairs[i].count,
		})
	}

	return result
}

func printReportText(env *execenv.Env, report Report) error {
	env.Out.Println("Project Statistics Report")
	env.Out.Println("========================")
	env.Out.Println()

	env.Out.Printf("Total bugs: %d\n", report.TotalBugs)
	env.Out.Println()

	env.Out.Println("Status distribution:")
	for status, count := range report.StatusCounts {
		env.Out.Printf("  %-8s: %d\n", status, count)
	}
	env.Out.Println()

	env.Out.Println("Label distribution:")
	if len(report.LabelCounts) == 0 {
		env.Out.Println("  (none)")
	} else {
		type labelPair struct {
			label string
			count int
		}
		pairs := make([]labelPair, 0, len(report.LabelCounts))
		for label, count := range report.LabelCounts {
			pairs = append(pairs, labelPair{label, count})
		}
		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].count > pairs[j].count
		})
		for _, p := range pairs {
			env.Out.Printf("  %-20s: %d\n", p.label, p.count)
		}
	}
	env.Out.Println()

	env.Out.Println("Average resolve time:")
	if report.AvgResolveTime == 0 {
		env.Out.Println("  (no closed bugs)")
	} else {
		env.Out.Printf("  %s\n", formatDuration(report.AvgResolveTime))
	}
	env.Out.Println()

	env.Out.Println("Top 10 reporters:")
	for i, r := range report.TopReporters {
		env.Out.Printf("  %2d. %s (%s): %d bugs\n", i+1, r.Name, r.Id, r.Count)
	}
	env.Out.Println()

	return nil
}

func printReportJSON(env *execenv.Env, report Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	env.Out.Println(string(data))
	return nil
}

func formatDuration(seconds float64) string {
	days := int(seconds / 86400)
	hours := int((seconds - float64(days)*86400) / 3600)
	minutes := int((seconds - float64(days)*86400 - float64(hours)*3600) / 60)

	if days > 0 {
		return fmt.Sprintf("%d days, %d hours, %d minutes", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%d hours, %d minutes", hours, minutes)
	} else {
		return fmt.Sprintf("%d minutes", minutes)
	}
}
