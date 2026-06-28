package database

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func New(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if err := runMigrations(db); err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

func runMigrations(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS groups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		pos_x INTEGER NOT NULL DEFAULT 0,
		pos_y INTEGER NOT NULL DEFAULT 0,
		width INTEGER NOT NULL DEFAULT 300,
		height INTEGER NOT NULL DEFAULT 200,
		collapsed BOOLEAN NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS nodes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		group_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		service_url TEXT,
		repo_url TEXT,
		health_check BOOLEAN NOT NULL DEFAULT 0,
		docs_path TEXT,
		icon_source TEXT,
		icon_type TEXT,
		sort_order INTEGER NOT NULL DEFAULT 0,
		FOREIGN KEY(group_id) REFERENCES groups(id) ON DELETE CASCADE
	);
	`
	_, err := db.Exec(schema)
	if err != nil {
		log.Printf("Migration failed: %v", err)
		return err
	}

	// Dynamic migration: add sort_order column to nodes if it doesn't exist yet
	_, _ = db.Exec("ALTER TABLE nodes ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0")

	return nil
}

func (db *DB) GetGroups() ([]Group, error) {
	rows, err := db.Query("SELECT id, name, pos_x, pos_y, width, height, collapsed FROM groups")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []Group
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.Name, &g.PosX, &g.PosY, &g.Width, &g.Height, &g.Collapsed); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}

	for i := range groups {
		nodes, err := db.GetNodesForGroup(groups[i].ID)
		if err != nil {
			return nil, err
		}
		groups[i].Nodes = nodes
	}

	return groups, nil
}

func (db *DB) GetNodesForGroup(groupID int) ([]Node, error) {
	rows, err := db.Query("SELECT id, group_id, name, COALESCE(description, ''), service_url, COALESCE(repo_url, ''), health_check, COALESCE(docs_path, ''), COALESCE(icon_source, ''), COALESCE(icon_type, '') FROM nodes WHERE group_id = ? ORDER BY sort_order ASC, id ASC", groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var n Node
		if err := rows.Scan(&n.ID, &n.GroupID, &n.Name, &n.Description, &n.ServiceURL, &n.RepoURL, &n.HealthCheck, &n.DocsPath, &n.IconSource, &n.IconType); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, nil
}

func (db *DB) CreateGroup(name string) (Group, error) {
	res, err := db.Exec("INSERT INTO groups (name, pos_x, pos_y, width, height, collapsed) VALUES (?, ?, ?, ?, ?, ?)", name, 100, 100, 300, 200, 0)
	if err != nil {
		return Group{}, err
	}
	id, _ := res.LastInsertId()
	return Group{
		ID:        int(id),
		Name:      name,
		PosX:      100,
		PosY:      100,
		Width:     300,
		Height:    200,
		Collapsed: false,
		Nodes:     []Node{},
	}, nil
}

func (db *DB) GetGroup(id int) (Group, error) {
	var g Group
	err := db.QueryRow("SELECT id, name, pos_x, pos_y, width, height, collapsed FROM groups WHERE id = ?", id).
		Scan(&g.ID, &g.Name, &g.PosX, &g.PosY, &g.Width, &g.Height, &g.Collapsed)
	if err != nil {
		return Group{}, err
	}
	nodes, err := db.GetNodesForGroup(g.ID)
	if err != nil {
		return Group{}, err
	}
	g.Nodes = nodes
	return g, nil
}

func (db *DB) ToggleGroupCollapse(id int) (Group, error) {
	_, err := db.Exec("UPDATE groups SET collapsed = NOT collapsed WHERE id = ?", id)
	if err != nil {
		return Group{}, err
	}
	return db.GetGroup(id)
}

func (db *DB) DeleteGroup(id int) error {
	_, err := db.Exec("DELETE FROM groups WHERE id = ?", id)
	return err
}

type LayoutUpdate struct {
	ID     int `json:"id"`
	PosX   int `json:"pos_x"`
	PosY   int `json:"pos_y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

func (db *DB) SaveLayout(updates []LayoutUpdate) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("UPDATE groups SET pos_x = ?, pos_y = ?, width = ?, height = ? WHERE id = ?")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, u := range updates {
		if _, err := stmt.Exec(u.PosX, u.PosY, u.Width, u.Height, u.ID); err != nil {
			log.Printf("Failed to update group %d layout: %v", u.ID, err)
		}
	}

	return tx.Commit()
}

func (db *DB) GetNode(id int) (Node, error) {
	var n Node
	err := db.QueryRow("SELECT id, group_id, name, COALESCE(description, ''), service_url, COALESCE(repo_url, ''), health_check, COALESCE(docs_path, ''), COALESCE(icon_source, ''), COALESCE(icon_type, '') FROM nodes WHERE id = ?", id).
		Scan(&n.ID, &n.GroupID, &n.Name, &n.Description, &n.ServiceURL, &n.RepoURL, &n.HealthCheck, &n.DocsPath, &n.IconSource, &n.IconType)
	return n, err
}

func (db *DB) CreateNode(n Node) (Node, error) {
	res, err := db.Exec("INSERT INTO nodes (group_id, name, description, service_url, repo_url, health_check, docs_path, icon_source, icon_type) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		n.GroupID, n.Name, n.Description, n.ServiceURL, n.RepoURL, n.HealthCheck, n.DocsPath, n.IconSource, n.IconType)
	if err != nil {
		return Node{}, err
	}
	id, _ := res.LastInsertId()
	n.ID = int(id)
	return n, nil
}

func (db *DB) UpdateNode(n Node) (Node, error) {
	_, err := db.Exec("UPDATE nodes SET name = ?, service_url = ?, description = ?, repo_url = ?, health_check = ?, docs_path = ?, icon_source = ? WHERE id = ?",
		n.Name, n.ServiceURL, n.Description, n.RepoURL, n.HealthCheck, n.DocsPath, n.IconSource, n.ID)
	if err != nil {
		return Node{}, err
	}
	return db.GetNode(n.ID)
}

func (db *DB) DeleteNode(id int) error {
	_, err := db.Exec("DELETE FROM nodes WHERE id = ?", id)
	return err
}

func (db *DB) RenameGroup(id int, name string) error {
	_, err := db.Exec("UPDATE groups SET name = ? WHERE id = ?", name, id)
	return err
}

type NodeReorderUpdate struct {
	ID      int `json:"id"`
	GroupID int `json:"group_id"`
}

func (db *DB) ReorderNodes(updates []NodeReorderUpdate) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("UPDATE nodes SET group_id = ?, sort_order = ? WHERE id = ?")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for i, u := range updates {
		if _, err := stmt.Exec(u.GroupID, i, u.ID); err != nil {
			log.Printf("Failed to update node %d reorder: %v", u.ID, err)
		}
	}

	return tx.Commit()
}
