package publisher

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	maxBucketSize     = 40 * 1024 * 1024
	maxBucketFiles    = 2000
	uploadConcurrency = 3
	maxUploadRetries  = 5
	maxDeployRetries  = 3
	cfBaseURL         = "https://api.cloudflare.com/client/v4"
)

type Publisher struct {
	APIToken    string
	AccountID   string
	ProjectName string
	BaseURL     string
	AssetsURL   string
}

type fileEntry struct {
	relPath     string
	absPath     string
	hash        string
	size        int64
	contentType string
}

type cfResponse struct {
	Success bool              `json:"success"`
	Result  json.RawMessage   `json:"result"`
	Errors  []json.RawMessage `json:"errors"`
}

func (p *Publisher) baseURL() string {
	if p.BaseURL != "" {
		return p.BaseURL
	}
	return cfBaseURL
}

func (p *Publisher) assetsURL() string {
	if p.AssetsURL != "" {
		return p.AssetsURL
	}
	return cfBaseURL
}

func (p *Publisher) Publish(dir string) error {
	if err := p.ensureProject(); err != nil {
		return fmt.Errorf("ensuring project: %w", err)
	}
	files, err := p.collectFiles(dir)
	if err != nil {
		return fmt.Errorf("collecting files: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("no files found in %s", dir)
	}
	fmt.Printf("Collected %d files for upload\n", len(files))

	jwt, err := p.getUploadToken()
	if err != nil {
		return fmt.Errorf("getting upload token: %w", err)
	}

	allHashes := make([]string, len(files))
	hashToFile := map[string]*fileEntry{}
	for i, f := range files {
		allHashes[i] = f.hash
		hashToFile[f.hash] = files[i]
	}

	missing, err := p.checkMissing(jwt, allHashes)
	if err != nil {
		return fmt.Errorf("checking missing: %w", err)
	}
	fmt.Printf("Uploading %d new files (%d already cached)\n", len(missing), len(files)-len(missing))

	if len(missing) > 0 {
		var toUpload []*fileEntry
		for _, h := range missing {
			if f, ok := hashToFile[h]; ok {
				toUpload = append(toUpload, f)
			}
		}
		if err := p.uploadFiles(jwt, toUpload); err != nil {
			return fmt.Errorf("uploading files: %w", err)
		}
	}

	if err := p.upsertHashes(jwt, allHashes); err != nil {
		return fmt.Errorf("upserting hashes: %w", err)
	}

	manifest := map[string]string{}
	for _, f := range files {
		manifest["/"+f.relPath] = f.hash
	}
	url, err := p.createDeployment(manifest)
	if err != nil {
		return fmt.Errorf("creating deployment: %w", err)
	}
	fmt.Printf("Deployment successful: %s\n", url)
	return nil
}

func (p *Publisher) ensureProject() error {
	url := fmt.Sprintf("%s/accounts/%s/pages/projects/%s", p.baseURL(), p.AccountID, p.ProjectName)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+p.APIToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var cr cfResponse
	json.NewDecoder(resp.Body).Decode(&cr)
	if cr.Success {
		return nil
	}

	fmt.Printf("Creating Cloudflare Pages project: %s\n", p.ProjectName)
	body, _ := json.Marshal(map[string]string{"name": p.ProjectName, "production_branch": "production"})
	createURL := fmt.Sprintf("%s/accounts/%s/pages/projects", p.baseURL(), p.AccountID)
	req, _ = http.NewRequest("POST", createURL, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+p.APIToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&cr)
	if !cr.Success {
		return fmt.Errorf("failed to create project: %s", cr.Errors)
	}
	return nil
}

func (p *Publisher) collectFiles(dir string) ([]*fileEntry, error) {
	var files []*fileEntry
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		hash, err := HashFile(path)
		if err != nil {
			return err
		}
		ct := mime.TypeByExtension(filepath.Ext(path))
		if ct == "" {
			ct = "application/octet-stream"
		}
		files = append(files, &fileEntry{relPath: rel, absPath: path, hash: hash, size: info.Size(), contentType: ct})
		return nil
	})
	return files, err
}

func (p *Publisher) getUploadToken() (string, error) {
	url := fmt.Sprintf("%s/accounts/%s/pages/projects/%s/upload-token", p.baseURL(), p.AccountID, p.ProjectName)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+p.APIToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var cr cfResponse
	json.NewDecoder(resp.Body).Decode(&cr)
	if !cr.Success {
		return "", fmt.Errorf("failed to get upload token: %s", cr.Errors)
	}
	var result struct {
		JWT string `json:"jwt"`
	}
	json.Unmarshal(cr.Result, &result)
	return result.JWT, nil
}

