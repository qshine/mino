package main

import (
	"os"

	"github.com/qshine/mino/internal/mino"
)

// Release builds set this from the Git tag; source builds report dev.
var version = "dev"

func main() {
	os.Exit(mino.Main(version))
}
