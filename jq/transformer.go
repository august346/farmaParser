package jq

import (
	"errors"

	"github.com/itchyny/gojq"
)

func Transform(data interface{}, JQScript string) (interface{}, error) {
	query, err := gojq.Parse(JQScript)
	if err != nil {
		return nil, err
	}

	code, err := gojq.Compile(query)
	if err != nil {
		return nil, err
	}

	iter := code.Run(data)
	transformed, ok := iter.Next()
	if !ok {
		return nil, errors.New("Empty")
	}
	if err, ok := transformed.(error); ok {
		return nil, err
	}

	return transformed, nil
}
