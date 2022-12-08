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
	"testing"

	"github.com/google/go-cmp/cmp"
	gw "github.com/kserve/rest-proxy/gen"
)

func generateProtoBufResponse() *gw.ModelInferResponse {
	expectedOutput := []*gw.ModelInferResponse_InferOutputTensor{{
		Name:     "predict",
		Datatype: "INT64",
		Shape:    []int64{2},
		Contents: &gw.InferTensorContents{
			Int64Contents: []int64{8, 8},
		},
	}}

	return &gw.ModelInferResponse{
		ModelName: "example",
		Id:        "foo",
		Outputs:   expectedOutput,
		Parameters: map[string]*gw.InferParameter{
			"content_type": {ParameterChoice: &gw.InferParameter_StringParam{StringParam: "bar"}},
			"headers":      {ParameterChoice: nil},
			"int_param":    {ParameterChoice: &gw.InferParameter_Int64Param{Int64Param: 12345}},
			"bool_param":   {ParameterChoice: &gw.InferParameter_BoolParam{BoolParam: false}},
		},
	}
}

var jsonResponse = `{"model_name":"example","id":"foo","parameters":{"bool_param":false,"content_type":"bar","headers":null,"int_param":12345},` +
	`"outputs":[{"name":"predict","datatype":"INT64","shape":[2],"data":[8,8]}]}`

func generateProtoBufBytesResponse() *gw.ModelInferResponse {
	expectedOutput := []*gw.ModelInferResponse_InferOutputTensor{{
		Name:     "predict",
		Datatype: "BYTES",
		Shape:    []int64{2, 2},
		Contents: &gw.InferTensorContents{
			BytesContents: [][]byte{[]byte("String1"), []byte("String2"), []byte("String3"), []byte("String4")},
		},
	}}

	return &gw.ModelInferResponse{
		ModelName: "example",
		Id:        "foo",
		Outputs:   expectedOutput,
		Parameters: map[string]*gw.InferParameter{
			"content_type": {ParameterChoice: &gw.InferParameter_StringParam{StringParam: "bar"}},
			"headers":      {ParameterChoice: nil},
			"int_param":    {ParameterChoice: &gw.InferParameter_Int64Param{Int64Param: 12345}},
			"bool_param":   {ParameterChoice: &gw.InferParameter_BoolParam{BoolParam: false}},
		},
	}
}

func generateProtoBufBytesResponseRawOutput() *gw.ModelInferResponse {
	expectedOutput := []*gw.ModelInferResponse_InferOutputTensor{{
		Name:     "predict",
		Datatype: "BYTES",
		Shape:    []int64{2, 2},
	}}

	seven := make([]byte, 4)
	binary.LittleEndian.PutUint32(seven, 7)
	rawBytes := append(seven, "String1"...)
	rawBytes = append(rawBytes, seven...)
	rawBytes = append(rawBytes, "String2"...)
	rawBytes = append(rawBytes, seven...)
	rawBytes = append(rawBytes, "String3"...)
	rawBytes = append(rawBytes, seven...)
	rawBytes = append(rawBytes, "String4"...)

	return &gw.ModelInferResponse{
		ModelName: "example",
		Id:        "foo",
		Outputs:   expectedOutput,
		Parameters: map[string]*gw.InferParameter{
			"content_type": {ParameterChoice: &gw.InferParameter_StringParam{StringParam: "bar"}},
			"headers":      {ParameterChoice: nil},
			"int_param":    {ParameterChoice: &gw.InferParameter_Int64Param{Int64Param: 12345}},
			"bool_param":   {ParameterChoice: &gw.InferParameter_BoolParam{BoolParam: false}},
		},
		RawOutputContents: [][]byte{rawBytes},
	}
}

var jsonBytesResponse = `{"model_name":"example","id":"foo","parameters":{"bool_param":false,"content_type":"bar","headers":null,"int_param":12345},` +
	`"outputs":[{"name":"predict","datatype":"BYTES","shape":[2,2],"parameters":{"content_type":"base64"},"data":["U3RyaW5nMQ==","U3RyaW5nMg==","U3RyaW5nMw==","U3RyaW5nNA=="]}]}`

func TestRESTResponse(t *testing.T) {
	c := CustomJSONPb{}
	v := generateProtoBufResponse()
	marshal, err := c.Marshal(v)
	if err != nil {
		t.Error(err)
	}
	if d := cmp.Diff(string(marshal), jsonResponse); d != "" {
		t.Errorf("diff :%s", d)
	}
}

func TestBytesRESTResponse(t *testing.T) {
	c := CustomJSONPb{}
	v := generateProtoBufBytesResponse()
	marshal, err := c.Marshal(v)
	if err != nil {
		t.Error(err)
	}
	if d := cmp.Diff(string(marshal), jsonBytesResponse); d != "" {
		t.Errorf("diff :%s", d)
	}
}

func TestBytesRESTResponseRawOutput(t *testing.T) {
	c := CustomJSONPb{}
	v := generateProtoBufBytesResponseRawOutput()
	marshal, err := c.Marshal(v)
	if err != nil {
		t.Error(err)
	}
	if d := cmp.Diff(string(marshal), jsonBytesResponse); d != "" {
		t.Errorf("diff :%s", d)
	}
}

func TestRESTResponseRawOutput(t *testing.T) {
	c := CustomJSONPb{}
	buf := new(bytes.Buffer)
	var val int64 = 7
	if err := binary.Write(buf, binary.LittleEndian, val); err != nil {
		t.Error(err)
	}
	v := &gw.ModelInferResponse{
		ModelName: "example",
		Id:        "foo",
		Outputs: []*gw.ModelInferResponse_InferOutputTensor{{
			Name:     "predict",
			Datatype: "INT64",
			Shape:    []int64{1, 1},
		}},
		RawOutputContents: [][]byte{
			buf.Bytes(),
		},
	}

	output, err := c.Marshal(v)
	if err != nil {
		t.Error(err)
	}

	expected := `{"model_name":"example","id":"foo","outputs":[{"name":"predict","datatype":"INT64","shape":[1,1],"data":[7]}]}`
	if d := cmp.Diff(expected, string(output)); d != "" {
		t.Errorf("diff :%s", d)
	}
}
