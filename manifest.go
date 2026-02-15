package main

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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

	manifest := struct {
		Charts []ParsedChart `json:"charts"`
	}{
		Charts: []ParsedChart{},
	}

	candidateFolders := []string{}
	err = filepath.WalkDir(chartsDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() != "maidata.txt" {
			return nil
		}
		dir := filepath.Dir(path)
		rel, err := filepath.Rel(chartsDir, dir)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		candidateFolders = append(candidateFolders, rel)
		return nil
	})
	if err != nil {
		return err
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
	results := make(chan chartResult, len(candidateFolders))
	var wg sync.WaitGroup

	for _, folderName := range candidateFolders {
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

	go func() {
		wg.Wait()
		close(results)
	}()

	total := len(candidateFolders)
	loaded := 0
	if total == 0 {
		fmt.Print("\rCharts loaded: 0/0\n")
	}
	for result := range results {
		loaded++
		fmt.Printf("\rLoading charts: %d/%d", loaded, total)
		if result.err != nil {
			fmt.Println()
			return result.err
		}
		manifest.Charts = append(manifest.Charts, result.chart)
	}
	if total > 0 {
		fmt.Printf("\rCharts loaded: %d/%d\n", loaded, total)
	}

	out, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	manifestPath := filepath.Join(root, "manifest.json")
	if err := os.WriteFile(manifestPath, out, 0o644); err != nil {
		return err
	}
	log.Printf("build path=%s file=%s", manifestPath, "manifest.json")
	return nil
}

func chartIDForFolder(folder string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(folder))
}
