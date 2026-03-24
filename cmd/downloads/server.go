package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

var (
	archiveDir   = getEnv("ARCHIVE_DIR", "/archives")
	port         = getEnv("PORT", "8080")
	pageTemplate = template.Must(template.ParseFS(templateFS, "templates/index.html"))
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type archiveFile struct {
	Name     string
	Size     float64
	OS       string
	Arch     string
	Checksum string
}

func main() {
	files, err := filepath.Glob(filepath.Join(archiveDir, "*.tar.gz"))
	if err != nil || len(files) == 0 {
		log.Fatal("No archives found in ", archiveDir)
	}
	log.Printf("Found %d archives", len(files))

	staticContent, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal("Failed to load static files: ", err)
	}
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticContent))))
	http.HandleFunc("/", listBinaries)
	http.HandleFunc("/download/", downloadBinary)

	log.Printf("Starting server on port %s", port)
	log.Printf("Serving archives from %s", archiveDir)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func listBinaries(w http.ResponseWriter, r *http.Request) {
	files, err := filepath.Glob(filepath.Join(archiveDir, "*.tar.gz"))
	if err != nil {
		http.Error(w, "Error listing archives", http.StatusInternalServerError)
		return
	}

	var linuxFiles, darwinFiles, windowsFiles []archiveFile
	for _, file := range files {
		name := filepath.Base(file)
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		size := float64(info.Size()) / (1024 * 1024)
		osName, arch := parsePlatform(name)
		checksum := readChecksum(file + ".sha256")
		af := archiveFile{Name: name, Size: size, OS: osName, Arch: arch, Checksum: checksum}
		switch osName {
		case "linux":
			linuxFiles = append(linuxFiles, af)
		case "darwin":
			darwinFiles = append(darwinFiles, af)
		case "windows":
			windowsFiles = append(windowsFiles, af)
		default:
			linuxFiles = append(linuxFiles, af)
		}
	}

	data := struct {
		LinuxFiles   []archiveFile
		DarwinFiles  []archiveFile
		WindowsFiles []archiveFile
	}{linuxFiles, darwinFiles, windowsFiles}

	w.Header().Set("Content-Type", "text/html")
	if err := pageTemplate.Execute(w, data); err != nil {
		log.Printf("Template error: %v", err)
	}
}

func readChecksum(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	fields := strings.Fields(string(data))
	if len(fields) > 0 {
		return fields[0]
	}
	return ""
}

func parsePlatform(filename string) (string, string) {
	name := strings.TrimSuffix(filename, ".tar.gz")
	parts := strings.Split(name, "_")
	if len(parts) >= 3 {
		return parts[len(parts)-2], parts[len(parts)-1]
	}
	return "unknown", "unknown"
}

func downloadBinary(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Base(r.URL.Path[len("/download/"):])

	if filepath.Dir(filename) != "." || !strings.HasSuffix(filename, ".tar.gz") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(archiveDir, filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Archive not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Type", "application/gzip")

	http.ServeFile(w, r, filePath)
	log.Printf("Downloaded: %s from %s", filename, r.RemoteAddr)
}
