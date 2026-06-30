// ZMark backend (Go) — multi-tenant bookmark/navigation API backed by MySQL.
//
// Data is scoped by owner_id: "global" (admin-curated, shown to everyone by
// default) or a user id (that user's personal nav). Tables are prefixed
// (default "onenav_") and created/upgraded idempotently, so it only ever ADDS
// to an existing database.
//
// Config from env; local dev can use backend-go/.env (gitignored, loaded at
// startup, real env wins). DB_PASSWORD is REQUIRED. See .env.example.
//
//	DB_HOST (127.0.0.1) DB_PORT (3306) DB_NAME (zmark)
//	DB_USERNAME (root) DB_PASSWORD (required)
//	PORT (8787) TABLE_PREFIX (onenav_) ADMIN_USER (admin) ADMIN_PASS (admin123)
//	AUTH_TTL_HOURS (168)
//
// Run:  cd backend-go && go run .
package main

import (
	"crypto/rand"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

//go:embed seed.json
var seedJSON []byte

const globalOwner = "global"

// ---------- models ----------

type Category struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Icon   string `json:"icon"`
	Weight int    `json:"weight"`
}

type Link struct {
	ID          string `json:"id"`
	CategoryID  string `json:"categoryId"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Clicks      int    `json:"clicks"`
	Weight      int    `json:"weight"`
}

type seedData struct {
	Categories []Category `json:"categories"`
	Links      []Link     `json:"links"`
}

type session struct {
	userID    string
	username  string
	role      string
	expiresAt time.Time
}

type UserInfo struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	CreatedAt string `json:"createdAt"`
}

type InviteInfo struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	Note      string `json:"note"`
	MaxUses   int    `json:"maxUses"` // 0 = unlimited
	UsedCount int    `json:"usedCount"`
	ExpiresAt string `json:"expiresAt"` // "" = never
	Disabled  bool   `json:"disabled"`
	GrantRole string `json:"grantRole"` // role granted to users who register with this code
	Status    string `json:"status"`    // active | used | expired | disabled
	CreatedAt string `json:"createdAt"`
}

type InviteUse struct {
	Username string `json:"username"`
	UsedAt   string `json:"usedAt"`
}

// ---------- server ----------

type Server struct {
	db        *sql.DB
	prefix    string
	tCats     string
	tLinks    string
	tUsers    string
	tInvites  string
	adminUser string
	adminPass string
	authTTL   time.Duration

	mu     sync.Mutex
	tokens map[string]session
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func loadDotEnv(paths ...string) {
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			k = strings.TrimSpace(k)
			v = strings.Trim(strings.TrimSpace(v), `"'`)
			if _, exists := os.LookupEnv(k); !exists {
				os.Setenv(k, v)
			}
		}
		return
	}
}

