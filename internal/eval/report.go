package eval

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/qshine/mino/internal/agent"
)

// RunResult is one execution of one case with its verdicts.
type RunResult struct {
	Run       int         `json:"run"`
	Passed    bool        `json:"passed"`
	Requests  int         `json:"requests"`
	ToolCalls int         `json:"tool_calls"`
	Usage     agent.Usage `json:"usage"`
	Outcomes  []Outcome   `json:"outcomes"`
}

type CaseReport struct {
	Name          string      `json:"name"`
	Runs          int         `json:"runs"`
	Passed        int         `json:"passed"`
	PassAtK       float64     `json:"pass_at_k"`
	PassPowK      float64     `json:"pass_pow_k"`
	MeanRequests  float64     `json:"mean_requests"`
	MeanToolCalls float64     `json:"mean_tool_calls"`
	Usage         agent.Usage `json:"usage"`
	Results       []RunResult `json:"results"`
}

type Report struct {
	Mode      string       `json:"mode"` // replay or live
	Model     string       `json:"model"`
	Started   time.Time    `json:"started"`
	Finished  time.Time    `json:"finished"`
	Cases     []CaseReport `json:"cases"`
	AllPassed bool         `json:"all_passed"`
}

// Summarize folds the runs of one case into rates. k equals the number of runs,
// so pass@k asks "did any run pass" and pass^k asks "did every run pass".
func Summarize(name string, results []RunResult) CaseReport {
	r := CaseReport{Name: name, Runs: len(results), Results: results}
	var requests, calls int
	for _, result := range results {
		if result.Passed {
			r.Passed++
		}
		requests += result.Requests
		calls += result.ToolCalls
		r.Usage.InputTokens += result.Usage.InputTokens
		r.Usage.OutputTokens += result.Usage.OutputTokens
		r.Usage.TotalTokens += result.Usage.TotalTokens
	}
	if r.Runs > 0 {
		r.MeanRequests = float64(requests) / float64(r.Runs)
		r.MeanToolCalls = float64(calls) / float64(r.Runs)
		r.PassAtK = PassAtK(r.Runs, r.Passed, r.Runs)
		r.PassPowK = PassPowK(r.Runs, r.Passed, r.Runs)
	}
	return r
}

// PassAtK estimates the chance that at least one of k attempts passes when c
// of n observed runs passed: 1 - C(n-c, k) / C(n, k).
func PassAtK(n, c, k int) float64 {
	if k <= 0 || k > n {
		return 0
	}
	return 1 - binomial(n-c, k)/binomial(n, k)
}

// PassPowK estimates the chance that all k attempts pass: C(c, k) / C(n, k).
func PassPowK(n, c, k int) float64 {
	if k <= 0 || k > n {
		return 0
	}
	return binomial(c, k) / binomial(n, k)
}

func binomial(n, k int) float64 {
	if k < 0 || k > n {
		return 0
	}
	result := 1.0
	for i := 1; i <= k; i++ {
		result = result * float64(n-k+i) / float64(i)
	}
	return result
}

func (r Report) Failed() int {
	failed := 0
	for _, c := range r.Cases {
		if c.Passed != c.Runs {
			failed++
		}
	}
	return failed
}

// WriteText prints one row per case followed by the reasons for failed runs.
func (r Report) WriteText(w io.Writer) error {
	runs := 0
	if len(r.Cases) > 0 {
		runs = r.Cases[0].Runs
	}
	source := "model " + r.Model
	if r.Mode == "replay" {
		source = "scripted model (synthetic token counts)"
	}
	if _, err := fmt.Fprintf(w, "Eval %s mode: %d cases, %d runs each, %s.\n", r.Mode, len(r.Cases), runs, source); err != nil {
		return err
	}
	table := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(table, "CASE\tRUNS\tPASS\tpass@k\tpass^k\tREQ\tTOOLS\tIN_TOK\tOUT_TOK\tSTATUS")
	for _, c := range r.Cases {
		status := "pass"
		if c.Passed != c.Runs {
			status = "FAIL"
		}
		fmt.Fprintf(table, "%s\t%d\t%d\t%.2f\t%.2f\t%.1f\t%.1f\t%d\t%d\t%s\n", c.Name, c.Runs, c.Passed, c.PassAtK, c.PassPowK, c.MeanRequests, c.MeanToolCalls, c.Usage.InputTokens, c.Usage.OutputTokens, status)
	}
	if err := table.Flush(); err != nil {
		return err
	}
	for _, c := range r.Cases {
		for _, result := range c.Results {
			for _, o := range result.Outcomes {
				if !o.Passed {
					if _, err := fmt.Fprintf(w, "  %s run %d: %s: %s\n", c.Name, result.Run, o.Grader, o.Detail); err != nil {
						return err
					}
				}
			}
		}
	}
	summary := "All cases passed."
	if failed := r.Failed(); failed > 0 {
		summary = fmt.Sprintf("%d of %d cases failed.", failed, len(r.Cases))
	}
	_, err := fmt.Fprintln(w, strings.TrimSpace(summary))
	return err
}

// WriteJSON stores the complete report for later comparison.
func (r Report) WriteJSON(path string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0600)
}
