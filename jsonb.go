package jsonb

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Order is taking from RFC7159
var values = map[string]byte{
	"false":  0x1,
	"null":   0x2,
	"true":   0x3,
	"object": 0x4,
	"array":  0x5,
	"number": 0x6,
	"string": 0x7,
}

type opType byte

const (
	opLITERAL opType = iota
	opSTRING
	opNUMBER
	opARRAY
	opOBJECT
	opKEY
	opKV_SEPERATOR
)

var opNames = [...]string{"Literal", "String", "Number", "Array", "Object", "Object Key", "Object Key-Value seperator"}

type operation struct {
	Type       opType
	StartIndex int
}

func Encode(json string) ([]byte, error) {
	jsonb := []byte{}
	op := []operation{{opLITERAL, 0}}

	var c byte
	var l int

	for i, jsonl := 0, len(json); i < jsonl; i++ {
		c = json[i]
		l = len(op)
		if l < 1 {
			if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
				continue
			}
			return nil, fmt.Errorf("Exceesive characters at %d", i)
		}

		switch op[l-1].Type {
		case opLITERAL:
			if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
				continue
			}
			switch c {
			case 'f':
				if jsonl < i+5 {
					return nil, fmt.Errorf("Expected keyword 'false' at %d but string length isn't sufficient", i)
				}
				if json[i:i+5] != "false" {
					return nil, fmt.Errorf("Expected keyword 'false' at %d but found '%s'", i, json[i:i+5])
				}
				jsonb = append(jsonb, values["false"])
				i += 4
				op = op[:l-1]
			case 't':
				if jsonl < i+4 {
					return nil, fmt.Errorf("Expected keyword 'true' at %d but string length isn't sufficient", i)
				}
				if json[i:i+4] != "true" {
					return nil, fmt.Errorf("Expected keyword 'true' at %d but found '%s'", i, json[i:i+4])
				}
				jsonb = append(jsonb, values["true"])
				i += 3
				op = op[:l-1]
			case 'n':
				if jsonl < i+4 {
					return nil, fmt.Errorf("Expected keyword 'null' at %d but string length isn't sufficient", i)
				}
				if json[i:i+4] != "null" {
					return nil, fmt.Errorf("Expected keyword 'null' at %d but found '%s'", i, json[i:i+4])
				}
				jsonb = append(jsonb, values["null"])
				i += 3
				op = op[:l-1]
			case '"':
				jsonb = append(jsonb, values["string"])
				op[l-1].Type = opSTRING
				op[l-1].StartIndex = i
			case '[':
				jsonb = append(jsonb, values["array"])
				op[l-1].Type = opARRAY
				op[l-1].StartIndex = i
				op = append(op, operation{opLITERAL, i + 1})
			case '{':
				jsonb = append(jsonb, values["object"])
				op[l-1].Type = opOBJECT
				op[l-1].StartIndex = i
				op = append(op, operation{opKEY, i + 1})
			case ']':
				if l > 1 && op[l-2].Type == opARRAY {
					jsonb = append(jsonb, 0x00)
					op = op[:l-2]
				}
			default:
				if c == '-' || (c > 47 && c < 58) {
					jsonb = append(jsonb, values["number"])
					op[l-1].Type = opNUMBER
					op[l-1].StartIndex = i
				} else {
					return nil, fmt.Errorf("Unexpected character %s at %d", string(c), i)
				}
			}
		case opSTRING:
			switch c {
			case 0x00:
				return nil, fmt.Errorf("Unexpected null byte at %d", i)
			case '"':
				jsonb = append(jsonb, []byte(json[op[l-1].StartIndex+1:i])...)
				jsonb = append(jsonb, 0x00)
				op = op[:l-1]
			}
		case opNUMBER:
			if (c > 47 && c < 58) || c == '.' {
				continue
			}
			num, err := strconv.ParseFloat(json[op[l-1].StartIndex:i], 64)
			if err != nil {
				return nil, fmt.Errorf("Invalid number at %d", op[l-1].StartIndex)
			}
			jsonb = binary.BigEndian.AppendUint64(jsonb, math.Float64bits(num))
			op = op[:l-1]
			i--
		case opARRAY:
			if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
				continue
			}
			switch c {
			case ']':
				jsonb = append(jsonb, 0x00)
				op = op[:l-1]
			case ',':
				op = append(op, operation{opLITERAL, i + 1})
			default:
				return nil, fmt.Errorf("Expected values seperator or array closing braces, but found %s at %d", string(c), i)
			}
		case opKEY:
			if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
				continue
			}
			switch c {
			case '}':
				jsonb = append(jsonb, 0x00)
				op = op[:l-1]
			case '"':
				op[l-1].Type = opKV_SEPERATOR
				op = append(op, operation{opSTRING, i})
			default:
				return nil, fmt.Errorf("Expected new Key-Value pairs or object closing brackets, but found %s at %d", string(c), i)
			}
		case opKV_SEPERATOR:
			if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
				continue
			}
			if c == ':' {
				op[l-1].Type = opLITERAL
				continue
			}
			return nil, fmt.Errorf("Expected Key-Value seperator, but found %s at %d", string(c), i)
		case opOBJECT:
			if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
				continue
			}
			switch c {
			case '}':
				jsonb = append(jsonb, 0x00)
				op = op[:l-1]
			case ',':
				op = append(op, operation{opKEY, i + 1})
			default:
				return nil, fmt.Errorf("Expected new Key-Value pairs or object closing brackets, but found %s at %d", string(c), i)
			}
		}
	}

	if len(op) > 0 {
		if len(op) == 1 && op[0].Type == opNUMBER {
			num, err := strconv.ParseFloat(json[op[0].StartIndex:], 64)
			if err != nil {
				return nil, fmt.Errorf("Invalid number at %d", op[0].StartIndex)
			}

			jsonb = binary.BigEndian.AppendUint64(jsonb, math.Float64bits(num))
		} else {
			return nil, fmt.Errorf("Unterminated operation %s, started at %d", opNames[op[len(op)-1].Type], op[len(op)-1].StartIndex)
		}
	}

	return jsonb, nil
}

