package main

import (
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"sort"
	"strings"
	"unicode"
)

type document struct {
	OpenAPI    string              `json:"openapi"`
	Paths      map[string]pathItem `json:"paths"`
	Components struct {
		Schemas map[string]schema `json:"schemas"`
	} `json:"components"`
}

type pathItem map[string]operation

type operation struct {
	OperationID string `json:"operationId"`
	RequestBody *struct {
		Content map[string]mediaType `json:"content"`
	} `json:"requestBody"`
	Responses map[string]response `json:"responses"`
}

type response struct {
	Content map[string]mediaType `json:"content"`
}

type mediaType struct {
	Schema schema `json:"schema"`
}

type schema struct {
	Ref                  string            `json:"$ref"`
	Type                 string            `json:"type"`
	Properties           map[string]schema `json:"properties"`
	Required             []string          `json:"required"`
	Items                *schema           `json:"items"`
	Enum                 []json.RawMessage `json:"enum"`
	AdditionalProperties json.RawMessage   `json:"additionalProperties"`
}

type renderer struct {
	typeNames map[string]string
	usedNames map[string]int
}

func generate(data []byte) ([]byte, error) {
	var spec document
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parse OpenAPI spec: %w", err)
	}
	if !strings.HasPrefix(spec.OpenAPI, "3.1.") {
		return nil, fmt.Errorf("unsupported OpenAPI version %q: want 3.1.x", spec.OpenAPI)
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse OpenAPI reference tree: %w", err)
	}
	if err := validateReferences(raw, spec.Components.Schemas); err != nil {
		return nil, err
	}

	render := newRenderer(spec.Components.Schemas)
	var out strings.Builder
	out.WriteString("// Code generated from OpenAPI spec. DO NOT EDIT.\n\n")
	out.WriteString("package opencode\n\n")

	schemaNames := sortedKeys(spec.Components.Schemas)
	for _, name := range schemaNames {
		render.writeDeclaration(&out, render.typeNames[name], spec.Components.Schemas[name])
	}

	paths := sortedKeys(spec.Paths)
	for _, path := range paths {
		item := spec.Paths[path]
		for _, method := range []string{"get", "post", "put", "patch", "delete", "options", "head", "trace"} {
			op, ok := item[method]
			if !ok || op.OperationID == "" {
				continue
			}
			baseName := "OpenCode" + goName(op.OperationID)
			if op.RequestBody != nil {
				if requestSchema, ok := jsonSchema(op.RequestBody.Content); ok {
					render.writeDeclaration(&out, render.uniqueName(baseName+"Request"), requestSchema)
				}
			}
			if responseSchema, ok := successResponseSchema(op.Responses); ok {
				render.writeDeclaration(&out, render.uniqueName(baseName+"Response"), responseSchema)
			}
		}
	}

	formatted, err := format.Source([]byte(out.String()))
	if err != nil {
		return nil, fmt.Errorf("format generated Go: %w", err)
	}
	return formatted, nil
}

