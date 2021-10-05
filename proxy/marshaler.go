package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
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

type tensorType struct {
	size      int
	sliceType reflect.Type
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

// This function adjusts the gRPC response before marshaling and
// returning to the user.
func (c *CustomJSONPb) Marshal(v interface{}) ([]byte, error) {
	if r, ok := v.(*gw.ModelInferResponse); ok {
		resp := &RESTResponse{}
		resp.ModelName = r.ModelName
		resp.ModelVersion = r.ModelVersion
		resp.Id = r.Id
		resp.Parameters = r.Parameters
		resp.Outputs = make([]Tensor, len(r.Outputs))

		for index, output := range r.Outputs {
			tensor := &resp.Outputs[index]
			tensor.Name = output.Name
			tensor.Datatype = output.Datatype
			tensor.Shape = output.Shape
			tensor.Parameters = output.Parameters

			if r.RawOutputContents != nil {
				tt, ok := tensorTypes[tensor.Datatype]
				if !ok {
					return nil, fmt.Errorf("unsupported datatype in inference response outputs: %s",
						tensor.Datatype)
				}
				switch tensor.Datatype {
				case BYTES:
					tensor.Data = r.RawOutputContents[index]
				case FP16:
					return nil, fmt.Errorf("FP16 tensors not supported (request tensor %s)", tensor.Name) //TODO
				default:
					numElements := int(elementCount(tensor.Shape))
					var err error
					if tensor.Data, err = readBytes(r.RawOutputContents[index], tt, 0, numElements); err != nil {
						return nil, err
					}
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
				case FP16:
					return nil, fmt.Errorf("FP16 tensors not supported (request tensor %s)", tensor.Name) //TODO
				case FP32:
					tensor.Data = output.Contents.Fp32Contents
				case FP64:
					tensor.Data = output.Contents.Fp64Contents
				case BYTES:
					tensor.Data = output.Contents.BytesContents
				default:
					return nil, fmt.Errorf("unsupported datatype in inference response outputs: %s",
						tensor.Datatype)
				}
			}
		}
		v = resp
	}

	return c.JSONPb.Marshal(v)
}

// This function adjusts the user input before a gRPC message is sent to the server.
func (c *CustomJSONPb) NewDecoder(r io.Reader) runtime.Decoder {
	return runtime.DecoderFunc(func(v interface{}) error {
		req, ok := v.(*gw.ModelInferRequest)
		if ok {
			logger.Info("Received REST inference request")
			restReq := RESTRequest{}
			if err := json.NewDecoder(r).Decode(&restReq); err != nil {
				return err
			}

			req.Id = restReq.Id
			req.Parameters = restReq.Parameters
			req.Outputs = restReq.Outputs
			req.Inputs = make([]*gw.ModelInferRequest_InferInputTensor, 0, len(restReq.Inputs))

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
				case BOOL:
					data := make([]bool, len(d))
					for i := range d {
						data[i] = d[i].(bool)
					}
					tensor.Contents = &gw.InferTensorContents{BoolContents: data}
				case UINT8, UINT16, UINT32:
					data := make([]uint32, len(d))
					for i := range d {
						data[i] = uint32(d[i].(float64))
					}
					tensor.Contents = &gw.InferTensorContents{UintContents: data}
				case UINT64:
					data := make([]uint64, len(d))
					for i := range d {
						data[i] = uint64(d[i].(float64))
					}
					tensor.Contents = &gw.InferTensorContents{Uint64Contents: data}
				case INT8, INT16, INT32:
					data := make([]int32, len(d))
					for i := range d {
						data[i] = int32(d[i].(float64))
					}
					tensor.Contents = &gw.InferTensorContents{IntContents: data}
				case INT64:
					data := make([]int64, len(d))
					for i := range d {
						data[i] = int64(d[i].(float64))
					}
					tensor.Contents = &gw.InferTensorContents{Int64Contents: data}
				case FP16:
					return fmt.Errorf("FP16 tensors not supported (response tensor %s)", tensor.Name) //TODO
				case FP32:
					data := make([]float32, len(d))
					for i := range d {
						data[i] = float32(d[i].(float64))
					}
					tensor.Contents = &gw.InferTensorContents{Fp32Contents: data}
				case FP64:
					data := make([]float64, len(d))
					for i := range d {
						data[i] = d[i].(float64)
					}
					tensor.Contents = &gw.InferTensorContents{Fp64Contents: data}
				case BYTES:
					// TODO: BytesContents is multi-dimensional. Figure out how to
					// correctly represent the data from a 2D slice.
					data := make([][]byte, 1)
					data[0] = make([]byte, len(d))
					for i := range d {
						data[index][i] = byte(d[i].(float64))
					}
					tensor.Contents = &gw.InferTensorContents{BytesContents: data}
				default:
					return fmt.Errorf("unsupported datatype: %s", tensor.Datatype)
				}
				req.Inputs = append(req.Inputs, tensor)
			}
			return nil
		}
		return c.JSONPb.NewDecoder(r).Decode(v)
	})
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
	return &data, binary.Read(buf, binary.LittleEndian, &data)
}

func sliceType(v interface{}) reflect.Type {
	return reflect.SliceOf(reflect.TypeOf(v))
}
