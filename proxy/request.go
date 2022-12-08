package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	gw "github.com/kserve/rest-proxy/gen"
)

// This function adjusts the user input before a gRPC message is sent to the server.
func (c *CustomJSONPb) NewDecoder(r io.Reader) runtime.Decoder {
	return runtime.DecoderFunc(func(v interface{}) error {
		req, ok := v.(*gw.ModelInferRequest)
		if !ok {
			return c.JSONPb.NewDecoder(r).Decode(v)
		}
		logger.Info("Received REST inference request")
		restReq := &RESTRequest{}
		if err := json.NewDecoder(r).Decode(restReq); err != nil {
			return err
		}
		transformRequest(restReq, req)
		return nil
	})
}

func transformRequest(restReq *RESTRequest, req *gw.ModelInferRequest) {
	req.Id = restReq.Id
	req.Parameters = restReq.Parameters
	req.Outputs = restReq.Outputs
	req.Inputs = make([]*gw.ModelInferRequest_InferInputTensor, len(restReq.Inputs))
	for i := range restReq.Inputs {
		req.Inputs[i] = (*gw.ModelInferRequest_InferInputTensor)(&restReq.Inputs[i])
	}
}

type RESTRequest struct {
	Id string `json:"id,omitempty"`
	//TODO figure out how to handle request-level content type parameter
	Parameters parameterMap                                       `json:"parameters,omitempty"`
	Inputs     []InputTensor                                      `json:"inputs,omitempty"`
	Outputs    []*gw.ModelInferRequest_InferRequestedOutputTensor `json:"outputs,omitempty"`
}

// Input tensors

type InputTensor gw.ModelInferRequest_InferInputTensor

type InputTensorMeta struct {
	Name       string       `json:"name"`
	Datatype   string       `json:"datatype"`
	Shape      []int64      `json:"shape"`
	Parameters parameterMap `json:"parameters"`
}

type InputTensorData struct {
	Data       tensorDataUnmarshaller `json:"data"`
	Parameters parameterMap           `json:"parameters"`
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
	isBytes := meta.Datatype == BYTES
	itd := &InputTensorData{Data: tensorDataUnmarshaller{
		target: target, shape: meta.Shape,
		bytes: isBytes, b64: isBytes && isBase64Content(meta.Parameters),
	}}
	if err := json.Unmarshal(data, itd); err != nil {
		return err
	}
	*t = InputTensor{
		Name:       meta.Name,
		Datatype:   meta.Datatype,
		Shape:      meta.Shape,
		Parameters: meta.Parameters,
		Contents:   contents,
	}

	return nil
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

type tensorDataUnmarshaller struct {
	shape  []int64
	bytes  bool
	b64    bool
	target interface{}
}

func (t *tensorDataUnmarshaller) UnmarshalJSON(data []byte) error {
	if t.bytes {
		return unmarshalBytesJson(t.target.(*[][]byte), t.shape, t.b64, data)
	}
	if len(t.shape) <= 1 {
		return json.Unmarshal(data, t.target) // single-dimension fast-path
	}
	start := -1
	for i, b := range data {
		if b == '[' {
			if start != -1 {
				if start != 0 {
					data = data[start:]
				}
				break // here we have nested arrays
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

func isSpace(c byte) bool {
	return c <= ' ' && (c == ' ' || c == '\t' || c == '\r' || c == '\n')
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

// Input parameters

var (
	NIL_PARAM   = &gw.InferParameter{}
	TRUE_PARAM  = &gw.InferParameter{ParameterChoice: &gw.InferParameter_BoolParam{BoolParam: true}}
	FALSE_PARAM = &gw.InferParameter{ParameterChoice: &gw.InferParameter_BoolParam{}}
)

type parameterMap map[string]*gw.InferParameter

func (p *parameterMap) MarshalJSON() ([]byte, error) {
	var pm map[string]interface{}
	if p != nil {
		pm = parameterMapToJson(*p)
	}
	return json.Marshal(pm)
}

func (p *parameterMap) UnmarshalJSON(data []byte) error {
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		return err
	}
	pm := make(parameterMap, len(jsonMap))
	for k, i := range jsonMap {
		switch v := i.(type) {
		case string:
			pm[k] = &gw.InferParameter{ParameterChoice: &gw.InferParameter_StringParam{StringParam: v}}
		case float64:
			intVal := int64(v)
			if float64(intVal) != v {
				logger.Error(nil, "Warning: Number parameter lost precision during int conversion",
					"parameter", k, "value", v)
			}
			pm[k] = &gw.InferParameter{ParameterChoice: &gw.InferParameter_Int64Param{Int64Param: intVal}}
		case bool:
			if v {
				pm[k] = TRUE_PARAM
			} else {
				pm[k] = FALSE_PARAM
			}
		case nil:
			pm[k] = NIL_PARAM
		default:
			logger.Error(nil, "Could not convert parameter of unsupported type (json array or object)",
				"parameter", k)
		}
	}
	*p = pm
	return nil
}