func main() {
	loadDotEnv(".env", "backend-go/.env")

	host := env("DB_HOST", "127.0.0.1")
	port := env("DB_PORT", "3306")
	name := env("DB_NAME", "zmark")
	user := env("DB_USERNAME", "root")
	pass := env("DB_PASSWORD", "")
	if pass == "" {
		log.Fatalf("DB_PASSWORD is required (set it via env or backend-go/.env — see .env.example)")
	}
	prefix := env("TABLE_PREFIX", "onenav_")
	appPort := env("PORT", "8787")
	ttlHours, _ := strconv.Atoi(env("AUTH_TTL_HOURS", "168"))
	if ttlHours <= 0 {
		ttlHours = 168
	}

	s := &Server{
		prefix:    prefix,
		tCats:     prefix + "categories",
		tLinks:    prefix + "links",
		tUsers:    prefix + "users",
		tInvites:  prefix + "invites",
		adminUser: env("ADMIN_USER", "admin"),
		adminPass: env("ADMIN_PASS", "admin123"),
		authTTL:   time.Duration(ttlHours) * time.Hour,
		tokens:    map[string]session{},
	}

	if err := s.connect(host, port, name, user, pass); err != nil {
		log.Fatalf("database connect failed: %v", err)
	}
	defer s.db.Close()

	if err := s.migrate(); err != nil {
		log.Fatalf("migrate failed: %v", err)
	}
	if err := s.seedIfEmpty(); err != nil {
		log.Fatalf("seed failed: %v", err)
	}
	go s.sweepTokens()

	srv := &http.Server{
		Addr:              ":" + appPort,
		Handler:           maxBody(5<<20, withCORS(s.routes())),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("ZMark Go API listening on http://localhost:%s (db=%s, prefix=%s)", appPort, name, prefix)
	log.Printf("  admin user: %s (override with ADMIN_USER / ADMIN_PASS)", s.adminUser)
	if s.adminPass == "admin123" {
		log.Printf("  WARNING: using the default admin password 'admin123' — set ADMIN_PASS before exposing this server")
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// ---------- db connect + migrate ----------

func (s *Server) connect(host, port, name, user, pass string) error {
	params := "charset=utf8mb4&parseTime=true&loc=Local"
	rootDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/?%s", user, pass, host, port, params)
	root, err := sql.Open("mysql", rootDSN)
	if err != nil {
		return err
	}
	root.SetConnMaxLifetime(time.Minute)
	if err := root.Ping(); err != nil {
		root.Close()
		return fmt.Errorf("ping: %w", err)
	}
	if _, err := root.Exec("CREATE DATABASE IF NOT EXISTS `" + name + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		root.Close()
		return fmt.Errorf("create database: %w", err)
	}
	root.Close()

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s", user, pass, host, port, name, params)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(3 * time.Minute)
	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}
	s.db = db
	return nil
}

func (s *Server) columnExists(table, col string) (bool, error) {
	var n int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? AND COLUMN_NAME=?",
		table, col,
	).Scan(&n)
	return n > 0, err
}

func (s *Server) ensureColumn(table, col, alter string) error {
	ok, err := s.columnExists(table, col)
	if err != nil || ok {
		return err
	}
	_, err = s.db.Exec(alter)
	return err
}

func (s *Server) migrate() error {
	stmts := []string{
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id VARCHAR(64) PRIMARY KEY,
			owner_id VARCHAR(64) NOT NULL DEFAULT 'global',
			name VARCHAR(255) NOT NULL,
			icon VARCHAR(100) NOT NULL DEFAULT '',
			weight INT NOT NULL DEFAULT 0,
			create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_%scat_owner (owner_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`, s.tCats, s.prefix),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id VARCHAR(64) PRIMARY KEY,
			owner_id VARCHAR(64) NOT NULL DEFAULT 'global',
			category_id VARCHAR(64) NOT NULL,
			title VARCHAR(255) NOT NULL,
			url TEXT NOT NULL,
			description TEXT,
			icon VARCHAR(255) NOT NULL DEFAULT '',
			clicks INT NOT NULL DEFAULT 0,
			weight INT NOT NULL DEFAULT 0,
			create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_%slink_owner (owner_id),
			INDEX idx_%slink_cat (category_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`, s.tLinks, s.prefix, s.prefix),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id VARCHAR(64) PRIMARY KEY,
			username VARCHAR(64) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			role VARCHAR(16) NOT NULL DEFAULT 'user',
			invite_code VARCHAR(32) NOT NULL DEFAULT '',
			create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`, s.tUsers),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id VARCHAR(64) PRIMARY KEY,
			code VARCHAR(32) NOT NULL UNIQUE,
			note VARCHAR(255) NOT NULL DEFAULT '',
			max_uses INT NOT NULL DEFAULT 1,
			used_count INT NOT NULL DEFAULT 0,
			expires_at TIMESTAMP NULL DEFAULT NULL,
			disabled TINYINT NOT NULL DEFAULT 0,
			grant_role VARCHAR(16) NOT NULL DEFAULT 'user',
			created_by VARCHAR(64) NOT NULL DEFAULT '',
			create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`, s.tInvites),
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	// Idempotent upgrades for pre-existing tables (adds columns, backfilling
	// existing rows to the column DEFAULT — i.e. owner_id='global', role='user').
	if err := s.ensureColumn(s.tCats, "owner_id", "ALTER TABLE "+s.tCats+" ADD COLUMN owner_id VARCHAR(64) NOT NULL DEFAULT 'global'"); err != nil {
		return err
	}
	if err := s.ensureColumn(s.tLinks, "owner_id", "ALTER TABLE "+s.tLinks+" ADD COLUMN owner_id VARCHAR(64) NOT NULL DEFAULT 'global'"); err != nil {
		return err
	}
	if err := s.ensureColumn(s.tUsers, "role", "ALTER TABLE "+s.tUsers+" ADD COLUMN role VARCHAR(16) NOT NULL DEFAULT 'user'"); err != nil {
		return err
	}
	if err := s.ensureColumn(s.tUsers, "invite_code", "ALTER TABLE "+s.tUsers+" ADD COLUMN invite_code VARCHAR(32) NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.ensureColumn(s.tInvites, "grant_role", "ALTER TABLE "+s.tInvites+" ADD COLUMN grant_role VARCHAR(16) NOT NULL DEFAULT 'user'"); err != nil {
		return err
	}
	// Make sure the configured admin account always has the admin role.
	if _, err := s.db.Exec("UPDATE "+s.tUsers+" SET role='admin' WHERE username=?", s.adminUser); err != nil {
		return err
	}
	return nil
}

func (s *Server) seedIfEmpty() error {
	var userCount int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM " + s.tUsers).Scan(&userCount); err != nil {
		return err
	}
	if userCount == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte(s.adminPass), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(
			"INSERT INTO "+s.tUsers+" (id, username, password_hash, role) VALUES (?,?,?,?)",
			uuid(), s.adminUser, string(hash), "admin",
		); err != nil {
			return err
		}
		log.Printf("seeded admin user %q", s.adminUser)
	}

	var catCount int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM "+s.tCats+" WHERE owner_id=?", globalOwner).Scan(&catCount); err != nil {
		return err
	}
	if catCount > 0 {
		return nil
	}

	var sd seedData
	if err := json.Unmarshal(seedJSON, &sd); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, c := range sd.Categories {
		if _, err := tx.Exec(
			"INSERT INTO "+s.tCats+" (id, owner_id, name, icon, weight) VALUES (?,?,?,?,?)",
			c.ID, globalOwner, c.Name, c.Icon, c.Weight,
		); err != nil {
			return err
		}
	}
	for _, l := range sd.Links {
		if _, err := tx.Exec(
			"INSERT INTO "+s.tLinks+" (id, owner_id, category_id, title, url, description, icon, clicks, weight) VALUES (?,?,?,?,?,?,?,?,?)",
			l.ID, globalOwner, l.CategoryID, l.Title, l.URL, l.Description, l.Icon, l.Clicks, l.Weight,
		); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	log.Printf("seeded %d global categories, %d global links", len(sd.Categories), len(sd.Links))
	return nil
}

