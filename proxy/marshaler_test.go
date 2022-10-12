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
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/google/go-cmp/cmp"
	gw "github.com/kserve/rest-proxy/gen"
)

func restRequest(data string, shape string) string {
	return `{
	"id": "foo",
    "parameters": {
		"top_level": "foo",
		"bool_param": false
    },
	"inputs": [{
		"name": "predict",
		"shape": ` + shape + `,
		"datatype": "FP32",
		"data":` + data + `,
        "parameters": {
            "content_type": "str",
			"headers": null,
			"int_param": 42,
			"bool_param": true
        }
		}]
	}`
}

var data1D = `
	[0.0, 0.0, 1.0, 11.0, 14.0, 15.0, 3.0, 0.0, 0.0, 1.0, 13.0, 16.0, 12.0, 16.0, 8.0, 0.0,
	0.0, 8.0, 16.0, 4.0, 6.0, 16.0, 5.0, 0.0, 0.0, 5.0, 15.0, 11.0, 13.0, 14.0, 0.0, 0.0, 0.0, 0.0,
	2.0, 12.0, 16.0, 13.0, 0.0, 0.0, 0.0, 0.0, 0.0, 13.0, 16.0, 16.0, 6.0, 0.0, 0.0, 0.0, 0.0, 16.0,
	16.0, 16.0, 7.0, 0.0, 0.0, 0.0, 0.0, 11.0, 13.0, 12.0, 1.0, 0.0, 0.0, 0.0, 1.0, 11.0, 14.0, 15.0,
	3.0, 0.0, 0.0, 1.0, 13.0, 16.0, 12.0, 16.0, 8.0, 0.0, 0.0, 8.0, 16.0, 4.0, 6.0, 16.0, 5.0, 0.0,
	0.0, 5.0, 15.0, 11.0, 13.0, 14.0, 0.0, 0.0, 0.0, 0.0, 2.0, 12.0, 16.0, 13.0, 0.0, 0.0, 0.0, 0.0,
	0.0, 13.0, 16.0, 16.0, 6.0, 0.0, 0.0, 0.0, 0.0, 16.0, 16.0, 16.0, 7.0, 0.0, 0.0, 0.0, 0.0, 11.0,
	13.0, 12.0, 1.0, 0.0]
`

var data2D = `
[
	[0.0, 0.0, 1.0, 11.0, 14.0, 15.0, 3.0, 0.0, 0.0, 1.0, 13.0, 16.0, 12.0, 16.0, 8.0,
	0.0, 0.0, 8.0, 16.0, 4.0, 6.0, 16.0, 5.0, 0.0, 0.0, 5.0, 15.0, 11.0, 13.0, 14.0, 0.0,
	0.0, 0.0, 0.0, 2.0, 12.0, 16.0, 13.0, 0.0, 0.0, 0.0, 0.0, 0.0, 13.0, 16.0, 16.0, 6.0,
	0.0, 0.0, 0.0, 0.0, 16.0, 16.0, 16.0, 7.0, 0.0, 0.0, 0.0, 0.0, 11.0, 13.0, 12.0, 1.0, 0.0],

	[0.0, 0.0, 1.0, 11.0, 14.0, 15.0, 3.0, 0.0, 0.0, 1.0, 13.0, 16.0, 12.0, 16.0, 8.0,
	0.0, 0.0, 8.0, 16.0, 4.0, 6.0, 16.0, 5.0, 0.0, 0.0, 5.0, 15.0, 11.0, 13.0, 14.0, 0.0,
	0.0, 0.0, 0.0, 2.0, 12.0, 16.0, 13.0, 0.0, 0.0, 0.0, 0.0, 0.0, 13.0, 16.0, 16.0, 6.0,
	0.0, 0.0, 0.0, 0.0, 16.0, 16.0, 16.0, 7.0, 0.0, 0.0, 0.0, 0.0, 11.0, 13.0, 12.0, 1.0, 0.0]
]
`

var data3D = `
[
	[
		[0.0, 0.0, 1.0, 11.0, 14.0, 15.0, 3.0, 0.0, 0.0, 1.0, 13.0, 16.0, 12.0, 16.0, 8.0, 0.0,
			0.0, 8.0, 16.0, 4.0, 6.0, 16.0, 5.0, 0.0, 0.0, 5.0, 15.0, 11.0, 13.0, 14.0, 0.0, 0.0
		],
		[0.0, 0.0, 2.0, 12.0, 16.0, 13.0, 0.0, 0.0, 0.0, 0.0, 0.0, 13.0, 16.0, 16.0, 6.0, 0.0,
			0.0, 0.0, 0.0, 16.0, 16.0, 16.0, 7.0, 0.0, 0.0, 0.0, 0.0, 11.0, 13.0, 12.0, 1.0, 0.0
		]
	],
	[
		[0.0, 0.0, 1.0, 11.0, 14.0, 15.0, 3.0, 0.0, 0.0, 1.0, 13.0, 16.0, 12.0, 16.0, 8.0, 0.0,
			0.0, 8.0, 16.0, 4.0, 6.0, 16.0, 5.0, 0.0, 0.0, 5.0, 15.0, 11.0, 13.0, 14.0, 0.0, 0.0
		],
		[0.0, 0.0, 2.0, 12.0, 16.0, 13.0, 0.0, 0.0, 0.0, 0.0, 0.0, 13.0, 16.0, 16.0, 6.0, 0.0,
			0.0, 0.0, 0.0, 16.0, 16.0, 16.0, 7.0, 0.0, 0.0, 0.0, 0.0, 11.0, 13.0, 12.0, 1.0, 0.0
		]
	]
]
`

