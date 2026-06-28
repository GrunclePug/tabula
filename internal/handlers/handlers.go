package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/grunclepug/tabula/internal/database"
	"github.com/grunclepug/tabula/web"
)

type Handler struct {
	db       *database.DB
	docsRoot string
	dataDir  string
	version  string
	tmpl     *template.Template
}

func New(db *database.DB, docsRoot, dataDir, version string) *Handler {
	tmpl := template.Must(template.ParseFS(web.FS, "templates/*.html"))
	return &Handler{
		db:       db,
		docsRoot: docsRoot,
		dataDir:  dataDir,
		version:  version,
		tmpl:     tmpl,
	}
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	groups, err := h.db.GetGroups()
	if err != nil {
		log.Printf("Failed to load groups: %v", err)
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}

	for i := range groups {
		groups[i].Editable = false
		for j := range groups[i].Nodes {
			groups[i].Nodes[j].Editable = false
		}
	}

	data := struct {
		Version  string
		Groups   []database.Group
		Editable bool
	}{
		Version:  h.version,
		Groups:   groups,
		Editable: false,
	}

	h.tmpl.ExecuteTemplate(w, "index.html", data)
}

func (h *Handler) Manage(w http.ResponseWriter, r *http.Request) {
	groups, err := h.db.GetGroups()
	if err != nil {
		log.Printf("Failed to load groups for management: %v", err)
		http.Error(w, "Failed to load dashboard data", http.StatusInternalServerError)
		return
	}

	for i := range groups {
		groups[i].Editable = true
		for j := range groups[i].Nodes {
			groups[i].Nodes[j].Editable = true
		}
	}

	data := struct {
		Version  string
		Groups   []database.Group
		Editable bool
	}{
		Version:  h.version,
		Groups:   groups,
		Editable: true,
	}

	h.tmpl.ExecuteTemplate(w, "index.html", data)
}

func (h *Handler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if name == "" {
		name = "New Group"
	}
	
	group, err := h.db.CreateGroup(name)
	if err != nil {
		log.Printf("Failed to create group: %v", err)
		http.Error(w, "Failed to create group", http.StatusInternalServerError)
		return
	}
	group.Editable = true
	
	h.tmpl.ExecuteTemplate(w, "group", group)
}

func (h *Handler) ToggleGroupCollapse(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	group, err := h.db.ToggleGroupCollapse(id)
	if err != nil {
		log.Printf("Failed to toggle group collapse: %v", err)
		http.Error(w, "Failed to update group state", http.StatusInternalServerError)
		return
	}
	
	manage := r.URL.Query().Get("manage") == "true"
	group.Editable = manage
	for i := range group.Nodes {
		group.Nodes[i].Editable = manage
	}

	h.tmpl.ExecuteTemplate(w, "group", group)
}

func (h *Handler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteGroup(id); err != nil {
		log.Printf("Failed to delete group: %v", err)
		http.Error(w, "Failed to delete group", http.StatusInternalServerError)
		return
	}
	
	defer h.CleanupUnusedIcons()
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetNode(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		http.Error(w, "Invalid node ID", http.StatusBadRequest)
		return
	}

	node, err := h.db.GetNode(id)
	if err != nil {
		log.Printf("Failed to get node: %v", err)
		http.Error(w, "Node not found", http.StatusNotFound)
		return
	}
	node.Editable = true
	h.tmpl.ExecuteTemplate(w, "node", node)
}

func (h *Handler) NewNodeForm(w http.ResponseWriter, r *http.Request) {
	groupIDStr := r.URL.Query().Get("group_id")
	var groupID int
	_, _ = fmt.Sscanf(groupIDStr, "%d", &groupID)
	
	node := database.Node{
		GroupID: groupID,
	}
	
	h.tmpl.ExecuteTemplate(w, "node_form", node)
}

