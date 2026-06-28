package handlers

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yuin/goldmark"
)

func (h *Handler) DocsAutocomplete(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		return
	}

	var results []string
	err := filepath.WalkDir(h.docsRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
			relPath, _ := filepath.Rel(h.docsRoot, path)
			if strings.Contains(strings.ToLower(relPath), strings.ToLower(query)) {
				results = append(results, relPath)
			}
		}
		if len(results) >= 10 {
			return filepath.SkipDir
		}
		return nil
	})

	if err != nil && err != filepath.SkipDir {
		log.Printf("Autocomplete walking error: %v", err)
		http.Error(w, "Failed to scan documents", http.StatusInternalServerError)
		return
	}

	for _, res := range results {
		fmt.Fprintf(w, "<li class='px-3 py-2 hover:bg-zinc-700 cursor-pointer text-sm text-zinc-300 transition-colors' onclick='selectDoc(\"%s\")'>%s</li>", res, res)
	}
}

func (h *Handler) DocsView(w http.ResponseWriter, r *http.Request) {
	docPath := r.URL.Query().Get("path")
	if docPath == "" {
		http.Error(w, "Missing path parameter", http.StatusBadRequest)
		return
	}

	// Resolve the absolute path and secure it against directory traversal
	cleanPath := filepath.Clean(docPath)
	fullPath := filepath.Join(h.docsRoot, cleanPath)
	rel, err := filepath.Rel(h.docsRoot, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		log.Printf("Security alert: Attempted directory traversal on path: %s", docPath)
		http.Error(w, "Forbidden path access", http.StatusForbidden)
		return
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		log.Printf("Failed to read document %s: %v", fullPath, err)
		http.Error(w, "Document not found or unreadable", http.StatusNotFound)
		return
	}

	var buf bytes.Buffer
	if err := goldmark.Convert(content, &buf); err != nil {
		log.Printf("Failed to convert markdown to HTML: %v", err)
		http.Error(w, "Failed to parse document content", http.StatusInternalServerError)
		return
	}

	data := struct {
		Title   string
		Content template.HTML
	}{
		Title:   filepath.Base(cleanPath),
		Content: template.HTML(buf.String()),
	}

	h.tmpl.ExecuteTemplate(w, "modal", data)
}

func (h *Handler) UploadIcon(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20) // 10MB
	file, handler, err := r.FormFile("icon")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	uploadsDir := filepath.Join(h.dataDir, "uploads")
	os.MkdirAll(uploadsDir, 0755)

	// Generate a unique filename to prevent overwriting
	ext := filepath.Ext(handler.Filename)
	if ext == "" {
		ext = ".png"
	}
	
	uniqueName := fmt.Sprintf("%d", time.Now().UnixNano())
	randomBytes := make([]byte, 8)
	if _, randErr := rand.Read(randomBytes); randErr == nil {
		uniqueName = hex.EncodeToString(randomBytes)
	}
	filename := uniqueName + ext

	dst, err := os.Create(filepath.Join(uploadsDir, filename))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "/uploads/%s", filename)
}

func (h *Handler) CleanupUnusedIcons() {
	uploadsDir := filepath.Join(h.dataDir, "uploads")
	files, err := os.ReadDir(uploadsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Printf("Failed to read uploads dir for cleanup: %v", err)
		return
	}

	rows, err := h.db.Query("SELECT DISTINCT icon_source FROM nodes WHERE icon_source LIKE '/uploads/%'")
	if err != nil {
		log.Printf("Failed to query active icon sources for cleanup: %v", err)
		return
	}
	defer rows.Close()

	activeIcons := make(map[string]bool)
	for rows.Next() {
		var src string
		if err := rows.Scan(&src); err == nil {
			parts := strings.Split(src, "/")
			if len(parts) > 0 {
				filename := parts[len(parts)-1]
				activeIcons[filename] = true
			}
		}
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		filename := file.Name()
		if !activeIcons[filename] {
			filePath := filepath.Join(uploadsDir, filename)
			log.Printf("Deleting unused icon file: %s", filePath)
			if err := os.Remove(filePath); err != nil {
				log.Printf("Failed to delete unused icon file %s: %v", filePath, err)
			}
		}
	}
}
