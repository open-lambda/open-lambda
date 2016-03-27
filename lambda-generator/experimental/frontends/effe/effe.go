package effe

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/tylerharter/open-lambda/lambda-generator/experimental/frontends"
)

const (
	templateUrl = "https://raw.githubusercontent.com/siscia/effe/master/logic/logic.go"
	effeUrl     = "https://raw.githubusercontent.com/siscia/effe/master/effe.go"

	templateName = "logic.go.template"
	effeName     = "effe.go.template"
)

type FrontEnd struct {
	*frontends.BaseFrontEnd

	templatePath string
	effePath     string
}

func NewFrontEnd(olDir string) *FrontEnd {
	return &FrontEnd{
		&frontends.BaseFrontEnd{
			Name:  "effe",
			OlDir: olDir,
		},
		filepath.Join(olDir, "frontends", "effe", templateName),
		filepath.Join(olDir, "frontends", "effe", effeName),
	}
}

// given: my/next/handler
// creates $WORKING_DIR/my/next/handler.go
//
// handler.go will contain a "Hello World" effe
func (fe *FrontEnd) AddLambda(location string) {
	fe.doInit()

	if location != path.Clean(location) {
		fmt.Printf("bad location\n")
		os.Exit(1)
	}
	if location == "." {
		fmt.Printf("handler must have a name\n")
		os.Exit(1)
	}

	// TODO: validate handlerName
	handlerName := path.Base(location)
	dir := path.Dir(location)
	fmt.Printf("creating %s.go in %s\n", handlerName, dir)
	err := os.Mkdir(dir, 0777)
	if err != nil {
		if !os.IsExist(err) {
			fmt.Printf("failed to create dir %s\n", dir)
			os.Exit(1)
		}
	}

	//TODO: lay down template, copy contents
	filePath := path.Join(dir, handlerName+".go")
	f, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("failed to create file %s with err %v", filePath, err)
		os.Exit(1)
	}
	defer f.Close()

	template, err := os.Open(fe.templatePath)
	if err != nil {
		fmt.Printf("failed to open file %s with err %v", fe.templatePath, err)
		os.Exit(1)
	}
	defer template.Close()

	if _, err = io.Copy(f, template); err != nil {
		fmt.Printf("failed to copy template to %s with err %v\n", filePath, err)
		os.Exit(1)
	}
}

func (fe *FrontEnd) doInit() {
	effeDir := filepath.Join(fe.OlDir, "frontends", "effe")
	info, err := os.Stat(effeDir)
	if err != nil {
		if os.IsNotExist(err) {
			if err = os.Mkdir(effeDir, 0777); err != nil {
				fmt.Printf("failed to create effe dir %s with err %v\n", effeDir, err)
				os.Exit(1)
			}
		}
	} else {
		if !info.IsDir() {
			log.Printf("%s is file but expected directory!\n", effeDir)
			os.Exit(1)
		}
	}

	fe.getTemplates()
}

// Downloads template to file
func (fe *FrontEnd) getTemplates() {
	// template
	if !exist(fe.templatePath) {
		download(templateUrl, fe.templatePath)
	}

	// effe
	if !exist(fe.effePath) {
		download(effeUrl, fe.effePath)
	}
}

func exist(file string) bool {
	_, err := os.Stat(file)
	if err != nil {
		return false
	}
	return true
}

func download(url, fileName string) {
	fmt.Printf("downloading %s\n", fileName)
	f, err := os.Create(fileName)
	if err != nil {
		fmt.Printf("failed to create template file %s with err %v\n", fileName, err)
		os.Exit(1)
	}
	defer f.Close()

	r, err := http.Get(url)
	if err != nil {
		fmt.Printf("failed to get template with err %v\n", err)
		os.Exit(1)
	}
	defer r.Body.Close()

	_, err = io.Copy(f, r.Body)
	if err != nil {
		fmt.Printf("failed to copy response to template file with err %v\n", err)
	}
}
