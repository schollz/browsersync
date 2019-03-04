// main.go is a dual-function program
// go run main.go will generate the index.html
// go build; ./now will run a live-reload server

//go:generate go run data/embed.go
package main

import (
	"bytes"
	"flag"
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

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"github.com/shurcooL/github_flavored_markdown"
)

var folderToWatch, indexPage string
var port int
var renderMarkdown bool

func main() {
	flag.IntVar(&port, "p", 8003, "port to serve")
	flag.BoolVar(&renderMarkdown, "style", false, "whether to add default styling to render markdown")
	flag.StringVar(&indexPage, "index", "index.html", "index page to render on /")
	flag.StringVar(&folderToWatch, "f", "", "folder to watch (default: current)")
	flag.Parse()
	if folderToWatch == "" {
		folderToWatch = "."
	}
	if filepath.Ext(indexPage) == ".md" {
		renderMarkdown = true
	}
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
	html := github_flavored_markdown.Markdown([]byte(markdown))
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
	log.Printf("listening on :%d", port)
	http.HandleFunc("/", handler)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
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
	} else if r.URL.Path == "/"+jsFile {
		w.Header().Set("Content-Type", "text/javascript")
		w.Write([]byte(js))
	} else {
		if r.URL.Path == "/" {
			r.URL.Path = "/" + indexPage
		}
		var b []byte
		log.Println(path.Join(".", path.Clean(r.URL.Path[1:])))
		b, err = ioutil.ReadFile(path.Join(".", path.Clean(r.URL.Path[1:])))
		if err != nil {
			log.Println(err)
			log.Println("try2:", path.Join(".", path.Clean(r.URL.Path[1:]), "index.html"))
			b, err = ioutil.ReadFile(path.Join(".", path.Clean(r.URL.Path[1:]), "index.html"))
			if err != nil {
				err = fmt.Errorf("could not find file")
				return
			}
		}

		var kind string
		if len(b) > 512 {
			kind = http.DetectContentType(b)
		} else {
			kind = http.DetectContentType(b[:512])
		}

		// https://developer.mozilla.org/en-US/docs/Web/HTTP/Basics_of_HTTP/MIME_types
		switch filepath.Ext(r.URL.Path) {
		case ".js":
			kind = "text/javascript"
		case ".css":
			kind = "text/css"
		case ".md":
			kind = "text/plain"
		default:
			kind = "text/html"
			mainTemplate, errTemplate := template.New("main").Funcs(funcMap).Parse(string(b))
			if errTemplate == nil {
				var buf bytes.Buffer
				err = mainTemplate.Execute(&buf, nil)
				b = buf.Bytes()
			} else {
				log.Println("problem as template: ", errTemplate)
			}
		}

		log.Println(renderMarkdown)
		if renderMarkdown {
			b = []byte(strings.Replace(defaultHTML, "XX", string(MarkdownToHTML(r.URL.Path[1:])), 1))
			kind = "text/html"
		}

		if kind == "text/html" {
			b = bytes.Replace(b,
				[]byte("</body>"),
				[]byte(fmt.Sprintf(`<script src="/%s"></script></body>`, jsFile)),
				1,
			)
		}

		w.Header().Set("Content-Type", kind)
		w.Write(b)

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

func watchFileSystem() (err error) {
	// creates a new file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	lastEvent := time.Now()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if time.Since(lastEvent).Nanoseconds() > (50 * time.Millisecond).Nanoseconds() {
					lastEvent = time.Now()
					log.Println("event:", event)
					wsConnections.Lock()
					for c := range wsConnections.cs {
						wsConnections.cs[c].WriteJSON(Payload{Message: "reload"})
					}
					wsConnections.Unlock()
				}

				// if event.Op&fsnotify.Write == fsnotify.Write {
				// 	log.Println("modified file:", event.Name)
				// }
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	filepath.Walk(folderToWatch, func(path string, fi os.FileInfo, err error) error {
		if fi.Mode().IsDir() {
			log.Printf("watching %s", path)
			return watcher.Add(path)
		}
		return nil
	})

	<-done
	return
}
