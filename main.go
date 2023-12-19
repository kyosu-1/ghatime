package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

type Repository struct {
	Name      string `json:"name"`
	TotalTime int64  `json:"totalTime"`
	AvgTime   int64  `json:"avgTime"`
	Jobs      []Job  `json:"jobs"`
}

type Job struct {
	Name      string `json:"name"`
	TotalTime int64  `json:"totalTime"`
	AvgTime   int64  `json:"avgTime"`
	RunCount  int    `json:"runCount"`
}

type WorkflowRun struct {
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Name      string `json:"name"`
}

type WorkflowRuns struct {
	TotalCount   int           `json:"total_count"`
	WorkflowRuns []WorkflowRun `json:"workflow_runs"`
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "ghatime",
		Short: "ghatime is a tool to analyze GitHub Actions execution time in an organization",
		Run:   analyzeExecutionTime,
	}

	rootCmd.Flags().StringP("org", "o", "", "organization name (required)")
	rootCmd.Flags().String("from", "", "start date (YYYY-MM-DD format)")
	rootCmd.Flags().String("to", "", "end date (YYYY-MM-DD format)")
	rootCmd.MarkFlagRequired("org")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func analyzeExecutionTime(cmd *cobra.Command, args []string) {
	org, _ := cmd.Flags().GetString("org")
	from, _ := cmd.Flags().GetString("from")
	to, _ := cmd.Flags().GetString("to")

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "no GitHub token provided")
		os.Exit(1)
	}

	dateRange, err := parseDateRange(from, to)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	repos, err := getRepositories(org, token)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to fetch repositories:", err)
		os.Exit(1)
	}

	eg, ctx := errgroup.WithContext(context.Background())
	orgReposChan := make(chan Repository)

	for _, repo := range repos {
		repo := repo
		eg.Go(func() error {
			runs, err := getWorkflowRuns(ctx, org, repo.Name, token, dateRange)
			if err != nil {
				return err
			}

			var totalTime, totalCount int64
			jobs := make(map[string]*Job)

			for _, run := range runs {
				startTime, _ := time.Parse(time.RFC3339, run.CreatedAt)
				endTime, _ := time.Parse(time.RFC3339, run.UpdatedAt)
				duration := int64(endTime.Sub(startTime).Seconds())

				totalTime += duration
				totalCount++

				if job, exists := jobs[run.Name]; exists {
					job.TotalTime += duration
					job.RunCount++
				} else {
					jobs[run.Name] = &Job{
						Name:      run.Name,
						TotalTime: duration,
						RunCount:  1,
					}
				}
			}

			for _, job := range jobs {
				if job.RunCount > 0 {
					job.AvgTime = job.TotalTime / int64(job.RunCount)
				}
			}

			if totalCount > 0 {
				repo.TotalTime = totalTime
				repo.AvgTime = totalTime / totalCount
				repo.Jobs = convertMapToSlice(jobs)
				orgReposChan <- repo
			}
			return nil
		})
	}

	go func() {
		err := eg.Wait()
		close(orgReposChan)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error occurred during fetching workflow runs:", err)
		}
	}()

	var orgRepos []Repository
	for repo := range orgReposChan {
		orgRepos = append(orgRepos, repo)
	}

	sort.Slice(orgRepos, func(i, j int) bool {
		return orgRepos[i].TotalTime > orgRepos[j].TotalTime
	})

	output := struct {
		Org   string       `json:"org"`
		Repos []Repository `json:"repos"`
	}{
		Org:   org,
		Repos: orgRepos,
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(data))
}

func getRepositories(org, token string) ([]Repository, error) {
	var allRepos []Repository
	page := 1

	for {
		repos, err := fetchRepositoriesPage(org, token, page)
		if err != nil {
			return nil, err
		}
		if len(repos) == 0 {
			break
		}

		allRepos = append(allRepos, repos...)
		page++
	}

	return allRepos, nil
}

func fetchRepositoriesPage(org, token string, page int) ([]Repository, error) {
	url := fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=100&page=%d", org, page)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "token "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var repos []Repository
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, err
	}

	return repos, nil
}

func getWorkflowRuns(ctx context.Context, org, repo, token string, dateRage string) ([]WorkflowRun, error) {
	var allRuns []WorkflowRun
	page := 1

	for {
		runs, err := fetchWorkflowRunsPage(ctx, org, repo, token, dateRage, page)
		if err != nil {
			return nil, err
		}
		fmt.Println("Found", len(runs), "workflow runs")
		if len(runs) == 0 {
			break
		}

		allRuns = append(allRuns, runs...)
		page++
	}

	return allRuns, nil
}

func fetchWorkflowRunsPage(ctx context.Context, org, repo, token, dateRange string, page int) ([]WorkflowRun, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/actions/runs", org, repo)
	query := fmt.Sprintf("status=completed&per_page=100&page=%d&created=%s", page, dateRange)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = query
	req.Header.Set("Authorization", "token "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var runs WorkflowRuns
	if err := json.NewDecoder(resp.Body).Decode(&runs); err != nil {
		return nil, err
	}

	return runs.WorkflowRuns, nil
}

func convertMapToSlice(jobsMap map[string]*Job) []Job {
	jobs := make([]Job, 0, len(jobsMap))
	for _, job := range jobsMap {
		jobs = append(jobs, *job)
	}
	return jobs
}

func parseDateRange(fromStr, toStr string) (dateRange string, err error) {
	const layout = "2006-01-02"

	if fromStr == "" && toStr == "" {
		// If both dates are empty, set default range to the last week
		fromStr = time.Now().AddDate(0, 0, -7).Format(layout)
		toStr = time.Now().Format(layout)
	} else {
		// Validate individual dates if provided
		if fromStr != "" {
			_, err := time.Parse(layout, fromStr)
			if err != nil {
				return "", fmt.Errorf("invalid start date format: please use YYYY-MM-DD")
			}
		} else {
			fromStr = time.Now().AddDate(0, 0, -7).Format(layout) // Default to one week ago
		}

		if toStr != "" {
			_, err := time.Parse(layout, toStr)
			if err != nil {
				return "", fmt.Errorf("invalid end date format: please use YYYY-MM-DD")
			}
		} else {
			toStr = time.Now().Format(layout) // Default to current date
		}
	}

	// Check if the date range is valid
	from, _ := time.Parse(layout, fromStr)
	to, _ := time.Parse(layout, toStr)
	if from.After(to) {
		return "", fmt.Errorf("the start date must be before the end date")
	}

	return fromStr + ".." + toStr, nil
}
