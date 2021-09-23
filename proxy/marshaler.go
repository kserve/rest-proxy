package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	gw "github.com/kserve/rest-proxy/gen"
)

type Tensor struct {
	Name       string                        `json:"name,omitempty"`
	Datatype   string                        `json:"datatype,omitempty"`
	Shape      []int64                       `json:"shape,omitempty"`
	Parameters map[string]*gw.InferParameter `json:"parameters,omitempty"`
	Data       interface{}                   `json:"data,omitempty"`
}

type RESTResponse struct {
	ModelName    string                        `json:"model_name,omitempty"`
	ModelVersion string                        `json:"model_version,omitempty"`
	Id           string                        `json:"id,omitempty"`
	Parameters   map[string]*gw.InferParameter `json:"parameters,omitempty"`
	Outputs      []Tensor                      `json:"outputs,omitempty"`
}

type RESTRequest struct {
	Id         string                                             `json:"id,omitempty"`
	Parameters map[string]*gw.InferParameter                      `json:"parameters,omitempty"`
	Inputs     []Tensor                                           `json:"inputs,omitempty"`
	Outputs    []*gw.ModelInferRequest_InferRequestedOutputTensor `json:"outputs,omitempty"`
}

type CustomJSONPb struct {
	runtime.JSONPb
}

// Sizes of each type in bytes.
var typeSizes = map[string]int64{
	"BOOL":   1,
	"UINT8":  1,
	"UINT16": 2,
	"UINT32": 4,
	"UINT64": 8,
	"INT8":   1,
	"INT16":  2,
	"INT32":  4,
	"INT64":  8,
	"FP16":   2,
	"FP32":   4,
	"FP64":   8,
	"BYTES":  1,
}

// This function adjusts the gRPC response before marshaling and
// returning to the user.
func (c *CustomJSONPb) Marshal(v interface{}) ([]byte, error) {
	var err error
	var j []byte
	switch v := v.(type) {
	case *gw.ModelInferResponse:
		resp := &RESTResponse{}
		resp.ModelName = v.ModelName
		resp.ModelVersion = v.ModelVersion
		resp.Id = v.Id
		resp.Parameters = v.Parameters

		for index, output := range v.Outputs {
			tensor := Tensor{}
			tensor.Name = output.Name
			tensor.Datatype = output.Datatype
			tensor.Shape = output.Shape
			tensor.Parameters = output.Parameters

			var numElements int64 = 1
			for j := range tensor.Shape {
				numElements = tensor.Shape[j] * numElements
			}

			var data interface{}
			switch tensor.Datatype {
			case "BOOL":
				if v.RawOutputContents == nil {
					data = output.Contents.BoolContents
				} else {
					outputData := make([]bool, numElements)
					err = readBytes(v.RawOutputContents[index], &outputData, 0, typeSizes[tensor.Datatype], numElements)
					data = outputData
				}
			case "UINT8":
				if v.RawOutputContents == nil {
					data = output.Contents.UintContents
				} else {
					outputData := make([]uint8, numElements)
					err = readBytes(v.RawOutputContents[index], &outputData, 0, typeSizes[tensor.Datatype], numElements)
					data = outputData
				}
			case "UINT16":
				if v.RawOutputContents == nil {
					data = output.Contents.UintContents
				} else {
					outputData := make([]uint16, numElements)
					err = readBytes(v.RawOutputContents[index], &outputData, 0, typeSizes[tensor.Datatype], numElements)
					data = outputData
				}
			case "UINT32":
				if v.RawOutputContents == nil {
					data = output.Contents.UintContents
				} else {
					outputData := make([]uint32, numElements)
					err = readBytes(v.RawOutputContents[index], &outputData, 0, typeSizes[tensor.Datatype], numElements)
					data = outputData
				}
			case "UINT64":
				if v.RawOutputContents == nil {
					data = output.Contents.Uint64Contents
				} else {
					outputData := make([]uint64, numElements)
					err = readBytes(v.RawOutputContents[index], &outputData, 0, typeSizes[tensor.Datatype], numElements)
					data = outputData
				}
			case "INT8":
				if v.RawOutputContents == nil {
					data = output.Contents.IntContents
				} else {
					outputData := make([]int8, numElements)
					err = readBytes(v.RawOutputContents[index], &outputData, 0, typeSizes[tensor.Datatype], numElements)
					data = outputData
				}
			case "INT16":
				if v.RawOutputContents == nil {
					data = output.Contents.IntContents
				} else {
					outputData := make([]int16, numElements)
					err = readBytes(v.RawOutputContents[index], &outputData, 0, typeSizes[tensor.Datatype], numElements)
					data = outputData
				}
			case "INT32":
				if v.RawOutputContents == nil {
					data = output.Contents.IntContents
				} else {
					outputData := make([]int32, numElements)
					err = readBytes(v.RawOutputContents[index], &outputData, 0, typeSizes[tensor.Datatype], numElements)
					data = outputData
				}
			case "INT64":
				if v.RawOutputContents == nil {
					data = output.Contents.Int64Contents
				} else {
					outputData := make([]int64, numElements)
					err = readBytes(v.RawOutputContents[index], &outputData, 0, typeSizes[tensor.Datatype], numElements)
					data = outputData
				}
			case "FP16":
				// TODO: Relies on raw_input_contents
			case "FP32":
				if v.RawOutputContents == nil {
					data = output.Contents.Fp32Contents
				} else {
					outputData := make([]float32, numElements)
					err = readBytes(v.RawOutputContents[index], &outputData, 0, typeSizes[tensor.Datatype], numElements)
					data = outputData
				}
			case "FP64":
				if v.RawOutputContents == nil {
					data = output.Contents.Fp64Contents
				} else {
					outputData := make([]float64, numElements)
					err = readBytes(v.RawOutputContents[index], &outputData, 0, typeSizes[tensor.Datatype], numElements)
					data = outputData
				}
			case "BYTES":
				if v.RawOutputContents == nil {
					data = output.Contents.BytesContents
				} else {
					data = v.RawOutputContents[index]
				}
			default:
				return nil, fmt.Errorf("Unsupported Datatype in inference response outputs")
			}

			if err != nil {
				return nil, err
			}

			tensor.Data = data
			resp.Outputs = append(resp.Outputs, tensor)

			j, err = c.JSONPb.Marshal(resp)
		}
	default:
		j, err = c.JSONPb.Marshal(v)
	}

	if err != nil {
		return nil, err
	}
	return j, nil
}

