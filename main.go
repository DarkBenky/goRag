package main

import (
	"database/sql"
	"math/rand"

	"gorag/ragMath"

	_ "modernc.org/sqlite"
)

var groupCount = 8192
var vecSize = 1024
var memoryLimit = 4 * 1024 * 1024 * 1024
var MODE = LOAD

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

func (r *Rag) addEmbedding(embedding []float32, text string) (dbRow int) {
	res, err := db.Exec("INSERT INTO embeddings(text) VALUES (?)", text)
	if err != nil {
		return 0
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0
	}
	dbRow = int(id)

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
	return dbRow
}

func (r *Rag) addEmbeddings(embeddings [][]float32, texts []string) (dbRows []int) {
	for i := range embeddings {
		dbRow := r.addEmbedding(embeddings[i], texts[i])
		dbRows = append(dbRows, dbRow)
	}
	return dbRows
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

func main() {
	var err error
	db, err = sql.Open("sqlite", "rag.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS embeddings (id INTEGER PRIMARY KEY AUTOINCREMENT, text TEXT NOT NULL)")
	if err != nil {
		panic(err)
	}

	cfg := loadConfig(".env")
	groupCount = cfg.GroupCount
	vecSize = cfg.VecSize
	memoryLimit = cfg.MemoryLimit
	MODE = cfg.Mode

	if MODE == LOAD {
		r := &Rag{memoryLimit: memoryLimit}
		r.createGroups()
	}
}
