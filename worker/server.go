package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func HandleCmd(w http.ResponseWriter, r *http.Request) {
	// parse img and args given as query params
	img := r.URL.Query().Get("img")
	givenArgs := r.URL.Query().Get("args")
	rawArgs := strings.Split(givenArgs, " ")
	args := rawArgs[:0]
	for _, arg := range rawArgs {
		if arg != "" {
			args = append(args, arg)
		}
	}

	// This will block for container to exit
	// TODO: can we stream stdout instead?
	cStdOut, cStdErr, err := DockerRunImg(img, args)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println("docker job failed", err)
		return
	}
	log.Println("--- out ---\n" + cStdOut)
	log.Println("--- err ---\n" + cStdErr)

	// TODO: Do the lambdas want stderr too?
	io.WriteString(w, cStdOut)
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("usage: %s <registry hostname> <registry port>\n", os.Args[0])
	}

	SetRegistry(os.Args[1], os.Args[2])

	http.HandleFunc("/runContainer", HandleCmd)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
