package packages


import (
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "sync"
    "sync/atomic"
    "regexp"
    "strconv"
    "bufio"
    "github.com/open-lambda/open-lambda/ol/common"
    "github.com/open-lambda/open-lambda/ol/worker/embedded"
    "github.com/open-lambda/open-lambda/ol/worker/sandbox"
)

// PackagePuller is the interface for installing pip packages locally.
// The manager installs to the worker host from an optional pip
// mirror.
type PackagePuller struct {
    sbPool    sandbox.SandboxPool
    depTracer *DepTracer

    // directory of lambda code that installs pip packages
    pipLambda string

    packages sync.Map
}

type Package struct {
    Name         string
    Meta         PackageMeta
    installMutex sync.Mutex
    installed    uint32
}

// the pip-install admin lambda returns this
type PackageMeta struct {
    Deps     []string `json:"Deps"`
    TopLevel []string `json:"TopLevel"`
}


type PackageInfo struct {
    Name string
    Size float64
}


var packages []PackageInfo
var totalPackageSize float64


func getUnusedDependencies(requirementsTxt string, unusedImports []string) ([]string, error) {
    var unusedDependencies []string

    file, err := os.Open(requirementsTxt)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    var prevLine string
    for scanner.Scan() {
        line := scanner.Text()
        if strings.Contains(line, "# via") {
            via := strings.TrimSpace(strings.Split(line, "# via")[1])
            for _, imp := range unusedImports {
                if imp == via {
                    unusedDependencies = append(unusedDependencies, prevLine)
                }
            }
        }
        prevLine = line
    }

    if err := scanner.Err(); err != nil {
        return nil, err
    }

    return unusedDependencies, nil
}

