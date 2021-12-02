package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
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

type OutputTensor struct {
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
	Outputs      []OutputTensor                `json:"outputs,omitempty"`
}

type RESTRequest struct {
	Id         string                                             `json:"id,omitempty"`
	Parameters map[string]*gw.InferParameter                      `json:"parameters,omitempty"`
	Inputs     []InputTensor                                      `json:"inputs,omitempty"`
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
		fmt.Printf("%v, %v\n", v, r)
		resp := &RESTResponse{}
		resp.ModelName = r.ModelName
		resp.ModelVersion = r.ModelVersion
		resp.Id = r.Id
		resp.Parameters = r.Parameters
		resp.Outputs = make([]OutputTensor, len(r.Outputs))

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

type InputTensor gw.ModelInferRequest_InferInputTensor

type InputTensorMeta struct {
	Name     string  `json:"name"`
	Datatype string  `json:"datatype"`
	Shape    []int64 `json:"shape"`
}

type InputTensorData struct {
	Data       tensorDataUnmarshaller        `json:"data"`
	Parameters map[string]*gw.InferParameter `json:"parameters"`
}

func (t *InputTensor) UnmarshalJSON(data []byte) error {
	meta := InputTensorMeta{}
	if err := json.Unmarshal(data, &meta); err != nil {
		return err
	}
	contents := &gw.InferTensorContents{}
	target, err := targetArray(meta.Datatype, meta.Name, contents)
	if err != nil {
		return err
	}
	itd := &InputTensorData{Data: tensorDataUnmarshaller{target: target, shape: meta.Shape}}
	if err := json.Unmarshal(data, itd); err != nil {
		return err
	}
	*t = InputTensor{
		Name:       meta.Name,
		Datatype:   meta.Datatype,
		Shape:      meta.Shape,
		Parameters: itd.Parameters,
		Contents:   contents,
	}

	return nil
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
			req.Inputs = make([]*gw.ModelInferRequest_InferInputTensor, len(restReq.Inputs))
			for i := range restReq.Inputs {
				req.Inputs[i] = (*gw.ModelInferRequest_InferInputTensor)(&restReq.Inputs[i])
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

type tensorDataUnmarshaller struct {
	target interface{}
	shape  []int64
}

func (t *tensorDataUnmarshaller) UnmarshalJSON(data []byte) error {
	if len(t.shape) <= 1 {
		return json.Unmarshal(data, t.target) // single-dimension fast-path
	}
	start := -1
	for i, b := range data {
		if b == '[' {
			if start != -1 {
				data = data[start:]
				break
			}
			start = i
		} else if !isSpace(b) {
			if start == -1 {
				return errors.New("invalid tensor data: not a json array")
			}
			// fast-path: flat array
			return json.Unmarshal(data, t.target)
		}
	}
	// here we have nested arrays

	//TODO handle strings / BYTES case

	// strip all the square brackets (update data slice in-place)
	var o, c int
	j := 1
	for _, b := range data {
		if b == '[' {
			o++
		} else if b == ']' {
			c++
		} else {
			data[j] = b
			j++
		}
	}
	if o != c || o != expectedBracketCount(t.shape) {
		return errors.New("invalid tensor data: invalid nested json arrays")
	}
	data[j] = ']'
	return json.Unmarshal(data[:j+1], t.target)
}

func expectedBracketCount(shape []int64) int {
	n := len(shape) - 1
	if n < 1 {
		return 1
	}
	p, s := 1, 1
	for i := 0; i < n; i++ {
		p *= int(shape[i])
		s += p
	}
	return s
}

func targetArray(dataType, tensorName string, contents *gw.InferTensorContents) (interface{}, error) {
	switch dataType {
	case BOOL:
		return &contents.BoolContents, nil
	case UINT8, UINT16, UINT32:
		return &contents.UintContents, nil
	case UINT64:
		return &contents.Uint64Contents, nil
	case INT8, INT16, INT32:
		return &contents.IntContents, nil
	case INT64:
		return &contents.Int64Contents, nil
	case FP16:
		return nil, fmt.Errorf("FP16 tensors not supported (response tensor %s)", tensorName) //TODO
	case FP32:
		return &contents.Fp32Contents, nil
	case FP64:
		return &contents.Fp64Contents, nil
	case BYTES:
		return &contents.BytesContents, nil //TODO still need to figure this one out
	default:
		return nil, fmt.Errorf("unsupported datatype: %s", dataType)
	}
}

func isSpace(c byte) bool {
	return c <= ' ' && (c == ' ' || c == '\t' || c == '\r' || c == '\n')
}
