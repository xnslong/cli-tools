package main

import (
	"encoding/hex"
	"flag"
	"io"
	"os"
)

var doDecode bool = false

func main() {
	flag.BoolVar(&doDecode, "d", false, "")
	flag.Parse()

	if doDecode {
		decode()
	} else {
		encode()
	}
}

func encode() {
	encoder := hex.NewEncoder(os.Stdout)
	io.Copy(encoder, os.Stdin)
}

func decode() {
	decoder := hex.NewDecoder(os.Stdin)
	io.Copy(os.Stdout, decoder)
}
