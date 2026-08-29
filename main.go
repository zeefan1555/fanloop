package main

import (
	"context"
	"os"

	"github.com/zeefan1555/commonloop/cmd"
)

func main() {
	os.Exit(cmd.Execute(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