func (h *Handler) CreateNode(w http.ResponseWriter, r *http.Request) {
	var n database.Node
	_, _ = fmt.Sscanf(r.FormValue("group_id"), "%d", &n.GroupID)
	n.Name = r.FormValue("name")
	n.ServiceURL = r.FormValue("service_url")
	n.Description = r.FormValue("description")
	n.RepoURL = r.FormValue("repo_url")
	n.DocsPath = r.FormValue("docs_path")
	n.IconSource = r.FormValue("icon_source")
	n.HealthCheck = r.FormValue("health_check") == "true"
	n.IconType = "url"

	node, err := h.db.CreateNode(n)
	if err != nil {
		log.Printf("Failed to create node: %v", err)
		http.Error(w, "Failed to create service node", http.StatusInternalServerError)
		return
	}
	node.Editable = true

	w.Header().Set("HX-Trigger", "closeModal")
	h.tmpl.ExecuteTemplate(w, "node", node)
}

func (h *Handler) EditNodeForm(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		http.Error(w, "Invalid node ID", http.StatusBadRequest)
		return
	}

	node, err := h.db.GetNode(id)
	if err != nil {
		log.Printf("Failed to load edit form: %v", err)
		http.Error(w, "Node not found", http.StatusNotFound)
		return
	}
	h.tmpl.ExecuteTemplate(w, "node_form", node)
}

func (h *Handler) UpdateNode(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		http.Error(w, "Invalid node ID", http.StatusBadRequest)
		return
	}

	var n database.Node
	n.ID = id
	n.Name = r.FormValue("name")
	n.ServiceURL = r.FormValue("service_url")
	n.Description = r.FormValue("description")
	n.RepoURL = r.FormValue("repo_url")
	n.DocsPath = r.FormValue("docs_path")
	n.IconSource = r.FormValue("icon_source")
	n.HealthCheck = r.FormValue("health_check") == "true"

	node, err := h.db.UpdateNode(n)
	if err != nil {
		log.Printf("Failed to update node: %v", err)
		http.Error(w, "Failed to save node changes", http.StatusInternalServerError)
		return
	}

	defer h.CleanupUnusedIcons()
	node.Editable = true

	w.Header().Set("HX-Trigger", "closeModal")
	h.tmpl.ExecuteTemplate(w, "node", node)
}

func (h *Handler) DeleteNode(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		http.Error(w, "Invalid node ID", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteNode(id); err != nil {
		log.Printf("Failed to delete node: %v", err)
		http.Error(w, "Failed to delete node", http.StatusInternalServerError)
		return
	}

	defer h.CleanupUnusedIcons()
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) EditGroupNameForm(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	group, err := h.db.GetGroup(id)
	if err != nil {
		log.Printf("Failed to get group for renaming: %v", err)
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}
	h.tmpl.ExecuteTemplate(w, "group_name_form", group)
}

func (h *Handler) GetGroupName(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	group, err := h.db.GetGroup(id)
	if err != nil {
		log.Printf("Failed to get group name: %v", err)
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}
	h.tmpl.ExecuteTemplate(w, "group_name_span", group)
}

func (h *Handler) UpdateGroupName(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	var id int
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "Group name cannot be empty", http.StatusBadRequest)
		return
	}

	if err := h.db.RenameGroup(id, name); err != nil {
		log.Printf("Failed to rename group: %v", err)
		http.Error(w, "Failed to update group name", http.StatusInternalServerError)
		return
	}

	group := database.Group{
		ID:   id,
		Name: name,
	}
	h.tmpl.ExecuteTemplate(w, "group_name_span", group)
}

func (h *Handler) SaveLayout(w http.ResponseWriter, r *http.Request) {
	var updates []database.LayoutUpdate
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if err := h.db.SaveLayout(updates); err != nil {
		log.Printf("Failed to save layout: %v", err)
		http.Error(w, "Failed to save canvas layout", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ReorderNodes(w http.ResponseWriter, r *http.Request) {
	var updates []database.NodeReorderUpdate
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if err := h.db.ReorderNodes(updates); err != nil {
		log.Printf("Failed to reorder nodes: %v", err)
		http.Error(w, "Failed to save node order", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
