package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
)

func writeCsv(info *CsvInfo, out io.Writer) {
	writer := csv.NewWriter(out)
	writer.Write(info.header)

	for _, record := range info.Records {
		writer.Write(record.Fields)
	}

	writer.Flush()
}

func requireNoError(err error, format string, args ...any) {
	if err != nil {
		log.Fatal(fmt.Sprintf(format, args...), err)
	}
}

func readCsv(in *csv.Reader) (*CsvInfo, error) {
	in.ReuseRecord = false

	info := &CsvInfo{}

	rec, err := in.Read()
	if err != nil {
		return nil, err
	}

	info.header = rec

	for {
		rec, err := in.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		info.Records = append(info.Records, CsvRecord{Fields: rec})
	}

	return info, nil

}

type CsvInfo struct {
	header  []string
	Records []CsvRecord
}

type CsvRecord struct {
	Fields []string
}

func getColIndex(title []string) map[string]int {
	var colIdx = make(map[string]int)
	for i, name := range title {
		colIdx[name] = i
	}
	return colIdx
}
