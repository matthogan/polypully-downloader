package message

import (
	"encoding/json"
	"fmt"
)

type EncoderAPI interface {
	Encode(interface{}) ([]byte, error)
}

func NewEncoder() EncoderAPI {
	return &JsonEncoder{}
}

type JsonEncoder struct {
}

func (e *JsonEncoder) Encode(a interface{}) ([]byte, error) {
	data, err := json.Marshal(a)
	if err != nil {
		return nil, fmt.Errorf("encoding failed %s", err)
	}
	return data, err
}