func getPythonImports(filepath string) ([]string, error) {
    file, err := os.Open(filepath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var imports []string
    scanner := bufio.NewScanner(file)
    importRegex := regexp.MustCompile(`^import (\S+)|^from (\S+) import`)

    for scanner.Scan() {
        line := scanner.Text()
        matches := importRegex.FindStringSubmatch(line)
        if matches != nil {
            if matches[1] != "" {
                imports = append(imports, matches[1])
            } else if matches[2] != "" {
                imports = append(imports, matches[2])
            }
        }
    }

    if err := scanner.Err(); err != nil {
        return nil, err
    }

    return imports, nil
}
func getPythonRequirements(filepath string) ([]string, error) {
    file, err := os.Open(filepath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var packages []string
    scanner := bufio.NewScanner(file)

    for scanner.Scan() {
        line := scanner.Text()
        // Ignore lines that are comments or empty
        if len(line) == 0 || line[0] == '#' {
            continue
        }
        // Split the line on '==' to separate the package name from the version
        parts := strings.Split(line, "==")
        packages = append(packages, parts[0])
    }

    if err := scanner.Err(); err != nil {
        return nil, err
    }

    return packages, nil
}
func getUnusedImports(importPath string, requirementsPath string) ([]string, error) {
    imports, err := getPythonImports(importPath)
    if err != nil {
        return nil, err
    }

    packages, err := getPythonRequirements(requirementsPath)
    if err != nil {
        return nil, err
    }

    // Convert the packages slice to a map for faster lookup
    importsMap := make(map[string]bool)
for _, imp := range imports {
    importsMap[imp] = true
}
//print out the importsMap


var unusedPackages []string
for _, pkg := range packages {
    if !importsMap[pkg] {
        unusedPackages = append(unusedPackages, pkg)
    }
}
    //print out the unused imports
    // for _, imp := range unusedImports {
    //     log.Printf("unused %s",imp)
    // }
requirementsPath = "/home/pjt07/open-lambda/myw/registry/scraper/requirements.txt"
unusedDependencies, err := getUnusedDependencies(requirementsPath, unusedPackages)
if err != nil {
    return nil, err
}
//print the unused dependencies to log
for _, dep := range unusedDependencies {
    log.Printf("Unused dependency: %s", dep)
}        

    return unusedDependencies, nil
}
func NewPackagePuller(sbPool sandbox.SandboxPool, depTracer *DepTracer) (*PackagePuller, error) {
    // create a lambda function for installing pip packages.  We do
    // each install in a Sandbox for two reasons:
    //
    // 1. packages may be malicious
    // 2. we want to install the right version, matching the Python
    //    in the Sandbox
    pipLambda := filepath.Join(common.Conf.Worker_dir, "admin-lambdas", "pip-install")
    if err := os.MkdirAll(pipLambda, 0700); err != nil {
        return nil, err
    }
    path := filepath.Join(pipLambda, "f.py")
    code := []byte(embedded.PackagePullerInstaller_py)
    if err := ioutil.WriteFile(path, code, 0600); err != nil {
        return nil, err
    }


    installer := &PackagePuller{
        sbPool:    sbPool,
        depTracer: depTracer,
        pipLambda: pipLambda,
    }


    return installer, nil
}


// From PEP-426: "All comparisons of distribution names MUST
// be case insensitive, and MUST consider hyphens and
// underscores to be equivalent."
func NormalizePkg(pkg string) string {
    return strings.ReplaceAll(strings.ToLower(pkg), "_", "-")
}


// "pip install" missing packages to Conf.Pkgs_dir
func (pp *PackagePuller) InstallRecursive(installs []string) ([]string, error) {
    // shrink capacity to length so that our appends are not
    // visible to caller
    installs = installs[:len(installs):len(installs)]


    installSet := make(map[string]bool)
    for _, install := range installs {
        name := strings.Split(install, "==")[0]
        installSet[name] = true
    }


    // Installs may grow as we loop, because some installs have
    // deps, leading to other installs
    for i := 0; i < len(installs); i++ {
        pkg := installs[i]
        if common.Conf.Trace.Package {
            log.Printf("On %v of %v", pkg, installs)
        }
        p, err := pp.GetPkg(pkg)
        if err != nil {
            return nil, err
        }


        if common.Conf.Trace.Package {
            log.Printf("Package '%s' has deps %v", pkg, p.Meta.Deps)
            log.Printf("Package '%s' has top-level modules %v", pkg, p.Meta.TopLevel)
        }


        // push any previously unseen deps on the list of ones to install
        for _, dep := range p.Meta.Deps {
            if !installSet[dep] {
                installs = append(installs, dep)
                installSet[dep] = true
            }
        }
    }


    return installs, nil
}


// GetPkg does the pip install in a Sandbox, taking care to never install the
// same Sandbox more than once.
//
// the fast/slow path code is tweaked from the sync.Once code, the
// difference being that may try the installed more than once, but we
// will never try more after the first success
func (pp *PackagePuller) GetPkg(pkg string) (*Package, error) {
    // get (or create) package
    pkg = NormalizePkg(pkg)
    tmp, _ := pp.packages.LoadOrStore(pkg, &Package{Name: pkg})
    p := tmp.(*Package)

    // fast path
    if atomic.LoadUint32(&p.installed) == 1 {
        return p, nil
    }

    // slow path
    p.installMutex.Lock()
    defer p.installMutex.Unlock()
    if p.installed == 0 {
        if err := pp.sandboxInstall(p); err != nil {
            return p, err
        }

        atomic.StoreUint32(&p.installed, 1)
        pp.depTracer.TracePackage(p)
        return p, nil
    }

    return p, nil
}

// sandboxInstall does the pip install within a new Sandbox, to a directory mapped from
// the host.  We want the package on the host to share with all, but
// want to run the install in the Sandbox because we don't trust it.
func (pp *PackagePuller) sandboxInstall(p *Package) (err error) {
    //need to check what packages are used in imports 
    //imports, err := getPythonImports("/home/pjt07/open-lambda/myw/registry/scraper/f.py")
    //packages, err := getPythonRequirements("/home/pjt07/open-lambda/myw/registry/scraper/requirements.in")
    //need to scan the directory for python files and check which imports are used, need to find the path for these files and the path for the requirements files
    unusedDependencies, err := getUnusedImports("/home/pjt07/open-lambda/myw/registry/scraper/f.py", "/home/pjt07/open-lambda/myw/registry/scraper/requirements.in")
    t := common.T0("pull-package")
    defer t.T1()

    // the pip-install lambda installs to /host, which is the the
    // same as scratchDir, which is the same as a sub-directory
    // named after the package in the packages dir
    scratchDir := filepath.Join(common.Conf.Pkgs_dir, p.Name)
    log.Printf("do pip install, using scratchDir='%v'", scratchDir)
    fileInfo, err := os.Stat(scratchDir)
    alreadyInstalled := false
    if _, err := os.Stat(scratchDir); err == nil {
        // assume dir existence means it is installed already
        log.Printf("%s appears already installed from previous run of OL", p.Name)
        alreadyInstalled = true
    } else {
        log.Printf("run pip install %s from a new Sandbox to %s on host", p.Name, scratchDir)
        if err := os.Mkdir(scratchDir, 0700); err != nil {
            return err
        }
    }
    
    //ADDED CODE
    //need to check which imports are used in files, if they are not used evict them
    cmd := exec.Command("du", "-sh", scratchDir)
    var out bytes.Buffer
    cmd.Stdout = &out
    err = cmd.Run()
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Disk usage of %s: %s\n", scratchDir, out.String())
    re := regexp.MustCompile(`^([\d\.]+[KMGTP]?)`)
    matches := re.FindStringSubmatch(out.String())
    if len(matches) < 2 {
        log.Fatal("Unexpected output from du command")
    }
    siz := matches[1]
    //convert sz to bytes
    var sz float64
    switch siz[len(siz)-1] {
    case 'K':
        sz = 1024
    case 'M':
        sz = 1024 * 1024
    case 'G':
        sz = 1024 * 1024 * 1024
    case 'T':
        sz = 1024 * 1024 * 1024 * 1024
    case 'P':
        sz = 1024 * 1024 * 1024 * 1024 * 1024
    default:
        sz = 1
    }
    var rest float64
    rest,err = strconv.ParseFloat(siz[:len(siz)-1], 64)
    if err != nil {
        log.Fatal(err)
    }
    sz = float64(sz) * rest / (1024 * 1024)
    log.Printf("Size in MB %s: %d\n", scratchDir, sz)
    totalPackageSize += sz
    log.Printf("Total package size: %d\n", totalPackageSize)
    //need to add the package to the list of packages
    packages = append(packages, PackageInfo{p.Name, sz})
    //TEST UNINSTALL   
    //check if the package's name is pandas, test uninstall method
    for _, dep := range unusedDependencies {
        if p.Name == dep {
            err = pp.Uninstall(p.Name)
            if err != nil {
                log.Printf("Error uninstalling package %s", p.Name)
            } else {
                return nil
            }
        }
    }
    //print out all the packages
    for _, p := range packages {
        log.Printf("Package: %s, Size: %d", p.Name, p.Size)
    }
    
    defer func() {
        if err != nil {
            os.RemoveAll(scratchDir)
        }
    }()

    meta := &sandbox.SandboxMeta{
        MemLimitMB: common.Conf.Limits.Installer_mem_mb,
    }
    log.Printf("the mem limit is %v", meta.MemLimitMB)
    //check if it is uninstalled, do not execute the sandbox

    sb, err := pp.sbPool.Create(nil, true, pp.pipLambda, scratchDir, meta, common.RT_PYTHON)
    if err != nil { 
        return err
    }
    defer sb.Destroy("package installation complete")

    // we still need to run a Sandbox to parse the dependencies, even if it is already installed
    msg := fmt.Sprintf(`{"pkg": "%s", "alreadyInstalled": %v}`, p.Name, alreadyInstalled)
    reqBody := bytes.NewReader([]byte(msg))

    // the URL doesn't matter, since it is local anyway
    req, err := http.NewRequest("POST", "http://container/run/pip-install", reqBody)
    if err != nil {
        return err
    }
    resp, err := sb.Client().Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return err
    }


    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("install lambda returned status %d, Body: '%s', Installer Sandbox State: %s",
            resp.StatusCode, string(body), sb.DebugString())
    }


    if err := json.Unmarshal(body, &p.Meta); err != nil {
        return err
    }


    for i, pkg := range p.Meta.Deps {
        p.Meta.Deps[i] = NormalizePkg(pkg)
    }


    return nil
}

//need a method to uninstall a package 
func (pp *PackagePuller) Uninstall(pkg string) error {
    pkg = NormalizePkg(pkg)
    //remove the package from the list of packages
    for i, p := range packages {
        if p.Name == pkg {
            packages = append(packages[:i], packages[i+1:]...)
            break
        }
    }
    //remove the package from the packages directory
    err := os.RemoveAll(filepath.Join(common.Conf.Pkgs_dir, pkg))
    if err != nil {
        return err
    }
    return nil
}
