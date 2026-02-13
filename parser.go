package main

import (
	"bufio"
	"strconv"
	"strings"
)

type ParsedChart struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Artist      string    `json:"artist"`
	Designer    string    `json:"designer"`
	Description string    `json:"description"`
	Timestamp   string    `json:"timestamp"`
	Hash        string    `json:"hash"`
	Levels      []*string `json:"levels"`
}

func parseMaidata(data []byte) (ParsedChart, error) {
	// Read line by line to extract metadata fields.
	chart := ParsedChart{
		Levels: make([]*string, 7),
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		line = strings.TrimPrefix(line, "\uFEFF")
		switch {
		case strings.HasPrefix(line, "&title="):
			chart.Title = strings.TrimPrefix(line, "&title=")
		case strings.HasPrefix(line, "&artist="):
			chart.Artist = strings.TrimPrefix(line, "&artist=")
		case strings.HasPrefix(line, "&des="):
			chart.Designer = strings.TrimPrefix(line, "&des=")
		case strings.HasPrefix(line, "&lv_"):
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			levelStr := strings.TrimPrefix(key, "&lv_")
			levelIndex, err := strconv.Atoi(levelStr)
			if err != nil || levelIndex < 1 || levelIndex > 7 {
				continue
			}
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			chart.Levels[levelIndex-1] = &value
		}
	}
	if err := scanner.Err(); err != nil {
		return ParsedChart{}, err
	}
	return chart, nil
}
