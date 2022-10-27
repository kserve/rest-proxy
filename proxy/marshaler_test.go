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
