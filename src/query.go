package main

import (
	"encoding/json"
	"fmt"

	"github.com/itchyny/gojq"
)

// ExecuteQuery executes a jq query on the input data
func ExecuteQuery(query string, data []byte) ([]byte, error) {
	// Parse the jq query
	jqQuery, err := gojq.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse jq query: %w", err)
	}

	// Unmarshal input data
	var input interface{}
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input JSON: %w", err)
	}

	// Execute the query
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

	// If there's only one result, return it directly (not wrapped in array)
	var output interface{}
	if len(results) == 1 {
		output = results[0]
	} else {
		output = results
	}

	// Marshal results back to JSON
	return json.MarshalIndent(output, "", "  ")
}
