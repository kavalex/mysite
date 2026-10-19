// genhash is a dev utility: prints an Argon2id hash for a given password.
// Usage: go run ./cmd/genhash "<password>"
package main

import (
	"fmt"
	"os"

	"github.com/kavalex/mysite/internal/auth"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: genhash <password>")
		os.Exit(2)
	}
	h, err := auth.HashPassword(os.Args[1])
	if err != nil {
		panic(err)
	}
	fmt.Print(h)
}