// ---------- helpers ----------

func uuid() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]), hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]), hex.EncodeToString(b[8:10]), hex.EncodeToString(b[10:16]))
}

// inviteCode generates a readable code (no ambiguous chars like 0/O/1/I).
func inviteCode() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 10)
	rand.Read(b)
	out := make([]byte, 10)
	for i, c := range b {
		out[i] = alphabet[int(c)%len(alphabet)]
	}
	return string(out)
}

func inviteStatus(maxUses, usedCount int, exp sql.NullTime, disabled bool) string {
	switch {
	case disabled:
		return "disabled"
	case exp.Valid && exp.Time.Before(time.Now()):
		return "expired"
	case maxUses > 0 && usedCount >= maxUses:
		return "used"
	default:
		return "active"
	}
}

func isHTTPURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) fail(w http.ResponseWriter, err error) {
	log.Printf("server error: %v", err)
	httpError(w, http.StatusInternalServerError, "internal server error")
}

func readJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return h[7:]
	}
	return ""
}

func maxBody(n int64, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, n)
		h.ServeHTTP(w, r)
	})
}

func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// currentUser resolves the bearer token to a live (non-expired) session.
func (s *Server) currentUser(r *http.Request) (sess session, ok bool) {
	tok := bearer(r)
	s.mu.Lock()
	sess, ok = s.tokens[tok]
	if ok && time.Now().After(sess.expiresAt) {
		delete(s.tokens, tok)
		ok = false
	}
	s.mu.Unlock()
	return
}

func (s *Server) auth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.currentUser(r); !ok {
			httpError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		h(w, r)
	}
}

func (s *Server) adminOnly(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, ok := s.currentUser(r)
		if !ok || sess.role != "admin" {
			httpError(w, http.StatusForbidden, "仅管理员可访问")
			return
		}
		h(w, r)
	}
}

func canEdit(owner string, sess session) bool {
	if owner == globalOwner {
		return sess.role == "admin"
	}
	return owner == sess.userID
}

// writeOwner resolves the owner a write targets from ?scope= and enforces the
// admin-only rule for the global scope. Caller must already be authenticated.
func (s *Server) writeOwner(w http.ResponseWriter, r *http.Request, sess session) (string, bool) {
	scope := r.URL.Query().Get("scope")
	if scope == "" || scope == "global" {
		if sess.role != "admin" {
			httpError(w, http.StatusForbidden, "只有管理员能配置全局导航")
			return "", false
		}
		return globalOwner, true
	}
	return sess.userID, true // "mine"
}

func (s *Server) sweepTokens() {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for range t.C {
		now := time.Now()
		s.mu.Lock()
		for tok, sess := range s.tokens {
			if now.After(sess.expiresAt) {
				delete(s.tokens, tok)
			}
		}
		s.mu.Unlock()
	}
}

func (s *Server) issueToken(sess session) string {
	token := uuid()
	sess.expiresAt = time.Now().Add(s.authTTL)
	s.mu.Lock()
	s.tokens[token] = sess
	s.mu.Unlock()
	return token
}

// ---------- routes ----------

func (s *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"ok": true, "time": time.Now()})
	})
	mux.HandleFunc("GET /api/all", s.handleAll)

	mux.HandleFunc("POST /api/register", s.handleRegister)
	mux.HandleFunc("POST /api/login", s.handleLogin)
	mux.HandleFunc("POST /api/logout", s.auth(s.handleLogout))
	mux.HandleFunc("GET /api/me", s.auth(s.handleMe))

	mux.HandleFunc("GET /api/users", s.adminOnly(s.handleListUsers))
	mux.HandleFunc("DELETE /api/users/{id}", s.adminOnly(s.handleDeleteUser))

	mux.HandleFunc("GET /api/invites", s.adminOnly(s.handleListInvites))
	mux.HandleFunc("GET /api/invites/{id}/uses", s.adminOnly(s.handleInviteUses))
	mux.HandleFunc("POST /api/invites", s.adminOnly(s.handleCreateInvite))
	mux.HandleFunc("PUT /api/invites/{id}", s.adminOnly(s.handleToggleInvite))
	mux.HandleFunc("DELETE /api/invites/{id}", s.adminOnly(s.handleDeleteInvite))

	mux.HandleFunc("POST /api/categories", s.auth(s.handleCreateCategory))
	mux.HandleFunc("PUT /api/categories/order", s.auth(s.handleReorderCategories))
	mux.HandleFunc("PUT /api/categories/{id}", s.auth(s.handleUpdateCategory))
	mux.HandleFunc("DELETE /api/categories/{id}", s.auth(s.handleDeleteCategory))

	mux.HandleFunc("POST /api/links", s.auth(s.handleCreateLink))
	mux.HandleFunc("PUT /api/links/order", s.auth(s.handleReorderLinks))
	mux.HandleFunc("POST /api/links/{id}/click", s.handleClick)
	mux.HandleFunc("PUT /api/links/{id}", s.auth(s.handleUpdateLink))
	mux.HandleFunc("DELETE /api/links/{id}", s.auth(s.handleDeleteLink))

	mux.HandleFunc("POST /api/import", s.auth(s.handleImport))
	mux.HandleFunc("POST /api/reset", s.auth(s.handleReset))

	return mux
}

