package main

import (
	"fmt"

	"github.com/abenz1267/elephant/internal/util"
)

func PrintDoc() {
	fmt.Println(readme)
	fmt.Println()
	util.PrintConfig(Config{}, Name)
}
