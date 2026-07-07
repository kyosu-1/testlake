// Package ghapi fetches the current job's start time from the GitHub API.
package ghapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func JobStartedAt(baseURL, token, repo string, runID int64, attempt int32, jobName string) (time.Time, error) {
	url := fmt.Sprintf("%s/repos/%s/actions/runs/%d/attempts/%d/jobs?per_page=100", baseURL, repo, runID, attempt)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return time.Time{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return time.Time{}, fmt.Errorf("github api: %s", resp.Status)
	}
	var body struct {
		Jobs []struct {
			Name      string    `json:"name"`
			StartedAt time.Time `json:"started_at"`
		} `json:"jobs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return time.Time{}, err
	}
	for _, j := range body.Jobs {
		if j.Name == jobName {
			return j.StartedAt.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("job %q not found in run %d", jobName, runID)
}
