package release

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"time"
)

type (
	ciRun struct {
		ID         int64  `json:"id"`
		Attempt    int    `json:"run_attempt"`
		SHA        string `json:"head_sha"`
		Branch     string `json:"head_branch"`
		Event      string `json:"event"`
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
		URL        string `json:"html_url"`
	}
	ciJob struct {
		Name       string `json:"name"`
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
	}
	ciEvidence struct {
		Source string           `json:"source"`
		Runs   map[string]ciRun `json:"runs"`
	}
)

func (p publisher) waitForCI(ctx context.Context, source string) (ciEvidence, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Minute)
	defer cancel()
	for {
		evidence, pending, err := p.checkCI(ctx, source)
		if err != nil {
			return evidence, err
		}
		if !pending {
			return evidence, nil
		}
		fmt.Printf("Waiting for CI on %s\n", source)
		if err := pollPause(ctx); err != nil {
			return evidence, fmt.Errorf("CI did not complete: %w", err)
		}
	}
}

func (p publisher) checkCI(ctx context.Context, source string) (ciEvidence, bool, error) {
	evidence := ciEvidence{Source: source, Runs: map[string]ciRun{}}
	pending := false
	for _, workflow := range []string{"test.yml", "codeql.yml"} {
		data, err := p.run(ctx, "gh", "api", "repos/"+repository+"/actions/workflows/"+workflow+"/runs?head_sha="+source+"&event=push&per_page=100")
		if err != nil {
			return evidence, false, err
		}
		var result struct {
			Runs []ciRun `json:"workflow_runs"`
		}
		if err := json.Unmarshal(data, &result); err != nil {
			return evidence, false, err
		}
		run, err := latestRun(result.Runs, source)
		if err != nil {
			return evidence, false, fmt.Errorf("%s: %w", workflow, err)
		}
		if run.Status != "completed" {
			pending = true
			continue
		}
		if run.Conclusion != "success" {
			return evidence, false, fmt.Errorf("%s run %d is %s", workflow, run.ID, run.Conclusion)
		}
		data, err = p.run(ctx, "gh", "api", fmt.Sprintf("repos/%s/actions/runs/%d/attempts/%d/jobs?per_page=100", repository, run.ID, run.Attempt), "--paginate", "--slurp")
		if err != nil {
			return evidence, false, err
		}
		var pages []struct {
			Jobs []ciJob `json:"jobs"`
		}
		if err := json.Unmarshal(data, &pages); err != nil {
			return evidence, false, err
		}
		var jobs []ciJob
		for _, page := range pages {
			jobs = append(jobs, page.Jobs...)
		}
		if err := validateJobs(workflow, jobs); err != nil {
			return evidence, false, err
		}
		evidence.Runs[workflow] = run
	}
	return evidence, pending, nil
}

func latestRun(runs []ciRun, source string) (ciRun, error) {
	var latest ciRun
	for _, run := range runs {
		if run.SHA == source && run.Branch == "main" && run.Event == "push" && run.ID > latest.ID {
			latest = run
		}
	}
	if latest.ID == 0 {
		return latest, errors.New("no main push CI evidence for selected source")
	}
	return latest, nil
}

func validateJobs(workflow string, jobs []ciJob) error {
	if len(jobs) == 0 {
		return errors.New("CI run has no jobs")
	}
	ready := workflow != "test.yml"
	for _, job := range jobs {
		if job.Status != "completed" || job.Conclusion != "success" {
			return fmt.Errorf("CI job %s is not successful", job.Name)
		}
		if job.Name == "Release eligibility" {
			ready = true
		}
	}
	if !ready {
		return errors.New("CI run has no release eligibility gate")
	}
	return nil
}
