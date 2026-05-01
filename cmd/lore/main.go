package main

import (
	"fmt"
	"os"

	"github.com/zpenka/lore"
)

func main() {
	if err := lore.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "lore:", err)
		os.Exit(1)
	}
}