var data4D = `
[
	[
		[
			[0.0, 0.0, 1.0, 11.0, 14.0, 15.0, 3.0, 0.0, 0.0, 1.0, 13.0, 16.0, 12.0, 16.0, 8.0, 0.0],
			[0.0, 8.0, 16.0, 4.0, 6.0, 16.0, 5.0, 0.0, 0.0, 5.0, 15.0, 11.0, 13.0, 14.0, 0.0, 0.0]
		],
		[
			[0.0, 0.0, 2.0, 12.0, 16.0, 13.0, 0.0, 0.0, 0.0, 0.0, 0.0, 13.0, 16.0, 16.0, 6.0, 0.0],
			[0.0, 0.0, 0.0, 16.0, 16.0, 16.0, 7.0, 0.0, 0.0, 0.0, 0.0, 11.0, 13.0, 12.0, 1.0, 0.0]
		]
	],
	[
		[
			[0.0, 0.0, 1.0, 11.0, 14.0, 15.0, 3.0, 0.0, 0.0, 1.0, 13.0, 16.0, 12.0, 16.0, 8.0, 0.0],
			[0.0, 8.0, 16.0, 4.0, 6.0, 16.0, 5.0, 0.0, 0.0, 5.0, 15.0, 11.0, 13.0, 14.0, 0.0, 0.0]
		],
		[
			[0.0, 0.0, 2.0, 12.0, 16.0, 13.0, 0.0, 0.0, 0.0, 0.0, 0.0, 13.0, 16.0, 16.0, 6.0, 0.0],
			[0.0, 0.0, 0.0, 16.0, 16.0, 16.0, 7.0, 0.0, 0.0, 0.0, 0.0, 11.0, 13.0, 12.0, 1.0, 0.0]
		]
	]
]
`

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

func generateProtoBufRequest(shape []int64) *gw.ModelInferRequest {
	var expectedInput = gw.ModelInferRequest_InferInputTensor{
		Name:     "predict",
		Datatype: "FP32",
		Shape:    shape,
		Parameters: map[string]*gw.InferParameter{
			"content_type": {ParameterChoice: &gw.InferParameter_StringParam{StringParam: "str"}},
			"headers":      {ParameterChoice: nil},
			"int_param":    {ParameterChoice: &gw.InferParameter_Int64Param{Int64Param: 42}},
			"bool_param":   {ParameterChoice: &gw.InferParameter_BoolParam{BoolParam: true}},
		},
		Contents: &gw.InferTensorContents{
			Fp32Contents: []float32{0.0, 0.0, 1.0, 11.0, 14.0, 15.0, 3.0, 0.0, 0.0, 1.0, 13.0, 16.0, 12.0,
				16.0, 8.0, 0.0, 0.0, 8.0, 16.0, 4.0, 6.0, 16.0, 5.0, 0.0, 0.0, 5.0, 15.0, 11.0, 13.0,
				14.0, 0.0, 0.0, 0.0, 0.0, 2.0, 12.0, 16.0, 13.0, 0.0, 0.0, 0.0, 0.0, 0.0, 13.0, 16.0,
				16.0, 6.0, 0.0, 0.0, 0.0, 0.0, 16.0, 16.0, 16.0, 7.0, 0.0, 0.0, 0.0, 0.0, 11.0, 13.0,
				12.0, 1.0, 0.0, 0.0, 0.0, 1.0, 11.0, 14.0, 15.0, 3.0, 0.0, 0.0, 1.0, 13.0, 16.0, 12.0,
				16.0, 8.0, 0.0, 0.0, 8.0, 16.0, 4.0, 6.0, 16.0, 5.0, 0.0, 0.0, 5.0, 15.0, 11.0, 13.0,
				14.0, 0.0, 0.0, 0.0, 0.0, 2.0, 12.0, 16.0, 13.0, 0.0, 0.0, 0.0, 0.0, 0.0, 13.0, 16.0,
				16.0, 6.0, 0.0, 0.0, 0.0, 0.0, 16.0, 16.0, 16.0, 7.0, 0.0, 0.0, 0.0, 0.0, 11.0, 13.0,
				12.0, 1.0, 0.0},
		},
	}

	var modelInferRequest = &gw.ModelInferRequest{
		Id: "foo",
		Parameters: map[string]*gw.InferParameter{
			"top_level":  {ParameterChoice: &gw.InferParameter_StringParam{StringParam: "foo"}},
			"bool_param": {ParameterChoice: &gw.InferParameter_BoolParam{BoolParam: false}},
		},
		Inputs: []*gw.ModelInferRequest_InferInputTensor{&expectedInput},
	}
	return modelInferRequest
}

func TestRESTRequest(t *testing.T) {
	c := CustomJSONPb{}
	inputDataArray := []string{data1D, data2D, data3D, data4D}
	inputDataShapes := [][]int64{{2, 64}, {2, 64}, {2, 2, 32}, {2, 2, 2, 16}}
	for k, data := range inputDataArray {
		out := &gw.ModelInferRequest{}
		buffer := &bytes.Buffer{}
		buffer.Write([]byte(restRequest(data, strings.Join(strings.Split(fmt.Sprintln(inputDataShapes[k]), " "), ","))))
		err := c.NewDecoder(buffer).Decode(out)
		if err != nil {
			t.Error(err)
		}
		expected := generateProtoBufRequest(inputDataShapes[k])
		if !proto.Equal(out, expected) {
			t.Errorf("REST request failed to decode for shape: %v", inputDataShapes[k])
		}
	}
}

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
