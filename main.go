package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	ps "github.com/mitchellh/go-ps"
)

// VoteMap contains the contains users voting hostIP->vote-time. In the future, votes should expire after 5 minutes.
var VoteMap map[string]time.Time

// pid of the Jamulus server
var pid int

var processName string

var toggleThreshold int

func toggleRecording(attempt int) {
	err := syscall.Kill(pid, syscall.SIGUSR2)
	if err != nil {
		log.Printf("Failed to signal Jamulus process at PID %d: %v, Assuming restarted process", pid, err)
		prevPid := pid
		pid = getJamulusPid()
		if prevPid == pid {
			log.Printf("Warning: Jamulus PID is still the same, missing privileges for kill syscall? Will not retry toggling recording state.")
		} else {
			if attempt < 5 {
				toggleRecording(attempt + 1)
			}
		}
	}
}

func toggleHandler(w http.ResponseWriter, r *http.Request) {
	VoteMap[r.Host] = time.Now()
	votes := len(VoteMap)
	if votes < toggleThreshold {
		w.Write([]byte(fmt.Sprintf("%d users are voting to toggle recording state, %d required", votes, toggleThreshold)))
	} else {
		w.Write([]byte(fmt.Sprintf("Triggering recording state change...")))
		toggleRecording(0)
		// reset votes
		VoteMap = make(map[string]time.Time)
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(fmt.Sprintf("%d users are voting to toggle recording state, %d required", len(VoteMap), toggleThreshold)))
}

type indexHandler struct {
	IndexPage []byte
}

func (ih *indexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write(ih.IndexPage)
}

func getJamulusPid() int {
	processes, err := ps.Processes()
	if err != nil {
		log.Fatalf("Failed to list processes")
	}
	for _, process := range processes {
		executable := process.Executable()
		if strings.HasSuffix(executable, processName) {
			return process.Pid()
		}
	}
	log.Fatalf("Could not find Jamulus Server PID")
	return -1
}

func main() {
	var (
		insecureListenAddress string
		indexPageHTML         string
	)

	flagset := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagset.StringVar(&insecureListenAddress, "listen-address", "", "The address the HTTP server should listen on.")
	flagset.StringVar(&indexPageHTML, "index-page", "", "The index page file to display at the root.")
	flagset.IntVar(&toggleThreshold, "toggle-threshold", 2, "The number of votes necessary to toggle recording.")
	flagset.StringVar(&processName, "process-name", "Jamulus", "Process name to scan for")
	//nolint: errcheck // Parse() will exit on error.
	flagset.Parse(os.Args[1:])

	VoteMap = make(map[string]time.Time)
	// Search for the Jamulus process
	pid = getJamulusPid()

	f, err := ioutil.ReadFile(indexPageHTML)
	if err != nil {
		log.Fatalf("Could not read config at %s: %v", indexPageHTML, err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", &indexHandler{IndexPage: f})
	mux.HandleFunc("/toggle", toggleHandler)
	mux.HandleFunc("/status", statusHandler)

	time.Now()
	srv := &http.Server{Handler: mux}

	l, err := net.Listen("tcp", insecureListenAddress)
	if err != nil {
		log.Fatalf("Failed to listen on address: %v", err)
	}

	errCh := make(chan error)
	go func() {
		log.Printf("Listening insecurely on %v", l.Addr())
		errCh <- srv.Serve(l)
	}()

	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	select {
	case <-term:
		log.Print("Received SIGTERM, exiting gracefully...")
		srv.Close()
	case err := <-errCh:
		if err != http.ErrServerClosed {
			log.Printf("Server stopped with %v", err)
		}
		os.Exit(1)
	}
}
