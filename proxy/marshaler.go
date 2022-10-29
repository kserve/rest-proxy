/*
Copyright 2021 IBM Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	gw "github.com/kserve/rest-proxy/gen"
)

const (
	BOOL   = "BOOL"
	UINT8  = "UINT8"
	UINT16 = "UINT16"
	UINT32 = "UINT32"
	UINT64 = "UINT64"
	INT8   = "INT8"
	INT16  = "INT16"
	INT32  = "INT32"
	INT64  = "INT64"
	FP16   = "FP16"
	FP32   = "FP32"
	FP64   = "FP64"
	BYTES  = "BYTES"
)

const CONTENT_TYPE = "content_type"
const BASE64 = "base64"

type CustomJSONPb struct {
	runtime.JSONPb
}

type tensorType struct {
	size      int
	sliceType reflect.Type
}

func sliceType(v interface{}) reflect.Type {
	return reflect.SliceOf(reflect.TypeOf(v))
}

// Sizes of each type in bytes.
var tensorTypes = map[string]tensorType{
	BOOL:   {1, sliceType(true)},
	UINT8:  {1, sliceType(uint8(0))},
	UINT16: {2, sliceType(uint16(0))},
	UINT32: {4, sliceType(uint32(0))},
	UINT64: {8, sliceType(uint64(0))},
	INT8:   {1, sliceType(int8(0))},
	INT16:  {2, sliceType(int16(0))},
	INT32:  {4, sliceType(int32(0))},
	INT64:  {8, sliceType(int64(0))},
	FP16:   {2, nil}, //TODO
	FP32:   {4, sliceType(float32(0))},
	FP64:   {8, sliceType(float64(0))},
	BYTES:  {1, sliceType(byte(0))},
}

type RESTResponse struct {
	ModelName    string                 `json:"model_name,omitempty"`
	ModelVersion string                 `json:"model_version,omitempty"`
	Id           string                 `json:"id,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	Outputs      []OutputTensor         `json:"outputs,omitempty"`
}

type OutputTensor struct {
	Name       string                 `json:"name,omitempty"`
	Datatype   string                 `json:"datatype,omitempty"`
	Shape      []int64                `json:"shape,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Data       interface{}            `json:"data,omitempty"`
}

// This function adjusts the gRPC response before marshaling and
// returning to the user.
func (c *CustomJSONPb) Marshal(v interface{}) ([]byte, error) {
	if r, ok := v.(*gw.ModelInferResponse); ok {
		var err error
		if v, err = transformResponse(r); err != nil {
			return nil, err
		}
	}
	return c.JSONPb.Marshal(v)
}

func transformResponse(r *gw.ModelInferResponse) (*RESTResponse, error) {
	resp := &RESTResponse{
		ModelName:    r.ModelName,
		ModelVersion: r.ModelVersion,
		Id:           r.Id,
		Parameters:   parameterMapToJson(r.Parameters),
		Outputs:      make([]OutputTensor, len(r.Outputs)),
	}

	for index, output := range r.Outputs {
		tensor := &resp.Outputs[index]
		tensor.Name = output.Name
		tensor.Datatype = output.Datatype
		tensor.Shape = output.Shape
		tensor.Parameters = parameterMapToJson(output.Parameters)
		if tensor.Datatype == FP16 {
			return nil, fmt.Errorf("FP16 tensors not supported (request tensor %s)", tensor.Name) //TODO
		}
		if tensor.Datatype == BYTES {
			tensor.Parameters[CONTENT_TYPE] = BASE64
		}
		if r.RawOutputContents != nil {
			tt, ok := tensorTypes[tensor.Datatype]
			if !ok {
				return nil, fmt.Errorf("unsupported datatype in inference response outputs: %s",
					tensor.Datatype)
			}
			numElements := int(elementCount(tensor.Shape))
			var err error
			if tensor.Datatype == BYTES {
				tensor.Data, err = splitRawBytes(r.RawOutputContents[index], numElements)
			} else {
				tensor.Data, err = readBytes(r.RawOutputContents[index], tt, 0, numElements)
			}
			if err != nil {
				return nil, err
			}
		} else {
			switch tensor.Datatype {
			case BOOL:
				tensor.Data = output.Contents.BoolContents
			case UINT8, UINT16, UINT32:
				tensor.Data = output.Contents.UintContents
			case UINT64:
				tensor.Data = output.Contents.Uint64Contents
			case INT8, INT16, INT32:
				tensor.Data = output.Contents.IntContents
			case INT64:
				tensor.Data = output.Contents.Int64Contents
			case FP32:
				tensor.Data = output.Contents.Fp32Contents
			case FP64:
				tensor.Data = output.Contents.Fp64Contents
			case BYTES:
				// this will be encoded as array of b64-encoded strings
				//TODO support UTF8 if it's specified as the content type
				tensor.Data = output.Contents.BytesContents
			default:
				return nil, fmt.Errorf("unsupported datatype in inference response outputs: %s",
					tensor.Datatype)
			}
		}
	}
	return resp, nil
}

func elementCount(shape []int64) int64 {
	var count int64 = 1
	for j := range shape {
		count *= shape[j]
	}
	return count
}

// This function is used for processing RawOutputContents byte array.
func readBytes(dataBytes []byte, elementType tensorType, index int, numElements int) (interface{}, error) {
	tensorSize := numElements * elementType.size
	start := index * tensorSize
	buf := bytes.NewBuffer(dataBytes[start : start+tensorSize])
	data := reflect.MakeSlice(elementType.sliceType, numElements, numElements).Interface()
	return data, binary.Read(buf, binary.LittleEndian, data)
}

// Output parameters

func parameterMapToJson(pm map[string]*gw.InferParameter) map[string]interface{} {
	jsonMap := make(map[string]interface{}, len(pm))
	for k, ip := range pm {
		var val interface{}
		switch v := ip.GetParameterChoice().(type) {
		case *gw.InferParameter_BoolParam:
			val = v.BoolParam
		case *gw.InferParameter_StringParam:
			val = v.StringParam
		case *gw.InferParameter_Int64Param:
			val = v.Int64Param
		}
		jsonMap[k] = val // may be nil
	}
	return jsonMap
}
