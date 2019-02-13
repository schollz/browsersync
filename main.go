// main.go is a dual-function program
// go run main.go will generate the index.html
// go build; ./now will run a live-reload server

package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"github.com/shurcooL/github_flavored_markdown"
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
	} else {
		if r.URL.Path == "/" {
			r.URL.Path = "/index.html"
		}
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

		// https://developer.mozilla.org/en-US/docs/Web/HTTP/Basics_of_HTTP/MIME_types
		switch filepath.Ext(r.URL.Path) {
		case ".md":
			kind = "text/plain"
		case ".html":
			kind = "text/html"
			mainTemplate, errTemplate := template.New("main").Funcs(funcMap).Parse(string(b))
			if errTemplate != nil {
				err = errTemplate
				return
			}
			var buf bytes.Buffer
			err = mainTemplate.Execute(&buf, nil)
			b = buf.Bytes()
		}
		fmt.Println(kind)

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

	filepath.Walk(".", func(path string, fi os.FileInfo, err error) error {
		if fi.Mode().IsDir() {
			return watcher.Add(path)
		}
		return nil
	})

	<-done
	return
}
