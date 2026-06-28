package server

import (
	"io/fs"
	"net/http"
	"path/filepath"

	"github.com/grunclepug/tabula/internal/database"
	"github.com/grunclepug/tabula/internal/handlers"
	"github.com/grunclepug/tabula/web"
)

type Server struct {
	mux      *http.ServeMux
	db       *database.DB
	docsRoot string
	dataDir  string
	version  string
}

func New(db *database.DB, docsRoot, dataDir, version string) *Server {
	s := &Server{
		mux:      http.NewServeMux(),
		db:       db,
		docsRoot: docsRoot,
		dataDir:  dataDir,
		version:  version,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	h := handlers.New(s.db, s.docsRoot, s.dataDir, s.version)
	go h.CleanupUnusedIcons()

	// Serve embedded static assets from web/static
	staticFS, err := fs.Sub(web.FS, "static")
	if err != nil {
		panic(err)
	}
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Serve persistent dynamic uploads from dataDir/uploads
	uploadsDir := filepath.Join(s.dataDir, "uploads")
	s.mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadsDir))))

	// Routes
	s.mux.HandleFunc("GET /", h.Index)
	s.mux.HandleFunc("GET /manage", h.Manage)
	s.mux.HandleFunc("POST /groups", h.CreateGroup)
	s.mux.HandleFunc("GET /groups/{id}/edit-name", h.EditGroupNameForm)
	s.mux.HandleFunc("GET /groups/{id}/name", h.GetGroupName)
	s.mux.HandleFunc("PUT /groups/{id}/name", h.UpdateGroupName)
	s.mux.HandleFunc("PUT /groups/{id}/collapse", h.ToggleGroupCollapse)
	s.mux.HandleFunc("DELETE /groups/{id}", h.DeleteGroup)

	s.mux.HandleFunc("GET /nodes/{id}", h.GetNode)
	s.mux.HandleFunc("GET /nodes/new", h.NewNodeForm)
	s.mux.HandleFunc("POST /nodes", h.CreateNode)
	s.mux.HandleFunc("GET /nodes/{id}/edit", h.EditNodeForm)
	s.mux.HandleFunc("PUT /nodes/{id}", h.UpdateNode)
	s.mux.HandleFunc("DELETE /nodes/{id}", h.DeleteNode)

	s.mux.HandleFunc("GET /docs/autocomplete", h.DocsAutocomplete)
	s.mux.HandleFunc("GET /docs/view", h.DocsView)
	s.mux.HandleFunc("POST /upload/icon", h.UploadIcon)
	s.mux.HandleFunc("POST /layout/save", h.SaveLayout)
	s.mux.HandleFunc("PUT /nodes/reorder", h.ReorderNodes)
}

func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}
