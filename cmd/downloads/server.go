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
	binaryDir    = getEnv("ARCHIVE_DIR", "/archives")
	port         = getEnv("PORT", "8080")
	pageTemplate = template.Must(template.ParseFS(templateFS, "templates/index.html"))
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type binaryFile struct {
	Name     string
	Size     float64
	OS       string
	Arch     string
	Checksum string
}

func main() {
	files, err := discoverBinaries()
	if err != nil || len(files) == 0 {
		log.Fatal("No binaries found in ", binaryDir)
	}
	log.Printf("Found %d binaries", len(files))

	staticContent, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal("Failed to load static files: ", err)
	}
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticContent))))
	http.HandleFunc("/", listBinaries)
	http.HandleFunc("/download/", downloadBinary)

	log.Printf("Starting server on port %s", port)
	log.Printf("Serving binaries from %s", binaryDir)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

// discoverBinaries finds kubectl-oadp binaries (excluding .sha256 and LICENSE files).
func discoverBinaries() ([]string, error) {
	entries, err := os.ReadDir(binaryDir)
	if err != nil {
		return nil, err
	}
	var binaries []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasSuffix(name, ".sha256") || name == "LICENSE" {
			continue
		}
		if strings.HasPrefix(name, "kubectl-oadp_") {
			binaries = append(binaries, filepath.Join(binaryDir, name))
		}
	}
	return binaries, nil
}

func listBinaries(w http.ResponseWriter, r *http.Request) {
	files, err := discoverBinaries()
	if err != nil {
		http.Error(w, "Error listing binaries", http.StatusInternalServerError)
		return
	}

	hasLicense := false
	if _, err := os.Stat(filepath.Join(binaryDir, "LICENSE")); err == nil {
		hasLicense = true
	}

	var linuxFiles, darwinFiles, windowsFiles []binaryFile
	for _, file := range files {
		name := filepath.Base(file)
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		size := float64(info.Size()) / (1024 * 1024)
		osName, arch := parsePlatform(name)
		checksum := readChecksum(file + ".sha256")
		bf := binaryFile{Name: name, Size: size, OS: osName, Arch: arch, Checksum: checksum}
		switch osName {
		case "linux":
			linuxFiles = append(linuxFiles, bf)
		case "darwin":
			darwinFiles = append(darwinFiles, bf)
		case "windows":
			windowsFiles = append(windowsFiles, bf)
		default:
			linuxFiles = append(linuxFiles, bf)
		}
	}

	data := struct {
		LinuxFiles   []binaryFile
		DarwinFiles  []binaryFile
		WindowsFiles []binaryFile
		HasLicense   bool
	}{linuxFiles, darwinFiles, windowsFiles, hasLicense}

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
	name := strings.TrimSuffix(filename, ".exe")
	parts := strings.Split(name, "_")
	if len(parts) >= 3 {
		return parts[len(parts)-2], parts[len(parts)-1]
	}
	return "unknown", "unknown"
}

func downloadBinary(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Base(r.URL.Path[len("/download/"):])

	if filepath.Dir(filename) != "." {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	// Allow downloading LICENSE or any kubectl-oadp binary
	if filename != "LICENSE" && !strings.HasPrefix(filename, "kubectl-oadp_") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(binaryDir, filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	if filename == "LICENSE" {
		w.Header().Set("Content-Type", "text/plain")
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	http.ServeFile(w, r, filePath)
	log.Printf("Downloaded: %s from %s", filename, r.RemoteAddr)
}
