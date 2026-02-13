package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
)

func serveChartList(root string) http.HandlerFunc {
	manifestPath := filepath.Join(root, "manifest.json")
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			http.Error(w, "manifest not found", http.StatusNotFound)
			return
		}
		var manifest struct {
			Charts []ParsedChart `json:"charts"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			http.Error(w, "invalid manifest", http.StatusInternalServerError)
			return
		}
		out, err := json.MarshalIndent(manifest.Charts, "", "  ")
		if err != nil {
			http.Error(w, "failed to serialize", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if _, err := w.Write(out); err != nil {
			http.Error(w, "failed to write response", http.StatusInternalServerError)
			return
		}
	}
}

func folderForChartID(chartID string) (string, error) {
	folderBytes, err := base64.RawURLEncoding.DecodeString(chartID)
	if err != nil {
		return "", err
	}
	folder := string(folderBytes)
	if folder == "" {
		return "", fmt.Errorf("empty folder name")
	}
	if strings.Contains(folder, "/") || strings.Contains(folder, "\\") || strings.Contains(folder, "..") {
		return "", fmt.Errorf("invalid folder name")
	}
	return folder, nil
}

func serveChartFile(root string) http.HandlerFunc {
	chartsDir := filepath.Join(root, "charts")
	const prefix = "/api/maichart/"
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if !strings.HasPrefix(path, prefix) {
			http.NotFound(w, r)
			return
		}
		rest := strings.TrimPrefix(path, prefix)
		parts := strings.Split(rest, "/")
		if len(parts) != 2 {
			http.NotFound(w, r)
			return
		}
		folder, err := folderForChartID(parts[0])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		asset := parts[1]
		switch asset {
		case "chart":
			maidataPath := filepath.Join(chartsDir, folder, "maidata.txt")
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			if err := serveFileWithHash(w, maidataPath, "text/plain; charset=utf-8"); err != nil {
				http.NotFound(w, r)
			}
		case "track":
			serveFirstExisting(w, r, chartsDir, folder, []string{"track.mp3", "track.ogg"})
		case "image":
			fullImage := r.URL.Query().Get("fullImage") == "true"
			serveChartImage(w, r, chartsDir, folder, fullImage)
		case "video":
			serveFirstExisting(w, r, chartsDir, folder, []string{"pv.mp4", "pv.webm"})
		default:
			http.NotFound(w, r)
		}
	}
}

func serveFirstExisting(w http.ResponseWriter, r *http.Request, chartsDir, folder string, candidates []string) {
	for _, name := range candidates {
		path := filepath.Join(chartsDir, folder, name)
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		ext := filepath.Ext(path)
		contentType := mime.TypeByExtension(ext)
		if err := serveFileWithHash(w, path, contentType); err != nil {
			log.Printf("failed to serve file %s: %v", path, err)
		}
		return
	}
	http.NotFound(w, r)
}

func serveChartImage(w http.ResponseWriter, r *http.Request, chartsDir, folder string, fullImage bool) {
	fullPath, err := firstExistingPath(chartsDir, folder, []string{"bg.jpg", "bg.png"})
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if fullImage {
		ext := filepath.Ext(fullPath)
		contentType := mime.TypeByExtension(ext)
		if err := serveFileWithHash(w, fullPath, contentType); err != nil {
			log.Printf("failed to serve image %s: %v", fullPath, err)
		}
		return
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		http.Error(w, "failed to read file", http.StatusInternalServerError)
		return
	}
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		http.Error(w, "failed to decode image", http.StatusInternalServerError)
		return
	}

	const maxDim = 512
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	newW, newH := resizeDimensions(width, height, maxDim)
	if newW == width && newH == height {
		contentType := mime.TypeByExtension(filepath.Ext(fullPath))
		if err := serveDataWithHash(w, fullPath, data, contentType); err != nil {
			log.Printf("failed to serve image %s: %v", fullPath, err)
		}
		return
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	var out bytes.Buffer
	switch format {
	case "png":
		if err := png.Encode(&out, dst); err != nil {
			http.Error(w, "failed to encode png", http.StatusInternalServerError)
			return
		}
	default:
		if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: 85}); err != nil {
			http.Error(w, "failed to encode jpeg", http.StatusInternalServerError)
			return
		}
		format = "jpeg"
	}

	contentType := "image/jpeg"
	if format == "png" {
		contentType = "image/png"
	}
	if err := serveDataWithHash(w, fullPath+" (thumbnail)", out.Bytes(), contentType); err != nil {
		log.Printf("failed to serve thumbnail %s: %v", fullPath, err)
	}
}

func resizeDimensions(width, height, maxDim int) (int, int) {
	if width <= maxDim && height <= maxDim {
		return width, height
	}
	if width >= height {
		newW := maxDim
		newH := int(float64(height) * float64(maxDim) / float64(width))
		if newH < 1 {
			newH = 1
		}
		return newW, newH
	}
	newH := maxDim
	newW := int(float64(width) * float64(maxDim) / float64(height))
	if newW < 1 {
		newW = 1
	}
	return newW, newH
}

func firstExistingPath(chartsDir, folder string, candidates []string) (string, error) {
	for _, name := range candidates {
		path := filepath.Join(chartsDir, folder, name)
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			return path, nil
		}
	}
	return "", fmt.Errorf("no file found")
}

func serveFileWithHash(w http.ResponseWriter, path, contentType string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return serveDataWithHash(w, filepath.Base(path), data, contentType)
}

func serveDataWithHash(w http.ResponseWriter, filename string, data []byte, contentType string) error {
	sum := sha256.Sum256(data)
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if filename != "" {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	}
	hashValue := base64.StdEncoding.EncodeToString(sum[:])
	w.Header().Set("hash", hashValue)
	log.Printf("response headers for %s: %v", filename, w.Header())
	n, err := w.Write(data)
	if err == nil {
		log.Printf("served %s bytes=%d hash=%s", filename, n, hashValue)
	}
	return err
}
