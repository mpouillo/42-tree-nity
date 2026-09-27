package main

import (
	// "flag"
	"os"
	"fmt"
	
)

func main() {
	// flagset := flag.NewFlagSet("subscribe", flag.ContinueOnError)
	args := os.Args[1:]
	argc := len(args)
	// raw := flagset.Bool("raw", false, "binary format")    // *bool
	// prefix := flagset.String("prefix", "", "key prefix")  // *string
	// offset := flagset.Int("offset", 0, "starting offset") // *int
	// err := flagset.Parse(os.Args[1:])
	// if argc < 2 {
	// 	os.Exit()
	// }
	fmt.Println(args, argc)
}
