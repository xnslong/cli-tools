package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/alecthomas/kong"
)

var cli struct {
	In      string   `name:"input-file" short:"i" help:"input file, \"-\" for standard input" default:"-"`
	Out     string   `name:"output-file" short:"o" help:"output file, \"-\" for standard output" default:"-"`
	Convert []string `name:"convert" short:"c" help:"to specify the type of the field, in format <field>=<type>, means the field should be converted to the target type, available types are bool, int, float, str, if not set, it will not do conversion, just keep the original str type"`
}

type jType int

func (j jType) String() string {
	switch j {
	case jTypeBool:
		return "bool"
	case jTypeInt:
		return "int"
	case jTypeFloat:
		return "float"
	case jTypeStr:
		return "string"
	default:
		return "<unknown>"
	}
}

const (
	jTypeStr = jType(iota)
	jTypeBool
	jTypeInt
	jTypeFloat
)

func main() {
	kong.Parse(&cli)

	m := parseFieldConversionTypes(cli.Convert)

	inFile, err := openReadFile(cli.In)
	if err != nil {
		log.Fatalf("open input file error, file:%s, error: %v", cli.In, err)
	}
	defer inFile.Close()

	outFile, err := openWriteFile(cli.Out)
	if err != nil {
		log.Fatalf("open output file error, file:%s, error: %v", cli.Out, err)
	}
	defer outFile.Close()

	reader := csv.NewReader(inFile)
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
		_, err = fmt.Fprintf(outFile, "%s\n", recordBytes)
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

func parseFieldConversionTypes(convert []string) map[string]jType {
	m := make(map[string]jType)
	for _, spec := range convert {
		split := strings.SplitN(spec, "=", 2)
		if len(split) != 2 {
			log.Fatalf("invalid option: %s", spec)
		}

		field := split[0]
		typeStr := split[1]
		fieldType, err := parseJType(typeStr, spec)
		if err != nil {
			log.Fatalf("invalid field type spec %s: %v", spec, err)
		}
		m[field] = fieldType
	}
	return m
}

func parseJType(typeStr string, spec string) (jType, error) {
	var fieldType jType
	switch typeStr {
	case "str", "string":
		fieldType = jTypeStr
	case "int", "i":
		fieldType = jTypeInt
	case "bool", "b", "boolean":
		fieldType = jTypeBool
	case "float", "f", "double":
		fieldType = jTypeFloat
	default:
		return 0, fmt.Errorf("unsupported field type %q for field", typeStr)
	}
	return fieldType, nil
}

func convType(value string, convType jType) (interface{}, error) {
	switch convType {
	case jTypeBool:
		v, err := strconv.ParseBool(value)
		return v, err
	case jTypeFloat:
		v, err := strconv.ParseFloat(value, 64)
		return v, err
	case jTypeInt:
		v, err := strconv.ParseInt(value, 10, 64)
		return v, err
	case jTypeStr:
		return value, nil
	default:
		return nil, errors.New("unsupported type")
	}
}

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
