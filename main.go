// main.go is a dual-function program
// go run main.go will generate the index.html
// go build; ./now will run a live-reload server

package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/microcosm-cc/bluemonday"
	blackfriday "github.com/russross/blackfriday/v2"
)

func main() {
	var err error
	err = serve()
	if err != nil {
		log.Println(err)
	}
}

func MarkdownToHTML(fname string) template.HTML {
	markdown, err := ioutil.ReadFile(fname)
	if err != nil {
		return template.HTML(err.Error())
	}
	html := blackfriday.Run([]byte(markdown),
		blackfriday.WithExtensions(
			blackfriday.Autolink|
				blackfriday.Strikethrough|
				blackfriday.SpaceHeadings|
				blackfriday.BackslashLineBreak|
				blackfriday.NoIntraEmphasis|
				blackfriday.Tables|
				blackfriday.FencedCode|
				blackfriday.AutoHeadingIDs|
				blackfriday.Footnotes|blackfriday.LaxHTMLBlocks),
	)
	p := bluemonday.UGCPolicy()
	p.AddTargetBlankToFullyQualifiedLinks(true)
	html = p.SanitizeBytes(html)

	return template.HTML(string(html))
}

var funcMap template.FuncMap

func init() {
	funcMap = template.FuncMap{
		"MarkdownToHTML": MarkdownToHTML,
	}
}

func serve() (err error) {
	go watchFileSystem()
	log.Println("running")
	http.HandleFunc("/", handler)
	return http.ListenAndServe(":8003", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	t := time.Now().UTC()
	err := handle(w, r)
	if err != nil {
		log.Println(err)
	}
	log.Printf("%v %v %v %s\n", r.RemoteAddr, r.Method, r.URL.Path, time.Since(t))
}

func handle(w http.ResponseWriter, r *http.Request) (err error) {
	// very special paths
	if r.URL.Path == "/robots.txt" {
		// special path
		w.Write([]byte(`User-agent: * 
Disallow: /`))
	} else if r.URL.Path == "/favicon.ico" {
		// TODO
	} else if r.URL.Path == "/sitemap.xml" {
		// TODO
	} else if r.URL.Path == "/ws" {
		// special path /ws
		return handleWebsocket(w, r)
	} else if r.URL.Path == "/" {
		var b []byte
		b, err = ioutil.ReadFile("index.html")
		if err != nil {
			return
		}
		mainTemplate, errTemplate := template.New("main").Funcs(funcMap).Parse(string(b))
		if errTemplate != nil {
			err = errTemplate
			return
		}
		err = mainTemplate.Execute(w, nil)
	} else {
		var b []byte
		b, err = ioutil.ReadFile(path.Join(".", path.Clean(r.URL.Path[1:])))
		if err != nil {
			return
		}

		var kind string
		if len(b) > 512 {
			kind = http.DetectContentType(b)
		} else {
			kind = http.DetectContentType(b[:512])
		}
		if kind == "application/octet-stream" {
			if strings.HasSuffix(r.URL.Path, ".md") {
				kind = "text/plain"
			}
		}
		fmt.Println(kind)

		w.Header().Set("Content-Type", kind)
		if strings.HasPrefix(kind, "text/html") {
			mainTemplate, errTemplate := template.New("main").Funcs(funcMap).Parse(string(b))
			if errTemplate != nil {
				err = errTemplate
				return
			}
			err = mainTemplate.Execute(w, nil)
		} else {
			w.Write(b)
		}

		// err = mainTemplate.Execute(w, nil)
	}
	return
}

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Payload struct {
	Message string `json:"message"`
}

type Connections struct {
	cs map[string]*websocket.Conn
	sync.RWMutex
}

var wsConnections Connections

func handleWebsocket(w http.ResponseWriter, r *http.Request) (err error) {

	// handle websockets on this page
	c, errUpgrade := wsupgrader.Upgrade(w, r, nil)
	if errUpgrade != nil {
		return errUpgrade
	}
	defer c.Close()

	log.Printf("%s connected\n", c.RemoteAddr().String())
	wsConnections.Lock()
	if len(wsConnections.cs) == 0 {
		wsConnections.cs = make(map[string]*websocket.Conn)
	}
	wsConnections.cs[c.RemoteAddr().String()] = c
	wsConnections.Unlock()
	defer func() {
		wsConnections.Lock()
		delete(wsConnections.cs, c.RemoteAddr().String())
		wsConnections.Unlock()
	}()

	var p Payload
	for {
		err := c.ReadJSON(&p)
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Println("recv: %v", p)
	}
	return
}

func watchFileSystem() {
	currentFiles := make(map[string]time.Time)
	files, _ := ListFilesRecursivelyInParallel(".")
	for _, f := range files {
		currentFiles[f.Path] = f.ModTime
	}

	for {
		time.Sleep(100 * time.Millisecond)
		files, _ := ListFilesRecursivelyInParallel(".")
		changedFiles := []string{}
		for _, f := range files {
			if _, ok := currentFiles[f.Path]; !ok {
				changedFiles = append(changedFiles, f.Path)
			} else if f.ModTime != currentFiles[f.Path] {
				changedFiles = append(changedFiles, f.Path)
			}
			currentFiles[f.Path] = f.ModTime
		}

		if len(changedFiles) > 0 {
			log.Printf("changed files: %+v, reloading", changedFiles)
			wsConnections.Lock()
			for c := range wsConnections.cs {
				wsConnections.cs[c].WriteJSON(Payload{Message: "reload"})
			}
			wsConnections.Unlock()
		}
	}
}

// File is the object that contains the info and path of the file
type File struct {
	Path    string
	Size    int64
	Mode    os.FileMode
	ModTime time.Time
	IsDir   bool
	Hash    uint64 `hash:"ignore"`
}

// ListFilesRecursivelyInParallel uses goroutines to list all the files
func ListFilesRecursivelyInParallel(dir string) (files []File, err error) {
	dir = filepath.Clean(dir)
	f, err := os.Open(dir)
	if err != nil {
		return
	}
	info, err := f.Stat()
	if err != nil {
		return
	}
	files = []File{
		{
			Path:    dir,
			Size:    info.Size(),
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
			IsDir:   info.IsDir(),
		},
	}
	f.Close()

	fileChan := make(chan File)
	startedDirectories := make(chan bool)
	go listFilesInParallel(dir, startedDirectories, fileChan)

	runningCount := 1
	for {
		select {
		case file := <-fileChan:
			files = append(files, file)
		case newDir := <-startedDirectories:
			if newDir {
				runningCount++
			} else {
				runningCount--
			}
		default:
		}
		if runningCount == 0 {
			break
		}
	}
	return
}

func listFilesInParallel(dir string, startedDirectories chan bool, fileChan chan File) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	for _, f := range files {
		fileStruct := File{
			Path:    path.Join(dir, f.Name()),
			Size:    f.Size(),
			Mode:    f.Mode(),
			ModTime: f.ModTime(),
			IsDir:   f.IsDir(),
		}
		fileChan <- fileStruct
		if f.IsDir() {
			startedDirectories <- true
			go listFilesInParallel(path.Join(dir, f.Name()), startedDirectories, fileChan)
		}
	}
	startedDirectories <- false
	return
}