func validateReferences(value any, schemas map[string]schema) error {
	const prefix = "#/components/schemas/"
	switch typed := value.(type) {
	case map[string]any:
		if refValue, ok := typed["$ref"]; ok {
			ref, ok := refValue.(string)
			if !ok {
				return fmt.Errorf("OpenAPI $ref must be a string")
			}
			if !strings.HasPrefix(ref, prefix) {
				return fmt.Errorf("unsupported non-local OpenAPI reference %q", ref)
			}
			name := strings.TrimPrefix(ref, prefix)
			if _, ok := schemas[name]; !ok {
				return fmt.Errorf("OpenAPI reference %q names missing schema %q", ref, name)
			}
		}
		for _, child := range typed {
			if err := validateReferences(child, schemas); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range typed {
			if err := validateReferences(child, schemas); err != nil {
				return err
			}
		}
	}
	return nil
}

func newRenderer(schemas map[string]schema) *renderer {
	render := &renderer{
		typeNames: make(map[string]string, len(schemas)),
		usedNames: make(map[string]int, len(schemas)),
	}
	for _, rawName := range sortedKeys(schemas) {
		render.typeNames[rawName] = render.uniqueName("OpenCode" + goName(rawName))
	}
	return render
}

func (render *renderer) uniqueName(base string) string {
	render.usedNames[base]++
	if render.usedNames[base] == 1 {
		return base
	}
	return fmt.Sprintf("%s%d", base, render.usedNames[base])
}

func (render *renderer) writeDeclaration(out *strings.Builder, name string, value schema) {
	if value.Ref != "" {
		fmt.Fprintf(out, "type %s = %s\n\n", name, render.goType(value))
		return
	}
	if value.Type != "object" {
		fmt.Fprintf(out, "type %s %s\n\n", name, render.goType(value))
		return
	}
	required := make(map[string]bool, len(value.Required))
	for _, field := range value.Required {
		required[field] = true
	}
	type field struct {
		name     string
		typeName string
		tag      string
	}
	fields := make([]field, 0, len(value.Properties))
	for _, property := range sortedKeys(value.Properties) {
		tag := property
		if !required[property] {
			tag += ",omitempty"
		}
		propertySchema := value.Properties[property]
		typeName := render.fieldType(out, name+goName(property), propertySchema)
		if !required[property] && render.isInlineObject(propertySchema) {
			typeName = "*" + typeName
		}
		fields = append(fields, field{name: goName(property), typeName: typeName, tag: tag})
	}
	fmt.Fprintf(out, "type %s struct {\n", name)
	for _, field := range fields {
		fmt.Fprintf(out, "\t%s %s `json:%q`\n", field.name, field.typeName, field.tag)
	}
	out.WriteString("}\n\n")
}

func (render *renderer) fieldType(out *strings.Builder, suggestedName string, value schema) string {
	if render.isInlineObject(value) {
		name := render.uniqueName(suggestedName)
		render.writeDeclaration(out, name, value)
		return name
	}
	if value.Type == "array" && value.Items != nil && render.isInlineObject(*value.Items) {
		name := render.uniqueName(suggestedName + "Item")
		render.writeDeclaration(out, name, *value.Items)
		return "[]" + name
	}
	return render.goType(value)
}

func (render *renderer) isInlineObject(value schema) bool {
	return value.Ref == "" && value.Type == "object" && len(value.Properties) > 0
}

func (render *renderer) goType(value schema) string {
	if value.Ref != "" {
		const prefix = "#/components/schemas/"
		rawName := strings.TrimPrefix(value.Ref, prefix)
		if name, ok := render.typeNames[rawName]; ok {
			return name
		}
		return "any"
	}
	switch value.Type {
	case "boolean":
		return "bool"
	case "integer":
		return "int64"
	case "number":
		return "float64"
	case "string":
		return "string"
	case "array":
		if value.Items == nil {
			return "[]any"
		}
		return "[]" + render.goType(*value.Items)
	case "object":
		if len(value.Properties) == 0 {
			return render.mapType(value.AdditionalProperties)
		}
		var inline strings.Builder
		inline.WriteString("struct {\n")
		required := make(map[string]bool, len(value.Required))
		for _, field := range value.Required {
			required[field] = true
		}
		for _, property := range sortedKeys(value.Properties) {
			tag := property
			if !required[property] {
				tag += ",omitempty"
			}
			fmt.Fprintf(&inline, "\t%s %s `json:%q`\n", goName(property), render.goType(value.Properties[property]), tag)
		}
		inline.WriteString("}")
		return inline.String()
	default:
		return "any"
	}
}

func (render *renderer) mapType(additional json.RawMessage) string {
	if len(additional) == 0 || string(additional) == "true" || string(additional) == "false" {
		return "map[string]any"
	}
	var value schema
	if err := json.Unmarshal(additional, &value); err != nil {
		return "map[string]any"
	}
	return "map[string]" + render.goType(value)
}

func jsonSchema(content map[string]mediaType) (schema, bool) {
	media, ok := content["application/json"]
	return media.Schema, ok
}

func successResponseSchema(responses map[string]response) (schema, bool) {
	statuses := sortedKeys(responses)
	for _, status := range statuses {
		if len(status) != 3 || status[0] != '2' {
			continue
		}
		if value, ok := jsonSchema(responses[status].Content); ok {
			return value, true
		}
	}
	return schema{}, false
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func goName(value string) string {
	if value == "id" {
		return "ID"
	}
	var out strings.Builder
	upperNext := true
	for _, char := range value {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) {
			upperNext = true
			continue
		}
		if out.Len() == 0 && unicode.IsDigit(char) {
			out.WriteByte('N')
		}
		if upperNext {
			char = unicode.ToUpper(char)
			upperNext = false
		}
		out.WriteRune(char)
	}
	if out.Len() == 0 {
		return "Value"
	}
	return out.String()
}

func run(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: openapi-gen <spec.json> <output.go>")
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("read spec: %w", err)
	}
	generated, err := generate(data)
	if err != nil {
		return err
	}
	if err := os.WriteFile(args[1], generated, 0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
