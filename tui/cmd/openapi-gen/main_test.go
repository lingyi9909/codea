package main

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"strings"
	"testing"
)

func TestGenerateFollowsPathsAndComponentSchemas(t *testing.T) {
	const spec = `{
	  "openapi": "3.1.0",
	  "paths": {
	    "/session": {
	      "post": {
	        "operationId": "session.create",
	        "requestBody": {
	          "content": {
	            "application/json": {
	              "schema": {
	                "type": "object",
	                "properties": {
	                  "parentID": {"type": "string"},
	                  "agents": {
	                    "type": "array",
	                    "items": {"$ref": "#/components/schemas/Agent"}
	                  }
	                },
	                "required": ["agents"]
	              }
	            }
	          }
	        },
	        "responses": {
	          "200": {
	            "content": {
	              "application/json": {
	                "schema": {"$ref": "#/components/schemas/Session"}
	              }
	            }
	          }
	        }
	      }
	    }
	  },
	  "components": {
	    "schemas": {
	      "Agent": {
	        "type": "string",
	        "enum": ["build", "plan"]
	      },
	      "Session": {
	        "type": "object",
	        "properties": {
	          "id": {"type": "string"},
	          "archived": {"type": "boolean"}
	        },
	        "required": ["id"]
	      }
	    }
	  }
	}`

	got, err := generate([]byte(spec))
	if err != nil {
		t.Fatalf("generate returned error: %v", err)
	}
	text := string(got)
	normalized := strings.Join(strings.Fields(text), " ")
	for _, want := range []string{
		"// Code generated from OpenAPI spec. DO NOT EDIT.",
		"package opencode",
		"type OpenCodeAgent string",
		"type OpenCodeSession struct {",
		"ID string `json:\"id\"`",
		"Archived bool `json:\"archived,omitempty\"`",
		"type OpenCodeSessionCreateRequest struct {",
		"ParentID string `json:\"parentID,omitempty\"`",
		"Agents []OpenCodeAgent `json:\"agents\"`",
		"type OpenCodeSessionCreateResponse = OpenCodeSession",
	} {
		if !strings.Contains(normalized, want) {
			t.Errorf("generated output missing %q:\n%s", want, text)
		}
	}

	again, err := generate([]byte(spec))
	if err != nil {
		t.Fatalf("second generate returned error: %v", err)
	}
	if string(again) != text {
		t.Fatal("generation is not deterministic")
	}
}

func TestGenerateLockedSpecProducesUsableTask2DTOs(t *testing.T) {
	data, err := os.ReadFile("../../../runtime/openapi/opencode-1.18.11.json")
	if err != nil {
		t.Fatalf("read locked spec: %v", err)
	}
	generated, err := generate(data)
	if err != nil {
		t.Fatalf("generate locked spec: %v", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "dto.go", generated, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse generated DTOs: %v", err)
	}
	config := types.Config{Importer: importer.Default()}
	if _, err := config.Check("codea/tui/internal/opencode", fset, []*ast.File{file}, nil); err != nil {
		t.Fatalf("type-check generated DTOs: %v", err)
	}

	text := strings.Join(strings.Fields(string(generated)), " ")
	for _, want := range []string{
		"type OpenCodeSessionCreateRequest struct {",
		"Agent string `json:\"agent,omitempty\"`",
		"WorkspaceID string `json:\"workspaceID,omitempty\"`",
		"type OpenCodeSessionPromptAsyncRequest struct {",
		"type OpenCodeSessionPromptAsyncRequestModel struct { ModelID string `json:\"modelID\"` ProviderID string `json:\"providerID\"` }",
		"Model *OpenCodeSessionPromptAsyncRequestModel `json:\"model,omitempty\"`",
		"Tools map[string]bool `json:\"tools,omitempty\"`",
		"type OpenCodePermissionReplyRequest struct {",
		"Message string `json:\"message,omitempty\"`",
		"Reply string `json:\"reply\"`",
		"type OpenCodePermissionReplyResponse bool",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("generated locked DTOs missing %q", want)
		}
	}
	if strings.Contains(text, "type OpenCodePermissionReplyRequest struct { Remember") ||
		strings.Contains(text, "Remember bool `json:\"remember") {
		t.Fatal("non-deprecated permission reply request must not contain remember")
	}
}

func TestGenerateRejectsUnknownComponentReference(t *testing.T) {
	const spec = `{
	  "openapi": "3.1.0",
	  "paths": {},
	  "components": {
	    "schemas": {
	      "Envelope": {"$ref": "#/components/schemas/Missing"}
	    }
	  }
	}`

	_, err := generate([]byte(spec))
	if err == nil || !strings.Contains(err.Error(), "Missing") {
		t.Fatalf("generate error = %v, want unknown reference error", err)
	}
}

func TestCommittedDTOIsCurrent(t *testing.T) {
	spec, err := os.ReadFile("../../../runtime/openapi/opencode-1.18.11.json")
	if err != nil {
		t.Fatalf("read locked spec: %v", err)
	}
	want, err := generate(spec)
	if err != nil {
		t.Fatalf("generate locked spec: %v", err)
	}
	got, err := os.ReadFile("../../internal/opencode/dto.go")
	if err != nil {
		t.Fatalf("read committed DTO: %v", err)
	}
	if string(got) != string(want) {
		t.Fatal("internal/opencode/dto.go is stale; rerun openapi-gen")
	}
}

// TestPromptAsyncJSONUsesIDNotMessageID is the regression test for
// OpenCode v1.18.11 Known Protocol Deviation 1:
//
//	The locked OpenAPI spec declares prompt_async.messageID, but the real
//	runtime only processes JSON field "id".  If someone removes the
//	fieldJSONOverrides entry, this test will catch it.
func TestPromptAsyncJSONUsesIDNotMessageID(t *testing.T) {
	spec, err := os.ReadFile("../../../runtime/openapi/opencode-1.18.11.json")
	if err != nil {
		t.Fatalf("read locked spec: %v", err)
	}
	generated, err := generate(spec)
	if err != nil {
		t.Fatalf("generate locked spec: %v", err)
	}

	src := string(generated)

	// OpenCode v1.18.11 Protocol Deviation 1:
	// prompt_async request must use "id" not "messageID".
	// Parse the generated code to check the specific struct field tag.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "dto.go", src, 0)
	if err != nil {
		t.Fatalf("parse generated DTO: %v", err)
	}

	var found bool
	ast.Inspect(f, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Name.Name != "OpenCodeSessionPromptAsyncRequest" {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			return true
		}
		for _, field := range st.Fields.List {
			if len(field.Names) == 1 && field.Names[0].Name == "MessageID" {
				found = true
				tag := field.Tag.Value
				if !strings.Contains(tag, `json:"id,omitempty"`) {
					t.Errorf("OpenCodeSessionPromptAsyncRequest.MessageID tag = %s, want json:\"id,omitempty\"", tag)
				}
				if strings.Contains(tag, `json:"messageID"`) {
					t.Errorf("OpenCodeSessionPromptAsyncRequest.MessageID tag = %s, must NOT contain \"messageID\"", tag)
				}
			}
		}
		return true
	})
	if !found {
		t.Error("OpenCodeSessionPromptAsyncRequest.MessageID field not found in generated DTO")
	}
}
