package database

type Group struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	PosX      int    `json:"pos_x"`
	PosY      int    `json:"pos_y"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Collapsed bool   `json:"collapsed"`
	Nodes     []Node `json:"nodes"`
	Editable  bool   `json:"-"`
}

type Node struct {
	ID          int    `json:"id"`
	GroupID     int    `json:"group_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ServiceURL  string `json:"service_url"`
	RepoURL     string `json:"repo_url"`
	HealthCheck bool   `json:"health_check"`
	DocsPath    string `json:"docs_path"`
	IconSource  string `json:"icon_source"`
	IconType    string `json:"icon_type"` // 'url', 'upload', 'clipboard'
	Editable    bool   `json:"-"`
}
