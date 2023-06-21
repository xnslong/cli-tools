package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/alecthomas/kong"
)

var cli struct {
	Extract struct {
		Fields []string `name:"field" short:"f"`
	} `cmd:"extract"`
	Sort struct {
		Fields  []string `name:"field" short:"f" help:"fields to sort by. fields will be compared considering the field types set by --field-type option. if not set, the field will be considered a string field"`
		Reverse bool     `name:"reverse" short:"r" help:"in reverse order. i.e. the greater comes earlier"`
	} `cmd:"sort"`
	ToJson struct {
	} `cmd:"to-json"`

	In        string   `name:"input-file" short:"i" help:"input file, \"-\" for standard input" default:"-"`
	Out       string   `name:"output-file" short:"o" help:"output file, \"-\" for standard output" default:"-"`
	FieldType []string `name:"field-type" short:"t" help:"to specify the type of the field, in format <field>=<type>, means the field should be converted to the target type, available types are bool (b), int (i), float (f), string (str, s), if not set, it will not do conversion, just keep the original str type"`
	Format    string   `name:"separator" short:"s" help:"file type: csv fields separated by comma, tsv fields separated by tab" enum:"csv,tsv" default:"csv"`
}

var m map[string]jType
var sep rune

func main() {
	context := kong.Parse(&cli)

	switch cli.Format {
	case "csv":
		sep = ','
	case "tsv":
		sep = '\t'
	default:
		log.Fatalf("unsupported format: %q", cli.Format)
	}

	m = parseFieldConversionTypes(cli.FieldType)
	var (
		inFile  io.ReadCloser
		outFile io.WriteCloser
		err     error
	)

	inFile, err = openReadFile(cli.In)
	if err != nil {
		log.Fatalf("open input file error, file:%s, error: %v", cli.In, err)
	}
	defer inFile.Close()

	reader := csv.NewReader(inFile)
	reader.Comma = sep

	outFile, err = openWriteFile(cli.Out)
	if err != nil {
		log.Fatalf("open output file error, file:%s, error: %v", cli.Out, err)
	}
	defer outFile.Close()

	switch context.Command() {
	case "to-json":
		convertToJson(reader, outFile)
	case "sort":
		sortCsv(reader, outFile)
	case "extract":
		extract(reader, outFile)
	}
}

func extract(reader *csv.Reader, file io.WriteCloser) {
	writer := csv.NewWriter(file)
	defer writer.Flush()

	title, err := reader.Read()
	colIdx := getColIndex(title)

	idx := make([]int, 0, len(cli.Extract.Fields))
	for _, name := range cli.Extract.Fields {
		idx = append(idx, colIdx[name])
	}

	out := extractFields(title, idx)
	err = writer.Write(out)
	if err != nil {
		log.Fatalf("write csv error: %v", err)
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Fatalf("read csv error: %v", err)
		}

		out := extractFields(record, idx)

		err = writer.Write(out)
		if err != nil {
			log.Fatalf("write csv error: %v", err)
		}
	}

}

func extractFields(rec []string, idx []int) []string {
	result := make([]string, 0, len(idx))

	for _, i := range idx {
		result = append(result, rec[i])
	}

	return result
}

func convertToJson(reader *csv.Reader, out io.Writer) {
	title, err := reader.Read()
	if err != nil {
		log.Fatalf("read title error: %v", err)
	}

	for i := 0; ; i++ {
		record, err := reader.Read()
		if err == io.EOF {
			return
		}

		if err != nil {
			log.Fatalf("read record error: %v (line=%d, 0-based)", err, i)
		}

		if len(record) != len(title) {
			log.Fatalf("invalid record, column number not match the title line (record=%d, 0-based)", i)
		}

		recordMap, err := convertRecord(title, record, m)
		if err != nil {
			log.Fatalf("convert record error: (record=%d, 0-based) error: %v", i, err)
		}

		recordBytes, _ := json.Marshal(recordMap)
		_, err = fmt.Fprintf(out, "%s\n", recordBytes)
		if err != nil {
			log.Fatalf("write body error: %v", err)
		}
	}
}

func convertRecord(title []string, record []string, m map[string]jType) (map[string]interface{}, error) {
	recordMap := make(map[string]interface{}, len(title))
	for j := 0; j < len(title); j++ {
		fieldName := title[j]
		fieldValue := record[j]

		var afterFieldValue interface{}

		fieldConvType, ok := m[fieldName]
		if ok {
			v, err := convType(fieldValue, fieldConvType)
			if err != nil {
				return nil, fmt.Errorf("convert field error: (field: name=%s, value=%s, type=%s) %w", fieldName, fieldValue, fieldConvType, err)
			}
			afterFieldValue = v
		} else {
			afterFieldValue = fieldValue
		}

		recordMap[fieldName] = afterFieldValue
	}
	return recordMap, nil
}
