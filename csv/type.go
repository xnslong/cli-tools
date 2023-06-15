package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
)

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
	case "string", "str", "s":
		fieldType = jTypeStr
	case "int", "i":
		fieldType = jTypeInt
	case "bool", "b", "boolean":
		fieldType = jTypeBool
	case "float", "f", "double", "d":
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
