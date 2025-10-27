package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	archiveDir = "/archives"
	port       = "8080"
)

func main() {
	// Verify archives exist
	files, err := filepath.Glob(filepath.Join(archiveDir, "*.tar.gz"))
	if err != nil || len(files) == 0 {
		log.Fatal("No archives found in ", archiveDir)
	}
	log.Printf("Found %d archives", len(files))

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

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<html><head><title>kubectl-oadp Downloads</title></head><body>")
	fmt.Fprintf(w, "<h1>kubectl-oadp Binary Downloads</h1>")
	fmt.Fprintf(w, "<p>Download pre-built binaries for your platform:</p><ul>")

	for _, file := range files {
		name := filepath.Base(file)
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		size := float64(info.Size()) / (1024 * 1024) // MB
		fmt.Fprintf(w, `<li><a href="/download/%s">%s</a> (%.2f MB)</li>`, name, name, size)
	}

	fmt.Fprintf(w, "</ul>")
	fmt.Fprintf(w, "<h3>Installation:</h3>")
	fmt.Fprintf(w, "<pre>tar -xzf kubectl-oadp_*.tar.gz\n")
	fmt.Fprintf(w, "chmod +x kubectl-oadp\n")
	fmt.Fprintf(w, "sudo mv kubectl-oadp /usr/local/bin/</pre>")
}

func downloadBinary(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Base(r.URL.Path[len("/download/"):])

	// Security: ensure filename is just the archive name
	if filepath.Dir(filename) != "." || !strings.HasSuffix(filename, ".tar.gz") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(archiveDir, filename)

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "Archive not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Type", "application/gzip")

	http.ServeFile(w, r, filePath)
	log.Printf("Downloaded: %s from %s", filename, r.RemoteAddr)
}