// ---------- auth handlers ----------

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var body struct{ Username, Password, Code string }
	if err := readJSON(r, &body); err != nil {
		httpError(w, 400, "bad request")
		return
	}
	username := strings.TrimSpace(body.Username)
	if len(username) < 3 || len(username) > 32 {
		httpError(w, 400, "用户名需 3-32 个字符")
		return
	}
	if len(body.Password) < 6 {
		httpError(w, 400, "密码至少 6 位")
		return
	}
	code := strings.ToUpper(strings.TrimSpace(body.Code))
	if code == "" {
		httpError(w, 400, "请输入邀请码")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		s.fail(w, err)
		return
	}

	tx, err := s.db.Begin()
	if err != nil {
		s.fail(w, err)
		return
	}
	defer tx.Rollback()

	// Lock + validate the invite so concurrent registrations can't over-use it.
	var invID, grantRole string
	var maxUses, usedCount int
	var disabled bool
	var exp sql.NullTime
	err = tx.QueryRow("SELECT id, max_uses, used_count, expires_at, disabled, grant_role FROM "+s.tInvites+" WHERE code=? FOR UPDATE", code).
		Scan(&invID, &maxUses, &usedCount, &exp, &disabled, &grantRole)
	if err == sql.ErrNoRows {
		httpError(w, 400, "邀请码无效")
		return
	} else if err != nil {
		s.fail(w, err)
		return
	}
	if disabled {
		httpError(w, 400, "邀请码已停用")
		return
	}
	if exp.Valid && exp.Time.Before(time.Now()) {
		httpError(w, 400, "邀请码已过期")
		return
	}
	if maxUses > 0 && usedCount >= maxUses {
		httpError(w, 400, "邀请码已用尽")
		return
	}

	var exists int
	tx.QueryRow("SELECT COUNT(*) FROM "+s.tUsers+" WHERE username=?", username).Scan(&exists)
	if exists > 0 {
		httpError(w, 409, "用户名已存在")
		return
	}

	if grantRole != "admin" {
		grantRole = "user"
	}
	id := uuid()
	if _, err := tx.Exec(
		"INSERT INTO "+s.tUsers+" (id, username, password_hash, role, invite_code) VALUES (?,?,?,?,?)",
		id, username, string(hash), grantRole, code,
	); err != nil {
		s.fail(w, err)
		return
	}
	if _, err := tx.Exec("UPDATE "+s.tInvites+" SET used_count = used_count + 1 WHERE id=?", invID); err != nil {
		s.fail(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		s.fail(w, err)
		return
	}

	token := s.issueToken(session{userID: id, username: username, role: grantRole})
	writeJSON(w, 201, map[string]any{"token": token, "user": map[string]string{"username": username, "role": grantRole}})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct{ Username, Password string }
	if err := readJSON(r, &body); err != nil {
		httpError(w, 400, "bad request")
		return
	}
	var id, hash, role string
	err := s.db.QueryRow("SELECT id, password_hash, role FROM "+s.tUsers+" WHERE username=?", body.Username).Scan(&id, &hash, &role)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Password)) != nil {
		httpError(w, 401, "用户名或密码错误")
		return
	}
	token := s.issueToken(session{userID: id, username: body.Username, role: role})
	writeJSON(w, 200, map[string]any{"token": token, "user": map[string]string{"username": body.Username, "role": role}})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	delete(s.tokens, bearer(r))
	s.mu.Unlock()
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.currentUser(r)
	writeJSON(w, 200, map[string]any{"user": map[string]string{"username": sess.username, "role": sess.role}})
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query("SELECT id, username, role, create_time FROM " + s.tUsers + " ORDER BY create_time ASC, username ASC")
	if err != nil {
		s.fail(w, err)
		return
	}
	defer rows.Close()
	out := []UserInfo{}
	for rows.Next() {
		var u UserInfo
		var t time.Time
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &t); err != nil {
			s.fail(w, err)
			return
		}
		u.CreatedAt = t.Format("2006-01-02 15:04")
		out = append(out, u)
	}
	writeJSON(w, 200, out)
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.currentUser(r)
	id := r.PathValue("id")
	if id == sess.userID {
		httpError(w, 403, "不能删除自己")
		return
	}
	var uname string
	if err := s.db.QueryRow("SELECT username FROM "+s.tUsers+" WHERE id=?", id).Scan(&uname); err != nil {
		if err == sql.ErrNoRows {
			httpError(w, 404, "not found")
			return
		}
		s.fail(w, err)
		return
	}
	// Protect the bootstrap admin (ADMIN_USER); other accounts (incl. admins
	// minted via an admin-granting invite) may be removed.
	if uname == s.adminUser {
		httpError(w, 403, "不能删除主管理员账号")
		return
	}
	tx, err := s.db.Begin()
	if err != nil {
		s.fail(w, err)
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM "+s.tLinks+" WHERE owner_id=?", id); err != nil {
		s.fail(w, err)
		return
	}
	if _, err := tx.Exec("DELETE FROM "+s.tCats+" WHERE owner_id=?", id); err != nil {
		s.fail(w, err)
		return
	}
	if _, err := tx.Exec("DELETE FROM "+s.tUsers+" WHERE id=?", id); err != nil {
		s.fail(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		s.fail(w, err)
		return
	}
	// Invalidate any active sessions for the deleted user.
	s.mu.Lock()
	for tok, se := range s.tokens {
		if se.userID == id {
			delete(s.tokens, tok)
		}
	}
	s.mu.Unlock()
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// ---------- invite codes (admin) ----------

func (s *Server) handleListInvites(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query("SELECT id, code, note, max_uses, used_count, expires_at, disabled, grant_role, create_time FROM " + s.tInvites + " ORDER BY create_time DESC")
	if err != nil {
		s.fail(w, err)
		return
	}
	defer rows.Close()
	out := []InviteInfo{}
	for rows.Next() {
		var v InviteInfo
		var exp sql.NullTime
		var ct time.Time
		if err := rows.Scan(&v.ID, &v.Code, &v.Note, &v.MaxUses, &v.UsedCount, &exp, &v.Disabled, &v.GrantRole, &ct); err != nil {
			s.fail(w, err)
			return
		}
		if exp.Valid {
			v.ExpiresAt = exp.Time.Format("2006-01-02 15:04")
		}
		v.Status = inviteStatus(v.MaxUses, v.UsedCount, exp, v.Disabled)
		v.CreatedAt = ct.Format("2006-01-02 15:04")
		out = append(out, v)
	}
	writeJSON(w, 200, out)
}

func (s *Server) handleCreateInvite(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.currentUser(r)
	var body struct {
		Note          string `json:"note"`
		MaxUses       int    `json:"maxUses"`
		ExpiresInDays int    `json:"expiresInDays"`
		Role          string `json:"role"`
	}
	readJSON(r, &body)
	if body.MaxUses < 0 {
		body.MaxUses = 0 // 0 = unlimited
	}
	grantRole := body.Role
	if grantRole != "admin" {
		grantRole = "user"
	}
	var expVal any
	var expStr string
	if body.ExpiresInDays > 0 {
		t := time.Now().Add(time.Duration(body.ExpiresInDays) * 24 * time.Hour)
		expVal = t
		expStr = t.Format("2006-01-02 15:04")
	}
	id := uuid()
	var code string
	var insErr error
	for i := 0; i < 6; i++ { // retry on the (vanishingly unlikely) code collision
		code = inviteCode()
		_, insErr = s.db.Exec(
			"INSERT INTO "+s.tInvites+" (id, code, note, max_uses, used_count, expires_at, disabled, grant_role, created_by) VALUES (?,?,?,?,0,?,0,?,?)",
			id, code, body.Note, body.MaxUses, expVal, grantRole, sess.userID,
		)
		if insErr == nil {
			break
		}
	}
	if insErr != nil {
		s.fail(w, insErr)
		return
	}
	writeJSON(w, 201, InviteInfo{
		ID: id, Code: code, Note: body.Note, MaxUses: body.MaxUses, UsedCount: 0,
		ExpiresAt: expStr, Disabled: false, GrantRole: grantRole, Status: "active",
		CreatedAt: time.Now().Format("2006-01-02 15:04"),
	})
}

func (s *Server) handleInviteUses(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var code string
	if err := s.db.QueryRow("SELECT code FROM "+s.tInvites+" WHERE id=?", id).Scan(&code); err != nil {
		if err == sql.ErrNoRows {
			httpError(w, 404, "not found")
			return
		}
		s.fail(w, err)
		return
	}
	rows, err := s.db.Query("SELECT username, create_time FROM "+s.tUsers+" WHERE invite_code=? ORDER BY create_time DESC", code)
	if err != nil {
		s.fail(w, err)
		return
	}
	defer rows.Close()
	out := []InviteUse{}
	for rows.Next() {
		var u InviteUse
		var t time.Time
		if err := rows.Scan(&u.Username, &t); err != nil {
			s.fail(w, err)
			return
		}
		u.UsedAt = t.Format("2006-01-02 15:04")
		out = append(out, u)
	}
	writeJSON(w, 200, out)
}

func (s *Server) handleToggleInvite(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Disabled bool `json:"disabled"`
	}
	if err := readJSON(r, &body); err != nil {
		httpError(w, 400, "bad request")
		return
	}
	res, err := s.db.Exec("UPDATE "+s.tInvites+" SET disabled=? WHERE id=?", body.Disabled, id)
	if err != nil {
		s.fail(w, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		httpError(w, 404, "not found")
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) handleDeleteInvite(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.db.Exec("DELETE FROM "+s.tInvites+" WHERE id=?", id); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// ---------- read ----------

func (s *Server) handleAll(w http.ResponseWriter, r *http.Request) {
	owner := globalOwner
	if r.URL.Query().Get("scope") == "mine" {
		sess, ok := s.currentUser(r)
		if !ok {
			httpError(w, 401, "unauthorized")
			return
		}
		owner = sess.userID
	}
	cats, err := s.categoriesFor(owner)
	if err != nil {
		s.fail(w, err)
		return
	}
	links, err := s.linksFor(owner)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"categories": cats, "links": links})
}

// ---------- category handlers ----------

func (s *Server) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.currentUser(r)
	owner, ok := s.writeOwner(w, r, sess)
	if !ok {
		return
	}
	var body Category
	if err := readJSON(r, &body); err != nil || body.Name == "" {
		httpError(w, 400, "name required")
		return
	}
	if body.Icon == "" {
		body.Icon = "ri:folder-line"
	}
	var weight int
	s.db.QueryRow("SELECT COALESCE(MAX(weight)+1, 0) FROM "+s.tCats+" WHERE owner_id=?", owner).Scan(&weight)
	body.ID = uuid()
	body.Weight = weight
	if _, err := s.db.Exec("INSERT INTO "+s.tCats+" (id, owner_id, name, icon, weight) VALUES (?,?,?,?,?)",
		body.ID, owner, body.Name, body.Icon, body.Weight); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 201, body)
}

