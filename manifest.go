package main

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

func buildManifest(root string) error {
	chartsDir := filepath.Join(root, "charts")
	info, err := os.Stat(chartsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("charts is not a directory")
	}

	entries, err := os.ReadDir(chartsDir)
	if err != nil {
		return err
	}

	manifest := struct {
		Charts []ParsedChart `json:"charts"`
	}{
		Charts: []ParsedChart{},
	}

	dirNames := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			dirNames = append(dirNames, entry.Name())
		}
	}

	type chartResult struct {
		chart ParsedChart
		err   error
	}

	workerLimit := runtime.NumCPU()
	if workerLimit < 1 {
		workerLimit = 1
	}
	sem := make(chan struct{}, workerLimit)
	results := make(chan chartResult, len(dirNames))
	var wg sync.WaitGroup

	for _, folderName := range dirNames {
		wg.Add(1)
		go func(folder string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			maidataPath := filepath.Join(chartsDir, folder, "maidata.txt")
			data, err := os.ReadFile(maidataPath)
			if err != nil {
				results <- chartResult{err: err}
				return
			}
			info, err := os.Stat(maidataPath)
			if err != nil {
				results <- chartResult{err: err}
				return
			}
			parsed, err := parseMaidata(data)
			if err != nil {
				results <- chartResult{err: err}
				return
			}
			parsed.ID = chartIDForFolder(folder)
			parsed.Description = ""
			parsed.Timestamp = info.ModTime().Format(time.RFC3339Nano)
			sum := md5.Sum(data)
			parsed.Hash = base64.StdEncoding.EncodeToString(sum[:])
			results <- chartResult{chart: parsed}
		}(folderName)
	}

	wg.Wait()
	for i := 0; i < len(dirNames); i++ {
		result := <-results
		if result.err != nil {
			return result.err
		}
		manifest.Charts = append(manifest.Charts, result.chart)
	}

	out, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "manifest.json"), out, 0o644)
}

func chartIDForFolder(folder string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(folder))
}
