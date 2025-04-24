package main

import (
    "database/sql"
    "embed"
    "fmt"
    "io/fs"
    "log"
    "net/http"
    "time"
    "path/filepath"
    "os"
    "strings"

    "github.com/go-chi/chi/v5"
    _ "github.com/mattn/go-sqlite3"
    "github.com/go-chi/chi/v5/middleware"

    "github.com/fuji-184/fmusik/controller"
    "github.com/fuji-184/fmusik/utils"
)

//go:embed all:frontend/build
var svelteStatic embed.FS

type Tes struct {
    Id   int    `json:"id"`
    Name string `json:"name"`
}

func main() {
    s, err := fs.Sub(svelteStatic, "frontend/build")
    if err != nil {
        panic(err)
    }

    staticServer := http.FileServer(http.FS(s))

    workDir, _ := os.Getwd()
    fileDir := http.Dir(filepath.Join(workDir, "files"))


    db, err := sql.Open("sqlite3", "./fuji.db")
    if err != nil {
        log.Fatal(err)
    }

    db.SetMaxOpenConns(10)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(time.Hour)

    defer db.Close()

    utils.MigrateSQLite(db)

    r := chi.NewRouter()

    r.Use(middleware.Compress(5, "text/html", "text/css", "application/javascript", "image/*"))

    r.Handle("/", staticServer)
    r.Handle("/_app/*", staticServer)
    r.Handle("/service-worker.js", staticServer)
    r.Handle("/favicon.png", staticServer)

    r.Get("/tes", func(w http.ResponseWriter, r *http.Request) {
        controller.HandleTes(w, r, db)
    })
    r.HandleFunc("/ws", controller.WsHandler)

    FileServer(r, "/files", fileDir)

    r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
        r.URL.Path = "/"
        staticServer.ServeHTTP(w, r)
    })

    fmt.Println("Running on port: 3000")
    log.Fatal(http.ListenAndServe(":3000", r))
}

func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