func (s *Server) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.currentUser(r)
	id := r.PathValue("id")
	var owner string
	if err := s.db.QueryRow("SELECT owner_id FROM "+s.tCats+" WHERE id=?", id).Scan(&owner); err != nil {
		if err == sql.ErrNoRows {
			httpError(w, 404, "not found")
			return
		}
		s.fail(w, err)
		return
	}
	if !canEdit(owner, sess) {
		httpError(w, 403, "无权修改")
		return
	}
	var body Category
	if err := readJSON(r, &body); err != nil {
		httpError(w, 400, "bad request")
		return
	}
	if _, err := s.db.Exec("UPDATE "+s.tCats+" SET name=?, icon=? WHERE id=?", body.Name, body.Icon, id); err != nil {
		s.fail(w, err)
		return
	}
	var c Category
	if err := s.db.QueryRow("SELECT id, name, icon, weight FROM "+s.tCats+" WHERE id=?", id).
		Scan(&c.ID, &c.Name, &c.Icon, &c.Weight); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, c)
}

func (s *Server) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.currentUser(r)
	id := r.PathValue("id")
	var owner string
	if err := s.db.QueryRow("SELECT owner_id FROM "+s.tCats+" WHERE id=?", id).Scan(&owner); err != nil {
		if err == sql.ErrNoRows {
			httpError(w, 404, "not found")
			return
		}
		s.fail(w, err)
		return
	}
	if !canEdit(owner, sess) {
		httpError(w, 403, "无权删除")
		return
	}
	tx, err := s.db.Begin()
	if err != nil {
		s.fail(w, err)
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM "+s.tLinks+" WHERE category_id=?", id); err != nil {
		s.fail(w, err)
		return
	}
	if _, err := tx.Exec("DELETE FROM "+s.tCats+" WHERE id=?", id); err != nil {
		s.fail(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) handleReorderCategories(w http.ResponseWriter, r *http.Request) {
	s.reorder(w, r, s.tCats)
}

// ---------- link handlers ----------

func (s *Server) handleCreateLink(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.currentUser(r)
	owner, ok := s.writeOwner(w, r, sess)
	if !ok {
		return
	}
	var body Link
	if err := readJSON(r, &body); err != nil || body.CategoryID == "" || body.Title == "" || body.URL == "" {
		httpError(w, 400, "categoryId, title, url required")
		return
	}
	if !isHTTPURL(body.URL) {
		httpError(w, 400, "url must start with http:// or https://")
		return
	}
	// The category must exist and belong to the same owner.
	var catOwner string
	if err := s.db.QueryRow("SELECT owner_id FROM "+s.tCats+" WHERE id=?", body.CategoryID).Scan(&catOwner); err != nil || catOwner != owner {
		httpError(w, 400, "invalid category")
		return
	}
	var weight int
	s.db.QueryRow("SELECT COALESCE(MAX(weight)+1, 0) FROM "+s.tLinks+" WHERE category_id=?", body.CategoryID).Scan(&weight)
	body.ID = uuid()
	body.Weight = weight
	body.Clicks = 0
	if _, err := s.db.Exec(
		"INSERT INTO "+s.tLinks+" (id, owner_id, category_id, title, url, description, icon, clicks, weight) VALUES (?,?,?,?,?,?,?,?,?)",
		body.ID, owner, body.CategoryID, body.Title, body.URL, body.Description, body.Icon, body.Clicks, body.Weight,
	); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 201, body)
}

