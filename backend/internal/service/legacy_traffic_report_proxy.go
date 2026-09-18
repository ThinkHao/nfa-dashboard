package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"nfa-dashboard/config"
	"nfa-dashboard/internal/model"
)

const (
	trafficReportLegacyEngineVersion = "legacy-proxy-v1"
	legacyPollInterval               = 2 * time.Second
	legacyPollTimeout                = 30 * time.Minute
)

type legacyJobArtifact struct {
	Filename string `json:"filename"`
}

type legacyJobResponse struct {
	ID             string                 `json:"id"`
	Status         string                 `json:"status"`
	ResolvedWindow map[string]interface{} `json:"resolved_window"`
	ResolvedParams map[string]interface{} `json:"resolved_params"`
	Artifacts      []legacyJobArtifact    `json:"artifacts"`
	ErrorMessage   string                 `json:"error_message"`
}

func legacyTaskID(params []byte) (uint64, bool) {
	var raw map[string]interface{}
	if err := json.Unmarshal(params, &raw); err != nil {
		return 0, false
	}
	legacy, ok := raw["legacy_nfatool"].(map[string]interface{})
	if !ok {
		return 0, false
	}
	value, ok := legacy["source_task_id"]
	if !ok {
		return 0, false
	}
	switch v := value.(type) {
	case float64:
		if v > 0 && v == float64(uint64(v)) {
			return uint64(v), true
		}
	case json.Number:
		id, err := strconv.ParseUint(string(v), 10, 64)
		return id, err == nil && id > 0
	case string:
		id, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64)
		return id, err == nil && id > 0
	}
	return 0, false
}

func (s *trafficReportService) executeLegacyProxy(ctx context.Context, task *model.TrafficReportTask, run *model.TrafficReportRun) (int64, map[string]interface{}, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(config.AppConfig.TrafficReport.LegacyBaseURL), "/")
	if baseURL == "" {
		return 0, nil, fmt.Errorf("legacy nfatool compatibility bridge is not configured")
	}
	legacyID, ok := legacyTaskID(task.Params)
	if !ok {
		return 0, nil, fmt.Errorf("legacy task marker is missing source_task_id")
	}
	client := &http.Client{Timeout: 30 * time.Second}
	jobID, err := s.startLegacyJob(ctx, client, baseURL, legacyID)
	if err != nil {
		return 0, nil, err
	}
	job, err := s.waitLegacyJob(ctx, client, baseURL, jobID)
	if err != nil {
		return 0, nil, err
	}
	if strings.ToLower(job.Status) != "succeeded" && strings.ToLower(job.Status) != "success" {
		if job.ErrorMessage != "" {
			return 0, nil, fmt.Errorf("legacy nfatool job %s failed: %s", jobID, job.ErrorMessage)
		}
		return 0, nil, fmt.Errorf("legacy nfatool job %s finished with status %s", jobID, job.Status)
	}

	dir := filepath.Join(config.GetTrafficReportStorageDir(), run.ID)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return 0, nil, err
	}
	var rowCount int64
	for _, artifact := range job.Artifacts {
		name := filepath.Base(strings.TrimSpace(artifact.Filename))
		if name == "" || name == "." || name == string(filepath.Separator) {
			continue
		}
		path := filepath.Join(dir, name)
		if err := s.downloadLegacyArtifact(ctx, client, baseURL, jobID, artifact.Filename, path); err != nil {
			return 0, nil, err
		}
		if strings.EqualFold(filepath.Ext(name), ".csv") {
			if n, err := countCSVRows(path); err == nil {
				rowCount += n
			}
		}
		st, err := os.Stat(path)
		if err != nil {
			return 0, nil, err
		}
		if err := s.repo.CreateArtifact(&model.TrafficReportArtifact{
			RunID: run.ID, FileName: name, MediaType: legacyMediaType(name), StoragePath: path, FileSize: st.Size(),
		}); err != nil {
			return 0, nil, err
		}
	}

	summary := map[string]interface{}{
		"engine_version":         trafficReportLegacyEngineVersion,
		"legacy_job_id":          jobID,
		"legacy_task_id":         legacyID,
		"legacy_status":          job.Status,
		"legacy_artifact_count":  len(job.Artifacts),
		"legacy_resolved_window": job.ResolvedWindow,
		"legacy_resolved_params": job.ResolvedParams,
	}
	return rowCount, summary, nil
}

func (s *trafficReportService) startLegacyJob(ctx context.Context, client *http.Client, baseURL string, taskID uint64) (string, error) {
	path := fmt.Sprintf("%s/api/tasks/%d/run", baseURL, taskID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, nil)
	if err != nil {
		return "", err
	}
	s.applyLegacyAuth(req)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("start legacy nfatool task: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("start legacy nfatool task: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload struct {
		JobID string `json:"job_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode legacy nfatool start response: %w", err)
	}
	if strings.TrimSpace(payload.JobID) == "" {
		return "", fmt.Errorf("legacy nfatool returned an empty job id")
	}
	return payload.JobID, nil
}

func (s *trafficReportService) waitLegacyJob(ctx context.Context, client *http.Client, baseURL, jobID string) (*legacyJobResponse, error) {
	deadline := time.NewTimer(legacyPollTimeout)
	defer deadline.Stop()
	ticker := time.NewTicker(legacyPollInterval)
	defer ticker.Stop()
	for {
		job, err := s.getLegacyJob(ctx, client, baseURL, jobID)
		if err != nil {
			return nil, err
		}
		status := strings.ToLower(strings.TrimSpace(job.Status))
		if status == "succeeded" || status == "success" || status == "failed" || status == "cancelled" {
			return job, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, fmt.Errorf("legacy nfatool job %s timed out", jobID)
		case <-ticker.C:
		}
	}
}

func (s *trafficReportService) getLegacyJob(ctx context.Context, client *http.Client, baseURL, jobID string) (*legacyJobResponse, error) {
	path := fmt.Sprintf("%s/api/jobs/%s", baseURL, url.PathEscape(jobID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	s.applyLegacyAuth(req)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("read legacy nfatool job: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("read legacy nfatool job: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var job legacyJobResponse
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, fmt.Errorf("decode legacy nfatool job: %w", err)
	}
	return &job, nil
}

func (s *trafficReportService) downloadLegacyArtifact(ctx context.Context, client *http.Client, baseURL, jobID, filename, destination string) error {
	path := fmt.Sprintf("%s/api/jobs/%s/download?file=%s", baseURL, url.PathEscape(jobID), url.QueryEscape(filename))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	s.applyLegacyAuth(req)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download legacy nfatool artifact %q: %w", filename, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("download legacy nfatool artifact %q: HTTP %d: %s", filename, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	f, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func (s *trafficReportService) applyLegacyAuth(req *http.Request) {
	if key := strings.TrimSpace(config.AppConfig.TrafficReport.LegacyAPIKey); key != "" {
		req.Header.Set("X-API-KEY", key)
	}
}

func countCSVRows(path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	var rows int64
	for {
		_, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return rows, err
		}
		rows++
	}
	if rows > 0 {
		rows--
	}
	return rows, nil
}

func legacyMediaType(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".csv":
		return "text/csv; charset=utf-8"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".json":
		return "application/json"
	case ".txt", ".log":
		return "text/plain; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}
