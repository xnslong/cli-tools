package main

import "os"

func openReadFile(file string) (*os.File, error) {
	if file == "-" {
		return os.Stdin, nil
	}

	return os.Open(file)
}

func openWriteFile(file string) (*os.File, error) {
	if file == "-" {
		return os.Stdout, nil
	}

	return os.OpenFile(file, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
}