func (s *Server) handleUpdateLink(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.currentUser(r)
	id := r.PathValue("id")
	var owner, oldCat string
	if err := s.db.QueryRow("SELECT owner_id, category_id FROM "+s.tLinks+" WHERE id=?", id).Scan(&owner, &oldCat); err != nil {
		if err == sql.ErrNoRows {
			httpError(w, 404, "not found")
			return
		}
		s.fail(w, err)
		return
	}
	if !canEdit(owner, sess) {
		httpError(w, 403, "无权修改")
		return
	}
	var body Link
	if err := readJSON(r, &body); err != nil {
		httpError(w, 400, "bad request")
		return
	}
	if body.URL != "" && !isHTTPURL(body.URL) {
		httpError(w, 400, "url must start with http:// or https://")
		return
	}
	if body.CategoryID == "" {
		body.CategoryID = oldCat
	}
	// A moved-to category must belong to the same owner.
	if body.CategoryID != oldCat {
		var catOwner string
		if err := s.db.QueryRow("SELECT owner_id FROM "+s.tCats+" WHERE id=?", body.CategoryID).Scan(&catOwner); err != nil || catOwner != owner {
			httpError(w, 400, "invalid category")
			return
		}
		var nw int
		s.db.QueryRow("SELECT COALESCE(MAX(weight)+1, 0) FROM "+s.tLinks+" WHERE category_id=?", body.CategoryID).Scan(&nw)
		if _, err := s.db.Exec(
			"UPDATE "+s.tLinks+" SET category_id=?, title=?, url=?, description=?, icon=?, weight=? WHERE id=?",
			body.CategoryID, body.Title, body.URL, body.Description, body.Icon, nw, id,
		); err != nil {
			s.fail(w, err)
			return
		}
	} else {
		if _, err := s.db.Exec(
			"UPDATE "+s.tLinks+" SET category_id=?, title=?, url=?, description=?, icon=? WHERE id=?",
			body.CategoryID, body.Title, body.URL, body.Description, body.Icon, id,
		); err != nil {
			s.fail(w, err)
			return
		}
	}
	var l Link
	var desc sql.NullString
	if err := s.db.QueryRow(
		"SELECT id, category_id, title, url, description, icon, clicks, weight FROM "+s.tLinks+" WHERE id=?", id,
	).Scan(&l.ID, &l.CategoryID, &l.Title, &l.URL, &desc, &l.Icon, &l.Clicks, &l.Weight); err != nil {
		s.fail(w, err)
		return
	}
	l.Description = desc.String
	writeJSON(w, 200, l)
}