func (p *Publisher) checkMissing(jwt string, hashes []string) ([]string, error) {
	body, _ := json.Marshal(map[string][]string{"hashes": hashes})
	url := fmt.Sprintf("%s/pages/assets/check-missing", p.assetsURL())
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var cr cfResponse
	json.NewDecoder(resp.Body).Decode(&cr)
	if !cr.Success {
		return nil, fmt.Errorf("check-missing failed: %s", cr.Errors)
	}
	var missing []string
	json.Unmarshal(cr.Result, &missing)
	return missing, nil
}

func (p *Publisher) uploadFiles(jwt string, files []*fileEntry) error {
	type bucket struct {
		files []*fileEntry
		size  int64
	}
	var buckets []bucket
	current := bucket{}
	for _, f := range files {
		if len(current.files) >= maxBucketFiles || current.size+f.size > maxBucketSize {
			if len(current.files) > 0 {
				buckets = append(buckets, current)
			}
			current = bucket{}
		}
		current.files = append(current.files, f)
		current.size += f.size
	}
	if len(current.files) > 0 {
		buckets = append(buckets, current)
	}

	sem := make(chan struct{}, uploadConcurrency)
	var mu sync.Mutex
	var firstErr error
	var wg sync.WaitGroup
	for i, b := range buckets {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, b bucket) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := p.uploadBucket(jwt, b.files); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("bucket %d: %w", idx, err)
				}
				mu.Unlock()
			}
		}(i, b)
	}
	wg.Wait()
	return firstErr
}

func (p *Publisher) uploadBucket(jwt string, files []*fileEntry) error {
	type uploadItem struct {
		Key      string            `json:"key"`
		Value    string            `json:"value"`
		Metadata map[string]string `json:"metadata"`
		Base64   bool              `json:"base64"`
	}
	var payload []uploadItem
	for _, f := range files {
		content, err := os.ReadFile(f.absPath)
		if err != nil {
			return err
		}
		payload = append(payload, uploadItem{
			Key:      f.hash,
			Value:    base64.StdEncoding.EncodeToString(content),
			Metadata: map[string]string{"contentType": f.contentType},
			Base64:   true,
		})
	}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/pages/assets/upload", p.assetsURL())
	var lastErr error
	for attempt := 0; attempt < maxUploadRetries; attempt++ {
		req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+jwt)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(math.Pow(2, float64(attempt))) * time.Second)
			continue
		}
		var cr cfResponse
		json.NewDecoder(resp.Body).Decode(&cr)
		resp.Body.Close()
		if cr.Success {
			return nil
		}
		lastErr = fmt.Errorf("upload failed: %s", cr.Errors)
		if resp.StatusCode >= 500 {
			time.Sleep(time.Duration(math.Pow(2, float64(attempt))) * time.Second)
			continue
		}
		return lastErr
	}
	return fmt.Errorf("upload failed after %d retries: %w", maxUploadRetries, lastErr)
}

func (p *Publisher) upsertHashes(jwt string, hashes []string) error {
	body, _ := json.Marshal(map[string][]string{"hashes": hashes})
	url := fmt.Sprintf("%s/pages/assets/upsert-hashes", p.assetsURL())
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var cr cfResponse
	json.NewDecoder(resp.Body).Decode(&cr)
	if !cr.Success {
		return fmt.Errorf("upsert-hashes failed: %s", cr.Errors)
	}
	return nil
}

func (p *Publisher) createDeployment(manifest map[string]string) (string, error) {
	manifestJSON, _ := json.Marshal(manifest)
	var lastErr error
	for attempt := 0; attempt < maxDeployRetries; attempt++ {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormField("manifest")
		if err != nil {
			return "", err
		}
		part.Write(manifestJSON)
		writer.Close()
		url := fmt.Sprintf("%s/accounts/%s/pages/projects/%s/deployments", p.baseURL(), p.AccountID, p.ProjectName)
		req, _ := http.NewRequest("POST", url, &body)
		req.Header.Set("Authorization", "Bearer "+p.APIToken)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(math.Pow(2, float64(attempt))) * time.Second)
			continue
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var cr cfResponse
		json.Unmarshal(respBody, &cr)
		if cr.Success {
			var result struct {
				URL string `json:"url"`
			}
			json.Unmarshal(cr.Result, &result)
			return result.URL, nil
		}
		lastErr = fmt.Errorf("deployment failed: %s", string(respBody))
		if resp.StatusCode >= 500 {
			time.Sleep(time.Duration(math.Pow(2, float64(attempt))) * time.Second)
			continue
		}
		return "", lastErr
	}
	return "", fmt.Errorf("deployment failed after %d retries: %w", maxDeployRetries, lastErr)
}
