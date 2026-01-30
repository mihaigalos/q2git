package main

import (
	"encoding/json"
	"fmt"

	"github.com/itchyny/gojq"
)

func ExecuteQuery(query string, data []byte) ([]byte, error) {
	jqQuery, err := gojq.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse jq query: %w", err)
	}

	var input interface{}
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input JSON: %w", err)
	}

	var results []interface{}
	iter := jqQuery.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return nil, fmt.Errorf("query execution error: %w", err)
		}
		results = append(results, v)
	}

	var output interface{}
	if len(results) == 1 {
		output = results[0]
	} else {
		output = results
	}

	return json.MarshalIndent(output, "", "  ")
}
