package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	gw "github.com/kserve/rest-proxy/gen"
	"google.golang.org/protobuf/proto"
)

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

type bytesTensorTestCase struct {
	shape      []int64
	jsonData   string
	pbBytes    [][]byte
	parameters map[string]string
}

var bytesTensorCases = []bytesTensorTestCase{
	{
		shape:    []int64{2},
		jsonData: `["My UTF8 String", "Another string"]`,
		pbBytes:  [][]byte{[]byte("My UTF8 String"), []byte("Another string")},
	},
	{
		shape:    []int64{1},
		jsonData: `[[77, 121, 32, 85, 84, 70, 56, 32, 83, 116, 114, 105, 110, 103]]`,
		pbBytes:  [][]byte{{77, 121, 32, 85, 84, 70, 56, 32, 83, 116, 114, 105, 110, 103}},
	},
	{
		shape:    []int64{2, 1},
		jsonData: `[["String1"], ["String2"]]`,
		pbBytes:  [][]byte{[]byte("String1"), []byte("String2")},
	},
	{
		shape:      []int64{2, 1},
		jsonData:   `["String1", "String2"]`,
		pbBytes:    [][]byte{[]byte("String1"), []byte("String2")},
		parameters: map[string]string{"content_type": "str"},
	},
	{
		shape:    []int64{2, 1},
		jsonData: `[[[83, 116, 114, 105, 110, 103, 32, 49]], [[83, 116, 114, 105, 110, 103, 32, 50]]]`,
		pbBytes:  [][]byte{{83, 116, 114, 105, 110, 103, 32, 49}, {83, 116, 114, 105, 110, 103, 32, 50}},
	},
	{
		shape:      []int64{2, 1},
		jsonData:   `["TXkgVVRGOCBTdHJpbmc=", "QW5vdGhlciBzdHJpbmc="]`,
		pbBytes:    [][]byte{[]byte("My UTF8 String"), []byte("Another string")},
		parameters: map[string]string{"content_type": "base64"},
	},
}

func bytesRestRequest(shape []int64, jsonData string, parameters map[string]string) string {
	shapeStr, _ := json.Marshal(shape)
	parameterStr := ""
	if len(parameters) != 0 {
		p, _ := json.Marshal(parameters)
		parameterStr = `, "parameters": ` + string(p)
	}

	return `{
	"id": "foo",
    "parameters": {
		"top_level": "foo",
		"bool_param": false
    },
	"inputs": [{
		"name": "predict",
		"shape": ` + string(shapeStr) + `,		
		"datatype": "BYTES",
		"data": ` + jsonData +
		parameterStr + `
	}]
	}`
}

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
		if err := c.NewDecoder(buffer).Decode(out); err != nil {
			t.Error(err)
		}
		expected := generateProtoBufRequest(inputDataShapes[k])
		if !proto.Equal(out, expected) {
			t.Errorf("REST request failed to decode for shape: %v", inputDataShapes[k])
		}
	}
}

func TestBytesRESTRequest(t *testing.T) {
	for _, test := range bytesTensorCases {
		c := CustomJSONPb{}
		buffer := &bytes.Buffer{}
		out := &gw.ModelInferRequest{}
		buffer.Write([]byte(bytesRestRequest(test.shape, test.jsonData, test.parameters)))
		if err := c.NewDecoder(buffer).Decode(out); err != nil {
			t.Error(err)
		}

		expected := &gw.ModelInferRequest{
			Id: "foo",
			Parameters: map[string]*gw.InferParameter{
				"top_level":  {ParameterChoice: &gw.InferParameter_StringParam{StringParam: "foo"}},
				"bool_param": {ParameterChoice: &gw.InferParameter_BoolParam{BoolParam: false}},
			},
			Inputs: []*gw.ModelInferRequest_InferInputTensor{{
				Name:     "predict",
				Datatype: "BYTES",
				Shape:    test.shape,
				Contents: &gw.InferTensorContents{
					BytesContents: test.pbBytes,
				}},
			},
			RawInputContents: nil,
		}

		if len(test.parameters) > 0 {
			p := map[string]*gw.InferParameter{}
			for k, v := range test.parameters {
				p[k] = &gw.InferParameter{
					ParameterChoice: &gw.InferParameter_StringParam{StringParam: v},
				}
			}
			expected.Inputs[0].Parameters = p
		}

		fmt.Println(out)
		if !proto.Equal(out, expected) {
			t.Errorf("REST request failed to decode for test: %v: %v != %v", test, out, expected)
		}
	}

}
