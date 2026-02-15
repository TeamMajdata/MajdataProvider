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
	"strconv"
	"strings"

	"golang.org/x/image/draw"
)

const (
	chartAPIPrefix    = "/api/maichart/"
	weekCacheControl  = "public,max-age=604800"
	maxThumbnailPixel = 512
)

type chartAssetRequest struct {
	folder string
	asset  string
}

func logServe(path, file, extra string) {
	if extra == "" {
		log.Printf("serve path=%s file=%s", path, file)
		return
	}
	log.Printf("serve path=%s file=%s %s", path, file, extra)
}

func logServeError(path string, err error) {
	log.Printf("serve_error path=%s err=%v", path, err)
}

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
		logServe(manifestPath, filepath.Base(manifestPath), "")
	}
}

func folderForChartID(chartID string) (string, error) {
	folderBytes, err := base64.RawURLEncoding.DecodeString(chartID)
	if err != nil {
		return "", err
	}
	folder := filepath.Clean(filepath.FromSlash(string(folderBytes)))
	if folder == "." || folder == "" {
		return "", fmt.Errorf("empty folder name")
	}
	if filepath.IsAbs(folder) || folder == ".." || strings.HasPrefix(folder, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid folder name")
	}
	return folder, nil
}

func parseChartAssetRequest(path string) (chartAssetRequest, error) {
	if !strings.HasPrefix(path, chartAPIPrefix) {
		return chartAssetRequest{}, fmt.Errorf("invalid api prefix")
	}
	rest := strings.TrimPrefix(path, chartAPIPrefix)
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return chartAssetRequest{}, fmt.Errorf("invalid api path")
	}
	folder, err := folderForChartID(parts[0])
	if err != nil {
		return chartAssetRequest{}, err
	}
	return chartAssetRequest{
		folder: folder,
		asset:  parts[1],
	}, nil
}

func serveChartFile(root string) http.HandlerFunc {
	chartsDir := filepath.Join(root, "charts")
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := parseChartAssetRequest(r.URL.Path)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		switch req.asset {
		case "chart":
			maidataPath := filepath.Join(chartsDir, req.folder, "maidata.txt")
			if err := serveFileWithHash(w, maidataPath, "text/plain; charset=utf-8", "maidata.txt", ""); err != nil {
				http.NotFound(w, r)
			}
		case "track":
			serveChartTrack(w, r, chartsDir, req.folder)
		case "image":
			fullImage := parseFullImageQuery(r.URL.Query())
			serveChartImage(w, r, chartsDir, req.folder, fullImage)
		case "video":
			serveChartVideo(w, r, chartsDir, req.folder)
		default:
			http.NotFound(w, r)
		}
	}
}

func parseFullImageQuery(values map[string][]string) bool {
	v, ok := values["fullimage"]
	if !ok || len(v) == 0 {
		return false
	}
	return v[0] == "true"
}

func serveChartTrack(w http.ResponseWriter, r *http.Request, chartsDir, folder string) {
	trackPath, err := firstExistingPath(chartsDir, folder, []string{"track.mp3"})
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := serveMediaFileWithHash(w, trackPath, "audio/mp3", "track.mp3"); err != nil {
		logServeError(trackPath, err)
	}
}

func serveChartVideo(w http.ResponseWriter, r *http.Request, chartsDir, folder string) {
	videoPath, err := firstExistingPath(chartsDir, folder, []string{"bg.mp4", "pv.mp4"})
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := serveMediaFileWithHash(w, videoPath, "video/mp4", "bg.mp4"); err != nil {
		logServeError(videoPath, err)
	}
}

func serveMediaFileWithHash(w http.ResponseWriter, path, contentType, downloadName string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", downloadName))
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Cache-Control", weekCacheControl)
	if err := writeHashedBody(w, data); err != nil {
		return err
	}
	logServe(path, downloadName, "")
	return nil
}

func serveChartImage(w http.ResponseWriter, r *http.Request, chartsDir, folder string, fullImage bool) {
	fullPath, err := firstExistingPath(chartsDir, folder, []string{"bg.jpg", "bg.png"})
	if err != nil {
		http.NotFound(w, r)
		return
	}
	logExtra := fmt.Sprintf("fullImage=%t", fullImage)
	if fullImage {
		ext := filepath.Ext(fullPath)
		contentType := mime.TypeByExtension(ext)
		if err := serveFileWithHash(w, fullPath, contentType, filepath.Base(fullPath), logExtra); err != nil {
			logServeError(fullPath, err)
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

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	newW, newH := resizeDimensions(width, height, maxThumbnailPixel)

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
	if err := serveDataWithHash(w, fullPath, filepath.Base(fullPath), out.Bytes(), contentType, logExtra); err != nil {
		logServeError(fullPath, err)
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

func serveFileWithHash(w http.ResponseWriter, path, contentType, downloadName, extra string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return serveDataWithHash(w, path, downloadName, data, contentType, extra)
}

func serveDataWithHash(w http.ResponseWriter, path, downloadName string, data []byte, contentType, extra string) error {
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if downloadName != "" {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", downloadName))
	}
	if err := writeHashedBody(w, data); err != nil {
		return err
	}
	logServe(path, downloadName, extra)
	return nil
}

func writeHashedBody(w http.ResponseWriter, data []byte) error {
	sum := sha256.Sum256(data)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("hash", base64.StdEncoding.EncodeToString(sum[:]))
	_, err := w.Write(data)
	return err
}
