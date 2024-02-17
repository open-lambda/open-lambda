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
    Size int64
}


var packages []PackageInfo




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
    t := common.T0("pull-package")
    defer t.T1()

    // the pip-install lambda installs to /host, which is the the
    // same as scratchDir, which is the same as a sub-directory
    // named after the package in the packages dir
    scratchDir := filepath.Join(common.Conf.Pkgs_dir, p.Name)
    log.Printf("%sPkgs_dir:",common.Conf.Pkgs_dir)
    log.Printf("%sp.Name:", p.Name)
    log.Printf("do pip install, using scratchDir='%v'", scratchDir)
    fileInfo, err := os.Stat(scratchDir)
    if err == nil {
        // assume dir existence means it is installed already
        //print the size of the package and fileInfo.Size()
        log.Printf("The size of the package %s is %v", fileInfo.Name(), fileInfo.Size())
    }
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
	other := matches[0]
    log.Printf("Size of %s: %s and size of other: %s\n", scratchDir, siz, other)
    //convert sz to bytes
    var sz int64
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
    sz = int64(float64(sz) * rest)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Size in bytes %s: %d\n", scratchDir, sz)

    //need to add the package to the list of packages
    packages = append(packages, PackageInfo{p.Name, sz})
    //TEST UNINSTALL   
    //check if the package's name is pandas, test uninstall method
    if p.Name == "pandas==1.5.0"{
        err = pp.Uninstall(p.Name)
        if err != nil {
            log.Printf("Error uninstalling package %s", p.Name)
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
