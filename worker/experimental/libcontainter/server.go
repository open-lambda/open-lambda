package main

import (
	"log"
	"net/http"
	"strings"
)

func HandleCmd(w http.ResponseWriter, r *http.Request) {
	// parse img and args given as query params
	img := r.URL.Query().Get("img")
	givenArgs := r.URL.Query().Get("args")
	log.Printf("running img %s with args %s\n", img, givenArgs)
	rawArgs := strings.Split(givenArgs, " ")
	args := rawArgs[:0]
	for _, arg := range rawArgs {
		if arg != "" {
			args = append(args, arg)
		}
	}

	// This will block for container to exit
	// TODO: can we stream stdout instead?
	RunContainer(img, args)
	// cStdOut, cStdErr, err := DockerRunImg(img, args)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	log.Println("docker job failed", err)
	// 	return
	// }
	// log.Println("--- out ---\n" + cStdOut)
	// log.Println("--- err ---\n" + cStdErr)

	// TODO: Do the lambdas want stderr too?
	// io.WriteString(w, cStdOut)
}

func main() {
	log.Println("hello world")
	http.HandleFunc("/runContainer", HandleCmd)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
