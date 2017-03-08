package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/fsnotify/fsnotify"
)

const appName = "zap"

// Used in version printer, set by GoReleaser.
var version = "develop"

func main() {

	var (
		configName = flag.String("config", "c.yml", "config file")
		port       = flag.Int("port", 8927, "port to bind to")
		host       = flag.String("host", "127.0.0.1", "host interface")
		v          = flag.Bool("v", false, "print version info")
	)
	flag.Parse()

	if *v {
		fmt.Println(version)
		os.Exit(0)
	}

	// load config for first time.
	c, err := parseYaml(*configName)
	if err != nil {
		log.Printf("Error parsing config file. Please fix syntax: %s\n", err)
		return
	}
	context := &context{config: c}
	updateHosts(context) // sync changes since last run.

	// Enable hot reload.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Enable hot reload.
	cb := makeCallback(context, *configName)
	go watchChanges(watcher, *configName, cb)
	err = watcher.Add(path.Dir(*configName))
	if err != nil {
		log.Fatal(err)
	}

	// Set up routes.
	http.Handle("/", ctxWrapper{context, IndexHandler})
	http.Handle("/varz", ctxWrapper{context, VarsHandler})
	http.HandleFunc("/healthz", HealthHandler)

	// TODO check for errors - addr in use, sudo issues, etc.
	fmt.Printf("Launching %s on %s:%d\n", appName, *host, *port)
	http.ListenAndServe(fmt.Sprintf("%s:%d", *host, *port), nil)
}
