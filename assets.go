// Package assets embeds the installer, default identity, and built-in skills for
// standalone builds.
package assets

import "embed"

// InstallerScript keeps updates independent of the checkout and working directory.
//
//go:embed install.sh
var InstallerScript string

// DefaultSoul seeds the user-editable identity on the first successful startup.
//
//go:embed SOUL.md
var DefaultSoul string

// BuiltinSkills holds the skills bundled with Mino, one skills/<name>/SKILL.md
// each. A project skill of the same name replaces one.
//
//go:embed skills
var BuiltinSkills embed.FS
