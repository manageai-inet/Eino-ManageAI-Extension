package manageai

import (
	"encoding/json"
	"strings"
	"testing"
)

type MyTestStruct struct {
	Name string `json:"name" jsonschema:"The name of the user"`
	Age  int    `json:"age,omitzero" jsonschema:"The age of the user"`
}

func TestStructToTool(t *testing.T) {
	tools, err := StructToTool[MyTestStruct]("my_tool", "my_tool_desc", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tools.Type != "function" {
		t.Errorf("expected Type function, got %s", tools.Type)
	}
	if tools.Function.Name != "my_tool" {
		t.Errorf("expected Name my_tool, got %s", tools.Function.Name)
	}
	if tools.Function.Description != "my_tool_desc" {
		t.Errorf("expected Description my_tool_desc, got %s", tools.Function.Description)
	}
	if tools.Function.Strict != true {
		t.Errorf("expected Strict true, got false")
	}
	if tools.Function.Parameters == nil {
		t.Fatalf("expected Parameters not to be nil")
	}

	// Simple check on the marshaled schema
	schemaBytes, err := json.Marshal(tools.Function.Parameters)
	if err != nil {
		t.Fatalf("unexpected error marshaling parameters: %v", err)
	}
	
	schemaStr := string(schemaBytes)
	if !strings.Contains(schemaStr, "name") {
		t.Errorf("expected schema to contain 'name', got: %s", schemaStr)
	}
	if !strings.Contains(schemaStr, "age") {
		t.Errorf("expected schema to contain 'age', got: %s", schemaStr)
	}
	t.Logf("Schema: %s", schemaStr)
}

func TestHandleToolCall(t *testing.T) {
	jsonArgs := `{"name": "Alice", "age": 30}`

	result, err := HandleToolCall[MyTestStruct](jsonArgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatalf("expected result not to be nil")
	}
	if result.Name != "Alice" {
		t.Errorf("expected Name Alice, got %s", result.Name)
	}
	if result.Age != 30 {
		t.Errorf("expected Age 30, got %d", result.Age)
	}
	t.Logf("Result: %+v", *result)

	// test error with invalid json type
	invalidJsonArgs := `{"name": "Alice", "age": "thirty"}`
	_, err = HandleToolCall[MyTestStruct](invalidJsonArgs)
	if err == nil {
		t.Errorf("expected error for invalid json, got nil")
	}
	
	// test error with broken json
	brokenJsonArgs := `{"name": "Alice", "age": 30`
	_, err = HandleToolCall[MyTestStruct](brokenJsonArgs)
	if err == nil {
		t.Errorf("expected error for broken json, got nil")
	}
}
