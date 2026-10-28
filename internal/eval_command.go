package mino

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	assets "github.com/qshine/mino"
	"github.com/qshine/mino/internal/eval"
	"github.com/qshine/mino/internal/gateway"
)

const evalUsage = `Usage:
  mino eval [--live] [--runs N] [--filter NAME] [--out DIR] [--timeout DURATION] [PATH]

PATH is one case file or a directory of *.yaml cases (default ./evals).
Replay mode (default) drives the Agent against a scripted local model: no key,
no network, no user data. --live uses the model from ~/.mino/config.json and
runs every case N times (default 3) to report pass rates and token usage;
cases marked live: false are skipped there.

  --runs N        runs per case (default 1; 3 with --live; at most 32)
  --filter NAME   run only the named case
  --out DIR       write report.json and one session log per run under DIR
  --timeout D     limit for one run (default 2m)

Every run uses a fresh temporary workspace, the manual permission mode, and no
MCP servers. Bash still runs with your account permissions. The exit status is 1
when any case fails.`

// runEval evaluates declared cases. It never opens the user's sessions, and in
// replay mode it never reads ~/.mino at all.
func runEval(ctx context.Context, cli *gateway.CLI, args []string, output io.Writer) error {
	if len(args) > 0 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h") {
		_, err := fmt.Fprintln(output, evalUsage)
		return err
	}
	flags := flag.NewFlagSet("mino eval", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	live := flags.Bool("live", false, "")
	runs := flags.Int("runs", 1, "")
	filter := flags.String("filter", "", "")
	out := flags.String("out", "", "")
	timeout := flags.Duration("timeout", 2*time.Minute, "")
	if err := flags.Parse(args); err != nil || flags.NArg() > 1 {
		return errors.New("Invalid eval arguments. Run mino eval help for usage.")
	}
	runsSet := false
	flags.Visit(func(f *flag.Flag) { runsSet = runsSet || f.Name == "runs" })
	if *live && !runsSet {
		*runs = 3
	}
	if *runs < 1 || *runs > eval.MaxRuns {
		return fmt.Errorf("--runs must be between 1 and %d", eval.MaxRuns)
	}
	if *timeout < time.Second || *timeout > 30*time.Minute {
		return errors.New("--timeout must be between 1s and 30m")
	}
	path := "./evals"
	if flags.NArg() == 1 {
		path = flags.Arg(0)
	}
	cases, err := eval.Load(path)
	if err != nil {
		return err
	}
	if *filter != "" {
		kept := cases[:0]
		for _, c := range cases {
			if c.Name == *filter {
				kept = append(kept, c)
			}
		}
		if len(kept) == 0 {
			return fmt.Errorf("No eval case named %q in %s", *filter, path)
		}
		cases = kept
	}
	options := eval.Options{Live: *live, Instructions: assets.DefaultSoul, Timeout: *timeout}
	model := "scripted"
	if *live {
		cfg, err := loadConfig(nil)
		if err != nil {
			return fmt.Errorf("--live requires a complete %s: %w", configLocation, err)
		}
		instructions, err := loadInstructions()
		if err != nil {
			return err
		}
		options.BaseURL, options.APIKey, options.Model, options.Instructions, options.Subagents = cfg.BaseURL, cfg.APIKey, cfg.Model, instructions, cfg.MaxConcurrentSubagents
		model = cfg.Model
		// A replay-only case asserts a program path (an unknown tool, a scripted
		// retry) that a real model would not reproduce; it is not a model failure.
		var skipped []string
		kept := cases[:0]
		for _, c := range cases {
			if c.ReplayOnly {
				skipped = append(skipped, c.Name)
				continue
			}
			kept = append(kept, c)
		}
		cases = kept
		if len(skipped) > 0 {
			if _, err := fmt.Fprintf(output, "Skipped replay-only cases (live: false): %s\n", strings.Join(skipped, ", ")); err != nil {
				return err
			}
		}
		if len(cases) == 0 {
			return errors.New("No cases left to run live; every case is replay-only")
		}
	} else {
		for _, c := range cases {
			if len(c.Script) == 0 {
				return fmt.Errorf("Case %s has no model script; add one or run with --live", c.Name)
			}
		}
	}
	if *out != "" {
		absolute, err := filepath.Abs(*out)
		if err != nil {
			return fmt.Errorf("Invalid --out directory: %w", err)
		}
		if home, err := os.UserHomeDir(); err == nil {
			if rel, err := filepath.Rel(filepath.Join(home, ".mino"), absolute); err == nil && (rel == "." || filepath.IsLocal(rel)) {
				return errors.New("--out must not be inside ~/.mino")
			}
		}
		if err := os.MkdirAll(absolute, 0700); err != nil {
			return fmt.Errorf("Cannot create --out directory: %w", err)
		}
		options.Out = absolute
	}
	mode := "replay"
	if *live {
		mode = "live"
	}
	report := eval.Report{Mode: mode, Model: model, Started: time.Now()}
	for _, c := range cases {
		count := *runs
		if c.Runs > 0 {
			count = c.Runs
		}
		var results []eval.RunResult
		for run := 1; run <= count; run++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			result, err := eval.RunCase(ctx, c, options, run)
			if err != nil {
				return err
			}
			results = append(results, result)
		}
		report.Cases = append(report.Cases, eval.Summarize(c.Name, results))
	}
	report.Finished = time.Now()
	report.AllPassed = report.Failed() == 0
	if err := report.WriteText(output); err != nil {
		return err
	}
	if options.Out != "" {
		if err := report.WriteJSON(filepath.Join(options.Out, "report.json")); err != nil {
			return fmt.Errorf("Cannot write report.json: %w", err)
		}
		if _, err := fmt.Fprintf(output, "Report written to %s\n", gateway.Text(options.Out)); err != nil {
			return err
		}
	}
	if !report.AllPassed {
		return fmt.Errorf("Eval failed: %d of %d cases did not pass every run", report.Failed(), len(report.Cases))
	}
	return nil
}
