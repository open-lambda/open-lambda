package effe

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/tylerharter/open-lambda/lambda-generator/experimental/frontends"
)

const (
	templateUrl   = "https://raw.githubusercontent.com/siscia/effe/master/logic/logic.go"
	effeUrl       = "https://raw.githubusercontent.com/siscia/effe/master/effe.go"
	dockerfileUrl = "https://gist.githubusercontent.com/phonyphonecall/2fcb0e59dd9462d656b5/raw/bf9fc0716fbe689346c7e5dbde042058568c2fbe/Dockerfile"

	templateName   = "logic.go.template"
	effeName       = "effe.go.template"
	dockerfileName = "Dockerfile"
)

type FrontEnd struct {
	*frontends.BaseFrontEnd
	templatePath   string
	effePath       string
	dockerfilePath string
}

func NewFrontEnd(olDir string) *FrontEnd {
	return &FrontEnd{
		&frontends.BaseFrontEnd{
			Name:       "effe",
			OlDir:      olDir,
			ProjectDir: filepath.Dir(olDir),
		},
		filepath.Join(olDir, "frontends", "effe", templateName),
		filepath.Join(olDir, "frontends", "effe", effeName),
		filepath.Join(olDir, "frontends", "effe", dockerfileName),
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

	dstPath := path.Join(dir, handlerName+".go")
	copyFile(fe.templatePath, dstPath)
}

// Creates a temp wd, and moves to it
// Copies lambda, effe, and dockerfile in
// Does a docker build
func (fe *FrontEnd) BuildLambda(path string) {
	fe.doInit()
	path, err := filepath.Abs(path)
	if err != nil {
		fmt.Printf("could not create abs from path %s\n", path)
		os.Exit(1)
	}

	newDir, oldDir := fe.initScratchWd()
	defer func() {
		// change to old dir
		if err := os.Chdir(oldDir); err != nil {
			fmt.Printf("failed to chdir back to %s\n", oldDir)
			os.Exit(1)
		}
		// Remove scratch dir
		if err := os.RemoveAll(newDir); err != nil {
			fmt.Printf("failed to remove %s with err %v\n", newDir, err)
			os.Exit(1)
		}
	}()

	copyFile(fe.effePath, "effe.go")
	if err = os.Mkdir("logic", 0777); err != nil {
		fmt.Printf("cannot make package dir 'logic' with err %v", err)
		os.Exit(1)
	}
	copyFile(path, "logic/logic.go")
	copyFile(fe.dockerfilePath, "Dockerfile")

	tag, err := fe.GetId(path)
	if err != nil {
		fmt.Printf("failed to create id for lambda %s\n", path)
		os.Exit(1)
	}

	out, err := exec.Command("docker", "build", "-t", tag, ".").Output()
	if err != nil {
		fmt.Printf("failed to build docker img %s with output %s and err %v\n", tag, out, err)
	}
}

// Create new wd, change to it, and return old
func (fe *FrontEnd) initScratchWd() (newDir, oldDir string) {
	newDir = filepath.Join(fe.OlDir, "frontends", "effe", getRandomName())
	oldDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("failed to get wd with err %v\n", err)
		os.Exit(1)
	}

	if err = os.Mkdir(newDir, 0777); err != nil {
		fmt.Printf("failed to mkdir %s with err %v\n", newDir, err)
		os.Exit(1)
	}

	if err = os.Chdir(newDir); err != nil {
		fmt.Printf("failed to chdir to %s with err %v\n", newDir, err)
	}
	return newDir, oldDir
}

// initializes effe resources
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
			fmt.Printf("%s is file but expected directory!\n", effeDir)
			os.Exit(1)
		}
	}

	fe.getTemplates()
}

// Downloads templates to files
func (fe *FrontEnd) getTemplates() {
	// template
	if !exist(fe.templatePath) {
		download(templateUrl, fe.templatePath)
	}

	// effe
	if !exist(fe.effePath) {
		download(effeUrl, fe.effePath)
	}

	// docker
	if !exist(fe.dockerfilePath) {
		download(dockerfileUrl, fe.dockerfilePath)
	}
}

// Utils

// Copies src into new file dst
func copyFile(src, dst string) {
	dstFile, err := os.Create(dst)
	if err != nil {
		fmt.Printf("failed to create file %s with err %v", dst, err)
		os.Exit(1)
	}
	defer dstFile.Close()

	srcFile, err := os.Open(src)
	if err != nil {
		fmt.Printf("failed to open file %s with err %v", src, err)
		os.Exit(1)
	}
	defer srcFile.Close()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		fmt.Printf("failed to copy file to %s with err %v\n", dst, err)
		os.Exit(1)
	}
}

// download loads file at url to fileName
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

// checks if file exists
func exist(file string) bool {
	_, err := os.Stat(file)
	if err != nil {
		return false
	}
	return true
}

func getRandomName() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz")
	length := 10
	buff := make([]rune, length)
	for i := range buff {
		buff[i] = letters[rand.Intn(len(letters))]
	}
	return string(buff)
}
