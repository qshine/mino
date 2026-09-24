package main

import (
	"os"

	mino "github.com/qshine/mino/internal"
)

// Release builds set this from the Git tag; source builds report dev.
var version = "dev"

func main() {
	os.Exit(mino.Main(version))
}
