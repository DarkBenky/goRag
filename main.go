package main

import (
	"database/sql"
	"log/slog"
	"math/rand"
	"strings"
	"sync"

	"gorag/ragMath"

	_ "modernc.org/sqlite"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

var groupCount = 8192
var vecSize = 1024
var memoryLimit = 4 * 1024 * 1024 * 1024
var MODE = LOAD

const dbPath = "rag.db"
const dbPragmas = "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"

var db *sql.DB

type Embedding struct {
	embedding []float32
	dbRow     int
}

type Group struct {
	// metadata always in RAM
	centroid         []float32
	fileName         [32]byte
	size             int
	memoryUsed       int
	loadedEmbeddings bool

	// embeddings only loaded when needed
	embeddings []Embedding
}

type Result struct {
	dbRow int
	score float32
}

type Rag struct {
	mu          sync.Mutex
	groups      []Group
	memoryUsed  int
	memoryLimit int
}

func makeData(vecSize int) []float32 {
	data := make([]float32, vecSize)
	for i := range data {
		data[i] = rand.Float32()
	}
	return data
}

func makeFileName() [32]byte {
	var fileName [32]byte

	for i := range fileName {
		fileName[i] = byte(rand.Intn(0x7E-0x21+1) + 0x21)
	}

	return fileName
}

func (r *Rag) createGroups() {
	r.groups = make([]Group, groupCount)
	for i := 0; i < groupCount; i++ {
		r.groups[i] = Group{centroid: makeData(vecSize), fileName: makeFileName()}
	}
}

func (r *Rag) addGroup() {
	group := Group{centroid: makeData(vecSize), fileName: makeFileName()}
	r.groups = append(r.groups, group)
}

func countWords(text string) int {
	return len(strings.Fields(text))
}

func (r *Rag) addEmbedding(embedding []float32, text string, category string) (dbRow int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	tx, err := db.Begin()
	if err != nil {
		slog.Error("begin transaction", "error", err)
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec("INSERT INTO embeddings(text, category) VALUES (?, ?)", text, category)
	if err != nil {
		slog.Error("insert embedding", "error", err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		slog.Error("last insert id", "error", err)
		return 0, err
	}
	dbRow = int(id)

	words := countWords(text)
	if _, err := tx.Exec(`INSERT INTO stats(category, samples, words) VALUES (?, 1, ?)
		ON CONFLICT(category) DO UPDATE SET samples = samples + 1, words = words + ?`, category, words, words); err != nil {
		slog.Error("update stats", "error", err)
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		slog.Error("commit transaction", "error", err)
		return 0, err
	}

	centroids := make([][]float32, len(r.groups))
	for i := range r.groups {
		centroids[i] = r.groups[i].centroid
	}
	scores := ragMath.DotProductsSimd(embedding, centroids)

	bestIndex := 0
	bestScore := scores[0]
	for i := 1; i < len(scores); i++ {
		if scores[i] > bestScore {
			bestIndex = i
			bestScore = scores[i]
		}
	}

	g := &r.groups[bestIndex]
	g.embeddings = append(g.embeddings, Embedding{embedding: embedding, dbRow: dbRow})
	g.size++
	g.loadedEmbeddings = true
	g.memoryUsed += len(embedding) * 4
	r.memoryUsed += len(embedding) * 4
	for i := range g.centroid {
		g.centroid[i] += (embedding[i] - g.centroid[i]) / float32(g.size)
	}
	return dbRow, nil
}

func (r *Rag) addEmbeddings(embeddings [][]float32, texts []string, categories []string) (dbRows []int, err error) {
	for i := range embeddings {
		var dbRow int
		dbRow, err = r.addEmbedding(embeddings[i], texts[i], categories[i])
		if err != nil {
			return dbRows, err
		}
		dbRows = append(dbRows, dbRow)
	}
	return dbRows, nil
}

// func (r *Rag) assign(embedding []float32) *Group {}
//
// func (g *Group) recomputeCentroid() {}
//
// func (r *Rag) rebuild(embeddings []Embedding, rounds int) {}
//
// func (r *Rag) storeGroups(dir string) error {}
//
// func (r *Rag) storeMeta(path string) error {}
//
// func (r *Rag) loadMeta(path string) error {}
//
// func (r *Rag) unloadGroup(i int) {}
//
// func (r *Rag) loadGroup(dir string, i int) error {}
//
// func (r *Rag) freeMemory(keep []int) {}
//
// func (r *Rag) search(query []float32, dir string, topGroups, topSamples int) []Result {}

type Op uint8

const (
	LOAD Op = iota
	SERVE
)

func postEmbedding(c *echo.Context) error {
	var req struct {
		Embedding []float32 `json:"embedding"`
		Text      string    `json:"text"`
		Category  string    `json:"category"`
	}

	if err := c.Bind(&req); err != nil {
		return err
	}

	dbRow, err := r.addEmbedding(req.Embedding, req.Text, req.Category)
	if err != nil {
		return echo.NewHTTPError(500, "failed to store embedding")
	}
	return c.JSON(200, map[string]any{"status": "ok", "dbRow": dbRow})
}

func postEmbeddings(c *echo.Context) error {
	var req struct {
		Embeddings [][]float32 `json:"embeddings"`
		Texts      []string    `json:"texts"`
		Categories []string    `json:"categories"`
	}

	if err := c.Bind(&req); err != nil {
		return err
	}

	dbRows, err := r.addEmbeddings(req.Embeddings, req.Texts, req.Categories)
	if err != nil {
		return echo.NewHTTPError(500, "failed to store embeddings")
	}
	return c.JSON(200, map[string]any{"status": "ok", "dbRows": dbRows})
}

var r *Rag

func main() {
	var err error
	db, err = sql.Open("sqlite", "file:"+dbPath+dbPragmas)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS embeddings (id INTEGER PRIMARY KEY AUTOINCREMENT, text TEXT NOT NULL, category TEXT NOT NULL)")
	if err != nil {
		panic(err)
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS stats (category TEXT PRIMARY KEY, samples INTEGER NOT NULL DEFAULT 0, words INTEGER NOT NULL DEFAULT 0)")
	if err != nil {
		panic(err)
	}

	cfg := loadConfig(".env")
	groupCount = cfg.GroupCount
	vecSize = cfg.VecSize
	memoryLimit = cfg.MemoryLimit
	MODE = cfg.Mode

	if MODE == LOAD {
		r = &Rag{memoryLimit: memoryLimit}
		r.createGroups()

		e := echo.New()

		e.Use(middleware.RequestLogger())
		e.Use(middleware.Recover())

		e.POST("/embedding", postEmbedding)
		e.POST("/embeddings", postEmbeddings)

		if err := e.Start(":8023"); err != nil {
			slog.Error("failed to start server", "error", err)
		}
	}
}
