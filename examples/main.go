package main

import (
	"fmt"
	"github.com/amirdlt/ffvm"
)

type Something struct {
	FirstName string `ffvm:"upper;lower,empty;upper;min_len=10;len=100"`
	Age       int    `ffvm:",max=5;min=11"`
}

func main() {
	st := Something{FirstName: "Amir", Age: 10}
	fmt.Println(ffvm.Validate(&st))
	fmt.Println(st)
}
