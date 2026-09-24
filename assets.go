// Package assets embeds the installer and default identity for standalone builds.
package assets

import _ "embed"

// InstallerScript keeps updates independent of the checkout and working directory.
//
//go:embed install.sh
var InstallerScript string

// DefaultSoul seeds the user-editable identity on the first successful startup.
//
//go:embed SOUL.md
var DefaultSoul string