// This function adjusts the user input before a gRPC message is sent to the
// server.
func (c *CustomJSONPb) NewDecoder(r io.Reader) runtime.Decoder {
	return runtime.DecoderFunc(func(v interface{}) error {
		req, ok := v.(*gw.ModelInferRequest)
		if ok {
			logger.Info("Received REST inference request")
			raw, err := ioutil.ReadAll(r)
			if err != nil {
				return err
			}
			restReq := RESTRequest{}
			err = json.Unmarshal(raw, &restReq)
			if err != nil {
				return err
			}

			req.Id = restReq.Id
			req.Parameters = restReq.Parameters
			req.Outputs = restReq.Outputs

			// TODO: Figure out better/cleaner way to do type coercion?
			// TODO: Flatten N-Dimensional data arrays.

			for index, input := range restReq.Inputs {
				tensor := &gw.ModelInferRequest_InferInputTensor{
					Name:       input.Name,
					Datatype:   input.Datatype,
					Shape:      input.Shape,
					Parameters: input.Parameters,
				}
				d := input.Data.([]interface{})
				switch tensor.Datatype {
				case "BOOL":
					data := make([]bool, len(d))
					for i := range d {
						data[i] = d[i].(bool)
					}
					tensor.Contents = &gw.InferTensorContents{BoolContents: data}
				case "UINT8", "UINT16", "UINT32":
					data := make([]uint32, len(d))
					for i := range d {
						data[i] = uint32(d[i].(float64))
					}
					tensor.Contents = &gw.InferTensorContents{UintContents: data}
				case "UINT64":
					data := make([]uint64, len(d))
					for i := range d {
						data[i] = uint64(d[i].(float64))
					}
					tensor.Contents = &gw.InferTensorContents{Uint64Contents: data}
				case "INT8", "INT16", "INT32":
					data := make([]int32, len(d))
					for i := range d {
						data[i] = int32(d[i].(float64))
					}
					tensor.Contents = &gw.InferTensorContents{IntContents: data}
				case "INT64":
					data := make([]int64, len(d))
					for i := range d {
						data[i] = int64(d[i].(float64))
					}
					tensor.Contents = &gw.InferTensorContents{Int64Contents: data}
				case "FP16":
					// TODO: Relies on raw_input_contents
				case "FP32":
					data := make([]float32, len(d))
					for i := range d {
						data[i] = float32(d[i].(float64))
					}
					tensor.Contents = &gw.InferTensorContents{Fp32Contents: data}
				case "FP64":
					data := make([]float64, len(d))
					for i := range d {
						data[i] = d[i].(float64)
					}
					tensor.Contents = &gw.InferTensorContents{Fp64Contents: data}
				case "BYTES":
					// TODO: BytesContents is multi-dimensional. Figure out how to
					// correctly represent the data from a 2D slice.
					data := make([][]byte, 1)
					data[0] = make([]byte, len(d))
					for i := range d {
						data[index][i] = byte(d[i].(float64))
					}
					tensor.Contents = &gw.InferTensorContents{BytesContents: data}
				default:
					return fmt.Errorf("Unsupported Datatype")
				}
				req.Inputs = append(req.Inputs, tensor)
			}
			return nil
		}
		return c.JSONPb.NewDecoder(r).Decode(v)
	})
}

// This function is used for processing RawOutputContents byte array.
func readBytes(dataBytes []byte, data interface{}, index int, size int64, numElements int64) error {
	start := int64(index) * numElements * size
	buf := bytes.NewBuffer(dataBytes[start : start+numElements*size])
	return binary.Read(buf, binary.LittleEndian, data)
}
