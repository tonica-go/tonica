package entities

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tonica-go/tonica/pkg/tonica/proto/entities"

	"gopkg.in/yaml.v3"
)

// Definition describes an entity loaded from metadata.
type Definition struct {
	ID          string
	DisplayName string
	Description string
	PrimaryKey  string
	Proto       string
	Fields      []FieldDefinition
	Metadata    map[string]string
}

// Field returns a field definition by id.
func (d Definition) Field(id string) (FieldDefinition, bool) {
	for _, field := range d.Fields {
		if field.ID == id {
			return field, true
		}
	}
	return FieldDefinition{}, false
}

// FieldDefinition describes a single attribute of an entity.
type FieldDefinition struct {
	ID              string
	DisplayName     string
	Type            entities.FieldType
	Required        bool
	Repeated        bool
	Sortable        bool
	FilterOperators []entities.FilterOperator
	Reference       *ReferenceDefinition
	Metadata        map[string]string
}

// ReferenceDefinition describes a relationship to another entity.
type ReferenceDefinition struct {
	Entity   string
	Field    string
	Label    string
	Metadata map[string]string
}

func (f FieldDefinition) Supports(op entities.FilterOperator) bool {
	for _, existing := range f.FilterOperators {
		if existing == op {
			return true
		}
	}
	return false
}

func (d Definition) ToProto() *entities.EntityDefinition {
	fields := make([]*entities.FieldDefinition, 0, len(d.Fields))
	for _, field := range d.Fields {
		fields = append(fields, &entities.FieldDefinition{
			Id:          field.ID,
			DisplayName: field.DisplayName,
			Type:        field.Type,
			Required:    field.Required,
			Repeated:    field.Repeated,
			Sortable:    field.Sortable,
			Filter: &entities.FilterDefinition{
				Operators: field.FilterOperators,
			},
			Metadata: field.Metadata,
		})
	}
	return &entities.EntityDefinition{
		Id:          d.ID,
		DisplayName: d.DisplayName,
		Description: d.Description,
		PrimaryKey:  d.PrimaryKey,
		Fields:      fields,
		Metadata:    d.Metadata,
	}
}

// LoadDefinitions reads all embedded YAML definitions.
func LoadDefinitions() (map[string]Definition, error) {
	entries, err := os.ReadDir("definitions")
	if err != nil {
		return nil, fmt.Errorf("read definitions directory: %w", err)
	}

	definitions := make(map[string]Definition, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		path := filepath.Join("definitions", entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}

		def, err := parseDefinition(data)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}

		key := strings.ToLower(def.ID)
		if _, exists := definitions[key]; exists {
			return nil, fmt.Errorf("duplicate entity id %q", def.ID)
		}

		definitions[key] = def
	}

	return definitions, nil
}

func parseDefinition(data []byte) (Definition, error) {
	var raw rawDefinition
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Definition{}, fmt.Errorf("unmarshal: %w", err)
	}

	id := strings.ToLower(strings.TrimSpace(raw.ID))
	if id == "" {
		return Definition{}, fmt.Errorf("id is required")
	}
	primaryKey := strings.TrimSpace(raw.PrimaryKey)
	if primaryKey == "" {
		return Definition{}, fmt.Errorf("primary_key is required")
	}
	if len(raw.Fields) == 0 {
		return Definition{}, fmt.Errorf("fields are required")
	}

	fields := make([]FieldDefinition, 0, len(raw.Fields))
	for _, rf := range raw.Fields {
		field, err := buildFieldDefinition(rf)
		if err != nil {
			return Definition{}, fmt.Errorf("field %q: %w", rf.ID, err)
		}
		fields = append(fields, field)
	}

	sort.SliceStable(fields, func(i, j int) bool { return fields[i].ID < fields[j].ID })

	metadata := cloneStringMap(raw.Metadata)

	return Definition{
		ID:          id,
		DisplayName: fallback(strings.TrimSpace(raw.DisplayName), humanizeIdentifier(id)),
		Description: strings.TrimSpace(raw.Description),
		PrimaryKey:  primaryKey,
		Proto:       strings.TrimSpace(raw.Proto),
		Fields:      fields,
		Metadata:    metadata,
	}, nil
}

func buildFieldDefinition(raw rawFieldDefinition) (FieldDefinition, error) {
	if strings.TrimSpace(raw.ID) == "" {
		return FieldDefinition{}, fmt.Errorf("id is required")
	}
	fieldID := strings.TrimSpace(raw.ID)
	fieldType, ok := parseFieldType(raw.Type)
	if !ok {
		return FieldDefinition{}, fmt.Errorf("unknown type %q", raw.Type)
	}

	metadata := cloneStringMap(raw.Metadata)

	filterOps := make([]entities.FilterOperator, 0, len(raw.Filter.Operators))
	for _, op := range raw.Filter.Operators {
		parsed, ok := parseFilterOperator(op)
		if !ok {
			return FieldDefinition{}, fmt.Errorf("invalid filter operator %q", op)
		}
		filterOps = append(filterOps, parsed)
	}

	reference, err := buildReferenceDefinition(raw.Reference)
	if err != nil {
		return FieldDefinition{}, fmt.Errorf("reference: %w", err)
	}
	if reference != nil {
		if metadata == nil {
			metadata = make(map[string]string)
		}
		if reference.Entity != "" {
			metadata["reference.entity"] = reference.Entity
		}
		if reference.Field != "" {
			metadata["reference.field"] = reference.Field
		}
		if reference.Label != "" {
			metadata["reference.label"] = reference.Label
		}
		if alias := reference.Metadata["entityId"]; alias != "" {
			metadata["reference.metadata.entityId"] = alias
		}
	}

	return FieldDefinition{
		ID:              fieldID,
		DisplayName:     fallback(strings.TrimSpace(raw.DisplayName), humanizeIdentifier(fieldID)),
		Type:            fieldType,
		Required:        raw.Required,
		Repeated:        raw.Repeated,
		Sortable:        raw.Sortable,
		FilterOperators: filterOps,
		Reference:       reference,
		Metadata:        metadata,
	}, nil
}

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return defaultValue
}

func humanizeIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	replacer := strings.NewReplacer("_", " ", "-", " ")
	value = replacer.Replace(value)
	parts := strings.Fields(value)
	for i, part := range parts {
		if len(part) == 1 {
			parts[i] = strings.ToUpper(part)
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
}

func parseFieldType(value string) (entities.FieldType, bool) {
	switch strings.ToLower(value) {
	case "string":
		return entities.FieldType_FIELD_TYPE_STRING, true
	case "number", "float", "int", "integer":
		return entities.FieldType_FIELD_TYPE_NUMBER, true
	case "bool", "boolean":
		return entities.FieldType_FIELD_TYPE_BOOLEAN, true
	case "datetime", "timestamp":
		return entities.FieldType_FIELD_TYPE_DATETIME, true
	case "uuid", "guid":
		return entities.FieldType_FIELD_TYPE_UUID, true
	case "enum":
		return entities.FieldType_FIELD_TYPE_ENUM, true
	case "object":
		return entities.FieldType_FIELD_TYPE_OBJECT, true
	case "array", "list":
		return entities.FieldType_FIELD_TYPE_ARRAY, true
	case "":
		return entities.FieldType_FIELD_TYPE_UNSPECIFIED, true
	default:
		return entities.FieldType_FIELD_TYPE_UNSPECIFIED, false
	}
}

func parseFilterOperator(value string) (entities.FilterOperator, bool) {
	switch strings.ToLower(value) {
	case "eq", "equals", "=":
		return entities.FilterOperator_FILTER_OPERATOR_EQ, true
	case "ne", "not_equals", "!=":
		return entities.FilterOperator_FILTER_OPERATOR_NE, true
	case "contains":
		return entities.FilterOperator_FILTER_OPERATOR_CONTAINS, true
	case "in":
		return entities.FilterOperator_FILTER_OPERATOR_IN, true
	case "gt", ">":
		return entities.FilterOperator_FILTER_OPERATOR_GT, true
	case "gte", ">=":
		return entities.FilterOperator_FILTER_OPERATOR_GTE, true
	case "lt", "<":
		return entities.FilterOperator_FILTER_OPERATOR_LT, true
	case "lte", "<=":
		return entities.FilterOperator_FILTER_OPERATOR_LTE, true
	case "":
		return entities.FilterOperator_FILTER_OPERATOR_UNSPECIFIED, true
	default:
		return entities.FilterOperator_FILTER_OPERATOR_UNSPECIFIED, false
	}
}

type rawDefinition struct {
	ID          string               `yaml:"id"`
	DisplayName string               `yaml:"display_name"`
	Description string               `yaml:"description"`
	PrimaryKey  string               `yaml:"primary_key"`
	Proto       string               `yaml:"proto"`
	Fields      []rawFieldDefinition `yaml:"fields"`
	Metadata    map[string]string    `yaml:"metadata"`
}

type rawFieldDefinition struct {
	ID          string                  `yaml:"id"`
	DisplayName string                  `yaml:"display_name"`
	Type        string                  `yaml:"type"`
	Required    bool                    `yaml:"required"`
	Repeated    bool                    `yaml:"repeated"`
	Sortable    bool                    `yaml:"sortable"`
	Filter      rawFilterDefinition     `yaml:"filter"`
	Reference   *rawReferenceDefinition `yaml:"reference"`
	Metadata    map[string]string       `yaml:"metadata"`
}

type rawFilterDefinition struct {
	Operators []string `yaml:"operators"`
}

type rawReferenceDefinition struct {
	Entity   string            `yaml:"entity"`
	Field    string            `yaml:"field"`
	Label    string            `yaml:"label"`
	Metadata map[string]string `yaml:"metadata"`
}

func buildReferenceDefinition(raw *rawReferenceDefinition) (*ReferenceDefinition, error) {
	if raw == nil {
		return nil, nil
	}
	entity := strings.ToLower(strings.TrimSpace(raw.Entity))
	if entity == "" {
		return nil, fmt.Errorf("entity is required")
	}
	field := strings.TrimSpace(raw.Field)
	if field == "" {
		return nil, fmt.Errorf("field is required")
	}
	reference := &ReferenceDefinition{
		Entity:   entity,
		Field:    field,
		Label:    strings.TrimSpace(raw.Label),
		Metadata: cloneStringMap(raw.Metadata),
	}
	return reference, nil
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	copyVal := make(map[string]string, len(input))
	for key, value := range input {
		copyVal[key] = value
	}
	return copyVal
}
