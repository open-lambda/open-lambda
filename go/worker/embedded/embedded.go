// Package embedded lets us include some Python code and other files directly inside
// the binary so that ol is a standalone deployable.
package embedded

import _ "embed"

// Default tree of 40 Zygotes nodes (with nodes for Pandas, etc)

//go:embed default-zygotes-40.json
var DefaultZygotes40_json string

// Used by github.com/open-lambda/open-lambda/ol/worker/lambda/packages
//
// We invoke this lambda to do the pip install in a Sandbox.
//
// The install is not recursive (it does not install deps), but it
// does parse and return a list of deps, based on a rough
// approximation of the PEP 508 format.  We ignore the "extra" marker
// and version numbers (assuming the latest).

//go:embed packagePullerInstaller.py
var PackagePullerInstaller_py string
