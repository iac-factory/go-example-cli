package main

import (
	"cli/internal/fs/tree"
	"fmt"
)

func main() {
	t := tree.New("./internal")

	fmt.Println(t)
	fmt.Println(t.YAML())
}
