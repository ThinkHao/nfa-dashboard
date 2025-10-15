package main

import (
	"fmt"
	"os"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: hashpass <password> [cost]")
		os.Exit(1)
	}
	pwd := os.Args[1]
	cost := bcrypt.DefaultCost
	if len(os.Args) >= 3 {
		if c, err := strconv.Atoi(os.Args[2]); err == nil {
			cost = c
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), cost)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(hash))
}
