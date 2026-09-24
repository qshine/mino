// Package installer embeds the release installer for the application's updater.
package installer

import _ "embed"

// Script keeps updates independent of the checkout and working directory.
//
//go:embed install.sh
var Script string
