// © 2015 the yoctobuild Authors under the MIT license. See AUTHORS for the list of authors.

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
)

var (
	configPath = flag.String("config", "./config.json", "Path to config file")
	badgePath  = flag.String("badges", "./badges/", "Path to badges")
	addr       = flag.String("addr", ":3001", "Address to serve on")
	secret     = flag.String("secret", "12345", "Secret to authorize builds")

	projects map[string]*project
)

type project struct {
	Before string
	After  string
	out    string
	err    error
	time   time.Time
}

func runBuild(name string) {
	steps := fmt.Sprintf("mkdir -p %s; cd %s; %s",
		name, name, projects[name].Before)
	script := bytes.NewBufferString(steps)
	projects[name].time = time.Time{}

	bash := exec.Command("bash")
	stdin, _ := bash.StdinPipe()

	io.Copy(stdin, script)
	stdin.Close()

	out, err := bash.CombinedOutput()

	projects[name].out = string(out)
	projects[name].err = err
	projects[name].time = time.Now()

	if err == nil {
		runPostBuild(name)
	}
}

func runPostBuild(name string) {
	steps := fmt.Sprintf("cd %s; %s",
		name, projects[name].After)
	script := bytes.NewBufferString(steps)

	bash := exec.Command("bash")
	stdin, _ := bash.StdinPipe()

	io.Copy(stdin, script)
	stdin.Close()

	if out, err := bash.CombinedOutput(); err != nil {
		projects[name].out = string(out)
		projects[name].err = err
	}
}

func readConfig() {
	if f, err := ioutil.ReadFile(*configPath); err != nil {
		log.Fatal("Could not access configuration.", err)
	} else {
		if err := json.Unmarshal(f, &projects); err != nil {
			log.Fatal("Could not read configuration.", err)
		}
	}
}

func getProject(r *http.Request) string {
	vars := mux.Vars(r)
	return vars["project"]
}

// TODO: Make these templates or something
func writeHeader(w http.ResponseWriter, title string) {
	fmt.Fprintf(w, "<html><head><title>%s - yoctobuild</title></head><body>", title)
}

func writeFooter(w http.ResponseWriter) {
	fmt.Fprintf(w, "<body></html>")
}

func projectIndex(w http.ResponseWriter, r *http.Request) {
	writeHeader(w, "index")
	fmt.Fprintf(w, "<p>Projects:</p><ul>")
	for name := range projects {
		fmt.Fprintf(w, `<li><a href="/projects/%s">%s</a></li>`, name, name)
	}
	fmt.Fprintf(w, "</ul>")
	writeFooter(w)
}

func projectStatus(w http.ResponseWriter, r *http.Request) {
	name := getProject(r)
	writeHeader(w, name)
	fmt.Fprintf(w, `<p><img src="/projects/%s/badge" /></p>`, name)
	if p, ok := projects[name]; ok && p.err != nil {
		fmt.Fprintf(w, "Last built: %s<br>\nError: <pre>%s</pre><br>\nOutput:<br>\n<pre>%s</pre>\n", p.time, p.err, p.out)
	} else if ok && !p.time.IsZero() {
		fmt.Fprintf(w, "Last built: %s<br>\nOutput:<br>\n<pre>%s</pre>\n", p.time, p.out)
	}
	writeFooter(w)
}

func projectBadge(w http.ResponseWriter, r *http.Request) {
	name, file := getProject(r), ""

	if p, ok := projects[name]; ok && p.err != nil {
		file = "failing.png"
	} else if ok && !p.time.IsZero() {
		file = "passing.png"
	} else {
		file = "pending.png"
	}

	// http.ServeFile is neat, but it likes to write its own status codes that are wrong
	if f, err := os.Open(filepath.Join(*badgePath, file)); err != nil {
		http.NotFound(w, r)
	} else {
		w.Header().Set("Cache-Control", "no-cache, private")
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(200)
		io.Copy(w, f)
	}
}

func projectBuild(w http.ResponseWriter, r *http.Request) {
	name := getProject(r)

	get, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || get.Get("secret") != *secret {
		w.WriteHeader(401)
		return
	}

	go runBuild(name)
	fmt.Fprintf(w, "Build scheduled.\n")
}

func main() {
	flag.Parse()

	readConfig()

	r := mux.NewRouter()
	r.Handle("/", http.RedirectHandler("/projects", 301))
	r.HandleFunc("/projects", projectIndex)
	r.HandleFunc("/projects/{project}", projectStatus)
	r.HandleFunc("/projects/{project}/badge", projectBadge)
	r.HandleFunc("/projects/{project}/build", projectBuild)

	log.Fatal(http.ListenAndServe(*addr, r))
}
