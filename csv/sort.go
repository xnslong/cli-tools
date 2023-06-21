package main

import (
	"encoding/csv"
	"io"
	"log"
	sort2 "sort"
)

func sort(info *CsvInfo) {
	var fieldIdx = getColIndex(info.header)

	var comparators []comparator
	for _, field := range cli.Sort.Fields {
		comparators = append(comparators, cmpField(info.Records, fieldIdx[field], m[field]))
	}
	cmp := seqCmp(comparators...)
	if cli.Sort.Reverse {
		cmp = reverse(cmp)
	}

	sort2.Slice(info.Records, func(i, j int) bool {
		return cmp(i, j) <= 0
	})
}

func sortCsv(reader *csv.Reader, out io.Writer) {
	csvInfo, err := readCsv(reader)
	requireNoError(err, "sortCsv error")

	sort(csvInfo)

	writeCsv(csvInfo, out)
}

func cmpField(rec []CsvRecord, idx int, fType jType) comparator {
	return func(i, j int) int {
		v1, err := convType(rec[i].Fields[idx], fType)
		requireNoError(err, "index: %d, value: %s convert to %s error", idx, rec[i].Fields[idx], fType)

		v2, err := convType(rec[j].Fields[idx], fType)
		requireNoError(err, "index: %d, value: %s convert to %s error", idx, rec[j].Fields[idx], fType)

		switch fType {
		case jTypeBool:
			return cmpValue(boolAsInt(v1.(bool)), boolAsInt(v2.(bool)))
		case jTypeInt:
			return cmpValue(v1.(int64), v2.(int64))
		case jTypeFloat:
			return cmpValue(v1.(float64), v2.(float64))
		case jTypeStr:
			return cmpValue(v1.(string), v2.(string))
		default:
			log.Fatalf("unsupported field type: %s", fType)
			return 0
		}
	}
}