func (s *Server) handleDeleteLink(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.currentUser(r)
	id := r.PathValue("id")
	var owner string
	if err := s.db.QueryRow("SELECT owner_id FROM "+s.tLinks+" WHERE id=?", id).Scan(&owner); err != nil {
		if err == sql.ErrNoRows {
			httpError(w, 404, "not found")
			return
		}
		s.fail(w, err)
		return
	}
	if !canEdit(owner, sess) {
		httpError(w, 403, "无权删除")
		return
	}
	if _, err := s.db.Exec("DELETE FROM "+s.tLinks+" WHERE id=?", id); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func (s *Server) handleReorderLinks(w http.ResponseWriter, r *http.Request) {
	s.reorder(w, r, s.tLinks)
}

func (s *Server) handleClick(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// Public clicks are allowed only on global links; an authenticated user may
	// also click their own. Prevents cross-tenant increment / existence probing.
	var res sql.Result
	var err error
	if sess, ok := s.currentUser(r); ok {
		res, err = s.db.Exec("UPDATE "+s.tLinks+" SET clicks = clicks + 1 WHERE id=? AND (owner_id=? OR owner_id=?)", id, globalOwner, sess.userID)
	} else {
		res, err = s.db.Exec("UPDATE "+s.tLinks+" SET clicks = clicks + 1 WHERE id=? AND owner_id=?", id, globalOwner)
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		httpError(w, 404, "not found")
		return
	}
	var clicks int
	s.db.QueryRow("SELECT clicks FROM "+s.tLinks+" WHERE id=?", id).Scan(&clicks)
	writeJSON(w, 200, map[string]int{"clicks": clicks})
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.currentUser(r)
	owner, ok := s.writeOwner(w, r, sess)
	if !ok {
		return
	}
	var raw json.RawMessage
	if err := readJSON(r, &raw); err != nil {
		httpError(w, 400, "bad request")
		return
	}
	res, err := s.importData(raw, owner)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, res)
}

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.currentUser(r)
	if sess.role != "admin" {
		httpError(w, 403, "只有管理员能重置")
		return
	}
	tx, err := s.db.Begin()
	if err != nil {
		s.fail(w, err)
		return
	}
	// Reset only the GLOBAL dataset; user data is left intact.
	if _, err := tx.Exec("DELETE FROM "+s.tLinks+" WHERE owner_id=?", globalOwner); err != nil {
		tx.Rollback()
		s.fail(w, err)
		return
	}
	if _, err := tx.Exec("DELETE FROM "+s.tCats+" WHERE owner_id=?", globalOwner); err != nil {
		tx.Rollback()
		s.fail(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		s.fail(w, err)
		return
	}
	if err := s.seedIfEmpty(); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// ---------- data access ----------

func (s *Server) categoriesFor(owner string) ([]Category, error) {
	rows, err := s.db.Query("SELECT id, name, icon, weight FROM "+s.tCats+" WHERE owner_id=? ORDER BY weight ASC, id ASC", owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Icon, &c.Weight); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Server) linksFor(owner string) ([]Link, error) {
	rows, err := s.db.Query("SELECT id, category_id, title, url, description, icon, clicks, weight FROM "+s.tLinks+" WHERE owner_id=? ORDER BY weight ASC, id ASC", owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Link{}
	for rows.Next() {
		var l Link
		var desc sql.NullString
		if err := rows.Scan(&l.ID, &l.CategoryID, &l.Title, &l.URL, &desc, &l.Icon, &l.Clicks, &l.Weight); err != nil {
			return nil, err
		}
		l.Description = desc.String
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Server) reorder(w http.ResponseWriter, r *http.Request, table string) {
	sess, _ := s.currentUser(r)
	var body struct {
		OrderedIDs []string `json:"orderedIds"`
	}
	if err := readJSON(r, &body); err != nil {
		httpError(w, 400, "bad request")
		return
	}
	if len(body.OrderedIDs) == 0 {
		writeJSON(w, 200, map[string]bool{"ok": true})
		return
	}
	// Permission is derived from the owner of the first row; updates are scoped
	// to that owner so a caller can't reweight another tenant's rows.
	var owner string
	if err := s.db.QueryRow("SELECT owner_id FROM "+table+" WHERE id=?", body.OrderedIDs[0]).Scan(&owner); err != nil {
		if err == sql.ErrNoRows {
			httpError(w, 404, "not found")
			return
		}
		s.fail(w, err)
		return
	}
	if !canEdit(owner, sess) {
		httpError(w, 403, "无权排序")
		return
	}
	tx, err := s.db.Begin()
	if err != nil {
		s.fail(w, err)
		return
	}
	defer tx.Rollback()
	for i, id := range body.OrderedIDs {
		if _, err := tx.Exec("UPDATE "+table+" SET weight=? WHERE id=? AND owner_id=?", i, id, owner); err != nil {
			s.fail(w, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// importData inserts tolerant ZMark/OneNav-ish JSON into the given owner's space.
func (s *Server) importData(raw json.RawMessage, owner string) (map[string]any, error) {
	type inCat struct {
		Name, Title, Icon string
	}
	type inLink struct {
		Title, Name, URL, Link, Description, Desc, Icon string
		Clicks                                          json.Number
		CategoryID, CategoryName, Category, CatName     string
	}
	var payload struct {
		Categories []inCat  `json:"categories"`
		Links      []inLink `json:"links"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		var bare []inLink
		if err2 := json.Unmarshal(raw, &bare); err2 != nil {
			return nil, fmt.Errorf("invalid import payload")
		}
		payload.Links = bare
	}

	cats, err := s.categoriesFor(owner)
	if err != nil {
		return nil, err
	}
	links, err := s.linksFor(owner)
	if err != nil {
		return nil, err
	}
	nameToID := map[string]string{}
	idSet := map[string]bool{}
	maxCatWeight := -1
	for _, c := range cats {
		nameToID[c.Name] = c.ID
		idSet[c.ID] = true
		if c.Weight > maxCatWeight {
			maxCatWeight = c.Weight
		}
	}
	seen := map[string]bool{}
	for _, l := range links {
		seen[l.CategoryID+"\n"+l.URL] = true
	}
	addedC, addedL, skipped := 0, 0, 0
	truncated := false

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	ensureCat := func(catName, icon string) (string, error) {
		if catName == "" {
			catName = "导入"
		}
		if id, ok := nameToID[catName]; ok {
			return id, nil
		}
		id := uuid()
		if icon == "" {
			icon = "ri:folder-line"
		}
		maxCatWeight++
		if _, err := tx.Exec("INSERT INTO "+s.tCats+" (id, owner_id, name, icon, weight) VALUES (?,?,?,?,?)", id, owner, catName, icon, maxCatWeight); err != nil {
			return "", err
		}
		nameToID[catName] = id
		idSet[id] = true
		addedC++
		return id, nil
	}

	for _, c := range payload.Categories {
		nm := c.Name
		if nm == "" {
			nm = c.Title
		}
		if _, err := ensureCat(nm, c.Icon); err != nil {
			return nil, err
		}
	}

	catCount := map[string]int{}
	for _, l := range payload.Links {
		if addedL >= 2000 {
			truncated = true
			break
		}
		title := firstNonEmpty(l.Title, l.Name)
		linkURL := firstNonEmpty(l.URL, l.Link)
		if title == "" || linkURL == "" || !isHTTPURL(linkURL) {
			skipped++
			continue
		}
		catID := l.CategoryID
		if catID == "" || !idSet[catID] {
			id, err := ensureCat(firstNonEmpty(l.CategoryName, l.Category, l.CatName), "")
			if err != nil {
				return nil, err
			}
			catID = id
		}
		key := catID + "\n" + linkURL
		if seen[key] {
			skipped++
			continue
		}
		seen[key] = true
		if _, ok := catCount[catID]; !ok {
			var n int
			s.db.QueryRow("SELECT COALESCE(MAX(weight)+1, 0) FROM "+s.tLinks+" WHERE category_id=?", catID).Scan(&n)
			catCount[catID] = n
		}
		clicks := 0
		if l.Clicks != "" {
			if v, err := strconv.Atoi(l.Clicks.String()); err == nil {
				clicks = v
			}
		}
		if _, err := tx.Exec(
			"INSERT INTO "+s.tLinks+" (id, owner_id, category_id, title, url, description, icon, clicks, weight) VALUES (?,?,?,?,?,?,?,?,?)",
			uuid(), owner, catID, title, linkURL, firstNonEmpty(l.Description, l.Desc), l.Icon, clicks, catCount[catID],
		); err != nil {
			return nil, err
		}
		catCount[catID]++
		addedL++
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return map[string]any{
		"addedCategories": addedC,
		"addedLinks":      addedL,
		"skipped":         skipped,
		"truncated":       truncated,
	}, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

var _ = mysql.Config{} // ensure driver import is referenced
