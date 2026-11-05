package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"

	"github.com/amikai/openings-mcp/internal/provider/jobsacuk"
)

const defaultBaseURL = "https://www.jobs.ac.uk"

func main() {
	os.Exit(run())
}

func run() int {
	rootFlags := ff.NewFlagSet("jobsacuk")
	var (
		baseURL = rootFlags.StringLong("base-url", defaultBaseURL, "jobs.ac.uk base URL")
		timeout = rootFlags.DurationLong("timeout", 60*time.Second, "request timeout")
		format  = rootFlags.StringEnumLong("format", "output format", "text", "json")
	)
	rootCmd := &ff.Command{
		Name:  "jobsacuk",
		Usage: "jobsacuk [FLAGS] <search|detail> [FLAGS]",
		Flags: rootFlags,
	}

	searchFS := ff.NewFlagSet("search").SetParent(rootFlags)
	var (
		keyword   = searchFS.StringLong("keyword", "", "free-text keywords")
		locations = searchFS.StringListLong("location", "location facet slug, e.g. london (repeatable)")
		nonAcad   = searchFS.StringListLong("area", "non-academic discipline slug, e.g. senior-management (repeatable)")
		acad      = searchFS.StringListLong("subject", "academic discipline slug, e.g. computer-sciences (repeatable)")
		salary    = searchFS.StringListLong("salary-band", "salary band slug, e.g. 50000-59999 (repeatable)")
		contract  = searchFS.StringListLong("contract", "permanent, fixed-term-contract or temporary (repeatable)")
		sort      = searchFS.StringEnumLong("sort", "result order", jobsacuk.SortRelevance, jobsacuk.SortDatePlaced, jobsacuk.SortClosingDate)
		page      = searchFS.IntLong("page", 1, "1-based page")
	)
	searchCmd := &ff.Command{
		Name:      "search",
		Usage:     "jobsacuk search [--keyword TEXT] [--location SLUG]... [--area SLUG]... [--subject SLUG]... [--salary-band SLUG]... [--contract SLUG]... [--sort relevance|date_placed|closing_date] [--page N]",
		ShortHelp: "search jobs.ac.uk (25 results per page)",
		Flags:     searchFS,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("search takes no positional arguments, got %v", args)
			}
			if *page < 1 {
				return fmt.Errorf("--page must be >= 1, got %d", *page)
			}
			ctx, cancel := context.WithTimeout(ctx, *timeout)
			defer cancel()
			resp, err := jobsacuk.NewClient(*baseURL, nil).Search(ctx, &jobsacuk.SearchRequest{
				Keyword:                *keyword,
				Locations:              *locations,
				NonAcademicDisciplines: *nonAcad,
				AcademicDisciplines:    *acad,
				SalaryBands:            *salary,
				ContractTypes:          *contract,
				Sort:                   *sort,
				Page:                   *page,
			})
			if err != nil {
				return err
			}
			if *format == "json" {
				return printJSON(resp)
			}
			fmt.Printf("total=%d page=%d total_pages=%d\n\n", resp.Total, resp.Page, resp.TotalPages)
			for i, j := range resp.Jobs {
				fmt.Printf("%d. [%s] %s\n", i+1, j.ID, j.Title)
				printField("employer", j.Employer)
				printField("department", j.Department)
				printField("location", j.Location)
				printField("salary", j.Salary)
				printField("date_placed", j.DatePlaced)
				printField("closes", j.Closes)
				printField("url", j.URL)
				fmt.Println()
			}
			return nil
		},
	}
	rootCmd.Subcommands = append(rootCmd.Subcommands, searchCmd)

	detailFS := ff.NewFlagSet("detail").SetParent(rootFlags)
	id := detailFS.StringLong("id", "", "advert id from search, e.g. DTJ660, or a job URL")
	detailCmd := &ff.Command{
		Name:      "detail",
		Usage:     "jobsacuk detail --id ID [--format text|json]",
		ShortHelp: "fetch one advert from /job/{id}",
		Flags:     detailFS,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("detail takes no positional arguments, got %v (did you mean --id %q?)", args, args[0])
			}
			if *id == "" {
				return errors.New("--id is required")
			}
			ctx, cancel := context.WithTimeout(ctx, *timeout)
			defer cancel()
			d, err := jobsacuk.NewClient(*baseURL, nil).JobDetail(ctx, *id)
			if err != nil {
				return err
			}
			if *format == "json" {
				return printJSON(d)
			}
			fmt.Printf("[%s] %s\n", d.ID, d.Title)
			printField("employer", d.Employer)
			printField("department", d.Department)
			printField("location", d.Location)
			printField("salary", d.Salary)
			printField("hours", d.Hours)
			printField("contract_type", d.ContractType)
			printField("posted_at", d.PostedAt)
			printField("closes_at", d.ClosesAt)
			printField("job_ref", d.JobRef)
			printField("url", d.URL)
			printField("apply_url", d.ApplyURL)
			if d.Description != "" {
				fmt.Printf("\n%s\n", d.Description)
			}
			return nil
		},
	}
	rootCmd.Subcommands = append(rootCmd.Subcommands, detailCmd)

	if err := rootCmd.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, ffhelp.Command(rootCmd.GetSelected()))
		if errors.Is(err, ff.ErrHelp) {
			return 0
		}
		fmt.Fprintln(os.Stderr, "err:", err)
		return 1
	}
	if rootCmd.GetSelected() == rootCmd {
		fmt.Fprintln(os.Stderr, ffhelp.Command(rootCmd))
		fmt.Fprintln(os.Stderr, "err: a subcommand (search or detail) is required")
		return 1
	}
	if err := rootCmd.Run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "err:", err)
		return 1
	}
	return 0
}

func printField(name, value string) {
	if value != "" {
		fmt.Printf("   %s: %s\n", name, value)
	}
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
