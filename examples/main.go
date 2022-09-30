package main

import (
	"fmt"
	"github.com/amirdlt/ffvm"
)

type Something struct {
	FirstName string `ffvm:"upper;lower,empty;upper"`
	Age       int    `ffvm:"upper;lower,empty;upper"`
}

func main() {
	parser := ffvm.NewParser()
	st := Something{FirstName: "Amir", Age: 10}
	fmt.Println(parser.Act(&st))
	fmt.Println(st)
}
