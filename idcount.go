package main

import "fmt"

var insIdGen = IdGenerator()

func IdGenerator() func() string {
	var a int
	a = 0

	return func() string {
		a++
		return fmt.Sprintf("id-%d", a)
	}
}
