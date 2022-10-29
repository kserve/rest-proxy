package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf16"
	"unicode/utf8"

	gw "github.com/kserve/rest-proxy/gen"
)

// This file contains logic related to marshalling/unmarshalling BYTES type tensor data
// "raw" parsing is done for performance and structural validation is done only on a best-effort basis

var escMap = map[byte]byte{'b': '\b', 'f': '\f', 'r': '\r', 't': '\t', 'n': '\n', '\\': '\\', '/': '/', '"': '"'}

func unmarshalBytesJson(target *[][]byte, shape []int64, b64 bool, data []byte) error {
	// Cases:
	// 1dim raw   ( [ [ N ) -> as-is
	// flat raw   ( [ [ N ) -> as-is
	// nested raw ( [ [ [ .. N ) -> strip "middles" (use shape prob)
	// 1dim str   ( [ " )
	// flat str   ( [ " )
	// nested str ( [ [ .. " ) -> strip all

	start := -1
	depth := 0
	isString := false
	for i, b := range data {
		if b == '[' {
			if start == -1 {
				start = i
			}
			depth += 1
		} else if !isSpace(b) {
			isString = b == '"'
			break
		}
	}
	if depth == 0 {
		return errors.New("invalid tensor data: not a json array")
	}
	if start > 0 {
		data = data[start:]
	}
	if isString {
		if depth != 1 && depth != len(shape) {
			return errors.New("data array nesting does not match tensor shape")
		}
		return unmarshalStringArray(target, shape, b64, data)
	}
	if depth <= 1 {
		return errors.New("invalid tensor data: must be an array of byte arrays")
	}
	if depth == 2 {
		// flat numeric case, e.g.  [[1,2,3],[4,5,6],[7,8,9]]
		return json.Unmarshal(data, target)
	}

	// nested numeric case, e.g.  [[[1,2],[3,4]],[[5,6],[7,8]]]
	// ignore innermost dimension because elements are lists of bytes
	if (depth - 1) != len(shape) {
		return errors.New("invalid tensor data: array nesting does not match tensor shape")
	}
	return unmarshalNestedNumeric(target, depth, data)
}

// nested numeric case, e.g.  [[[1,2],[3,4]],[[5,6],[7,8]]]
func unmarshalNestedNumeric(target *[][]byte, depth int, data []byte) error {
	d := 0
	j := 1
	for _, b := range data {
		include := true
		if b == '[' {
			d++
			if d > depth {
				return errors.New("invalid tensor data: array nesting does not match tensor shape")
			}
			include = d == depth
		} else if b == ']' {
			include = d == depth
			d--
		}
		if include {
			data[j] = b
			j++
		}
	}
	if d != 0 {
		return errors.New("invalid tensor data: array nesting does not match tensor shape")
	}
	data[j] = ']'
	return json.Unmarshal(data[:j+1], target)
}

func unmarshalStringArray(target *[][]byte, shape []int64, b64 bool, data []byte) error {
	elems := int(elementCount(shape))
	t := make([][]byte, 0, elems)

	depth := 0
	strStart := -1
	j := 0
	var ok bool
	l := len(data)
	for i := 0; i < l; i++ {
		b := data[i]
		if strStart == -1 {
			if b == '[' {
				depth++
			} else if b == ']' {
				depth--
			} else if b == '"' {
				if len(t) >= elems {
					return errors.New("more strings than expected for tensor shape")
				}
				strStart = i
				j = i + 1
			} else if b != ',' && !isSpace(b) {
				return errors.New("tensor data must be a flat or nested json array of strings")
			}
			continue
		}
		// here we are mid-string
		if b == '\\' {
			i++
			if i == l {
				break // will error with unexpected end
			}
			b = data[i]
			if b == 'u' {
				i += 4
				if i >= l {
					break // will error with unexpected end
				}
				cp := utf16.Decode([]uint16{binary.BigEndian.Uint16(data[i-3 : i+1])})
				for _, r := range cp {
					j += utf8.EncodeRune(data[j:], r)
				}
				continue
			} else if b, ok = escMap[b]; !ok {
				return errors.New("invalid escaped char in json string")
			}
		} else if b == '"' {
			//end of string
			s := data[strStart+1 : j]
			if b64 {
				if n, err := base64.StdEncoding.Decode(s, s); err != nil {
					return fmt.Errorf("error decoding json string as base64: %w", err)
				} else {
					s = s[:n]
				}
			}
			t = append(t, s)
			strStart = -1
		}
		if j != i {
			data[j] = b
		}
		j++
	}
	if strStart != -1 {
		return errors.New("fewer strings than expected for tensor shape")
	}
	if depth != 0 {
		return errors.New("invalid tensor data: invalid nested json arrays")
	}

	*target = t
	return nil
}

func isBase64Content(parameters map[string]*gw.InferParameter) bool {
	ct := parameters[CONTENT_TYPE].GetStringParam()
	if ct == "" || ct == "utf8" || ct == "str" || ct == "UTF8" {
		return false
	}
	if ct == BASE64 || ct == "b64" || ct == "BASE64" || ct == "B64" {
		return true
	}
	if ct != "utf-8" && ct != "UTF-8" {
		logger.Error(nil, "Unrecognized content_type, treating as utf8", CONTENT_TYPE, ct)
	}
	return false
}

// Split raw bytes into separate byte arrays based on 4-byte size delimeters
func splitRawBytes(raw []byte, expectedSize int) ([][]byte, error) {
	off, length := int64(0), int64(len(raw))
	strings := make([][]byte, expectedSize)
	r := bytes.NewReader(raw)
	for i := 0; i < expectedSize; i++ {
		var size uint32
		if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
			return nil, errors.New("unexpected end of raw tensor bytes")
		}
		start := off + 4
		if off, _ = r.Seek(int64(size), io.SeekCurrent); off > length {
			return nil, errors.New("unexpected end of raw tensor bytes")
		}
		strings[i] = raw[start:off]
	}
	if off < length {
		return nil, errors.New("more raw tensor bytes than expected")
	}
	return strings, nil
}