func Decode(jsonb []byte) (string, error) {
	json := ""
	op := []operation{{opLITERAL, 0}}

	var c byte
	var l int

	for i, jsonl := 0, len(jsonb); i < jsonl; i++ {
		c = jsonb[i]
		l = len(op)

		if l < 1 {
			return "", fmt.Errorf("Exceesive bytes at %d", i)
		}

		switch op[l-1].Type {
		case opLITERAL:
			switch c {
			case values["string"]:
				op[l-1].Type = opSTRING
				op[l-1].StartIndex = i + 1
			case values["false"]:
				json += "false"
				op = op[:l-1]
			case values["true"]:
				json += "true"
				op = op[:l-1]
			case values["null"]:
				json += "null"
				op = op[:l-1]
			case values["number"]:
				numArr := jsonb[i+1 : i+9]
				numU64 := binary.BigEndian.Uint64(numArr)
				num := math.Float64frombits(numU64)
				json += fmt.Sprint(num)
				op = op[:l-1]
				i += 8
			case values["array"]:
				json += "["
				op[l-1].Type = opARRAY
				op = append(op, operation{opLITERAL, i + 1})
			case values["object"]:
				json += "{"
				op[l-1].Type = opOBJECT
				op = append(op, operation{opKEY, i + 1})
			case 0x00:
				if l > 1 && op[l-2].Type == opARRAY {
					json = strings.TrimSuffix(json, ",")
					json += "]"
					op = op[:l-2]
					continue
				}
				fallthrough
			default:
				return "", fmt.Errorf("Unknown byte %#.2x at %d", c, i)
			}
		case opSTRING:
			if c == 0x00 {
				str := jsonb[op[l-1].StartIndex:i]
				json += "\"" + string(str) + "\""
				op = op[:l-1]
			}
		case opARRAY:
			json += ","
			op = append(op, operation{opLITERAL, i})
			i--
		case opOBJECT:
			json += ","
			op = append(op, operation{opKEY, i})
			i--
		case opKEY:
			if c == 0x00 {
				json = strings.TrimSuffix(json, ",")
				json += "}"
				op = op[:l-2]
			} else {
				op[l-1].Type = opKV_SEPERATOR
				op = append(op, operation{opSTRING, i})
			}
		case opKV_SEPERATOR:
			json += ":"
			op[l-1].Type = opLITERAL
			op[l-1].StartIndex = i
			i--
		}
	}

	if len(op) > 0 {
		return "", fmt.Errorf("Unterminated operation %s, started at %d", opNames[op[len(op)-1].Type], op[len(op)-1].StartIndex)
	}

	return json, nil
}
