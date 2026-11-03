package coakkahttp

import (
	"errors"
	"fmt"
	"unsafe"
)

const (
	maximumMetadataTextBytes   = 64 << 10
	maximumRouteMetadataBytes  = 16 << 20
	maximumMetadataItems       = 16_384
	maximumMetadataSchemaNodes = 16_384
	maximumMetadataSchemaDepth = 128
)

// APIParameterLocation is Core's language-neutral documented parameter location.
type APIParameterLocation uint32

const (
	// APIParameterPath documents one encoded-path capture.
	APIParameterPath APIParameterLocation = 1
	// APIParameterQuery documents one query parameter.
	APIParameterQuery APIParameterLocation = 2
	// APIParameterHeader documents one HTTP field.
	APIParameterHeader APIParameterLocation = 3
	// APIParameterCookie documents one cookie parameter.
	APIParameterCookie APIParameterLocation = 4
)

// APISchemaKind is Core's language-neutral documented value shape.
type APISchemaKind uint32

const (
	// APISchemaBoolean is a boolean value.
	APISchemaBoolean APISchemaKind = 1
	// APISchemaInteger is an integer value.
	APISchemaInteger APISchemaKind = 2
	// APISchemaNumber is a floating-point or decimal value.
	APISchemaNumber APISchemaKind = 3
	// APISchemaString is a text value.
	APISchemaString APISchemaKind = 4
	// APISchemaBytes is an opaque byte value.
	APISchemaBytes APISchemaKind = 5
	// APISchemaObject is a named-property object.
	APISchemaObject APISchemaKind = 6
	// APISchemaArray is a homogeneous array.
	APISchemaArray APISchemaKind = 7
	// APISchemaReference names one route-local schema definition.
	APISchemaReference APISchemaKind = 8
)

// APIAdditionalProperties is documented object-schema additional-property intent.
type APIAdditionalProperties uint32

const (
	// APIAdditionalPropertiesUnspecified delegates semantic rejection to Core.
	APIAdditionalPropertiesUnspecified APIAdditionalProperties = iota
	// APIAdditionalPropertiesForbid rejects undeclared object properties.
	APIAdditionalPropertiesForbid
	// APIAdditionalPropertiesAllow permits undeclared object properties.
	APIAdditionalPropertiesAllow
)

// APIExample is one named example. Value is copied during native projection.
type APIExample struct {
	// Name is the route-local example identity.
	Name string
	// Summary is optional public documentation.
	Summary string
	// Value is an opaque language-neutral payload.
	Value []byte
}

// APISchemaProperty is one documented object property.
type APISchemaProperty struct {
	// Name is the property name.
	Name string
	// Description is optional public documentation.
	Description string
	// Schema documents the property value.
	Schema *APISchema
	// Required marks the property as required.
	Required bool
}

// APISchema is one recursive language-neutral schema. Core owns all semantic
// validation; the connector enforces only finite graph and memory bounds.
type APISchema struct {
	// Kind selects the documented value shape.
	Kind APISchemaKind
	// Format is an optional shape-specific hint.
	Format string
	// Reference names one route-local definition for APISchemaReference.
	Reference string
	// EnumValues is the optional allowed text set.
	EnumValues []string
	// Items documents array elements.
	Items *APISchema
	// Properties documents object members.
	Properties []APISchemaProperty
	// AdditionalProperties is explicit object-property intent.
	AdditionalProperties APIAdditionalProperties
	// Nullable is meaningful only when HasNullable is true.
	Nullable bool
	// HasNullable distinguishes explicit false from absence.
	HasNullable bool
}

// APISchemaDefinition is one named route-local schema.
type APISchemaDefinition struct {
	// Name is the route-local definition identity.
	Name string
	// Schema is the definition value.
	Schema *APISchema
}

// APIParameter documents one operation parameter.
type APIParameter struct {
	// Location selects path, query, header, or cookie.
	Location APIParameterLocation
	// Name is the parameter name.
	Name string
	// Description is optional public documentation.
	Description string
	// Schema documents the parameter value.
	Schema *APISchema
	// Examples are copied named examples.
	Examples []APIExample
	// Required marks the parameter as required.
	Required bool
}

// APIMediaType documents one HTTP media representation.
type APIMediaType struct {
	// MediaType is the HTTP media type.
	MediaType string
	// Schema documents its value.
	Schema *APISchema
	// Examples are copied named examples.
	Examples []APIExample
}

// APIRequestBody documents an optional operation request body.
type APIRequestBody struct {
	// Description is optional public documentation.
	Description string
	// Content is the supported media representation set.
	Content []APIMediaType
	// Required marks the body as required.
	Required bool
}

// APIResponseHeader documents one response field.
type APIResponseHeader struct {
	// Name is the HTTP field name.
	Name string
	// Description is optional public documentation.
	Description string
	// Schema documents the field value.
	Schema *APISchema
	// Examples are copied named examples.
	Examples []APIExample
}

// APIResponse documents one status response or a default response.
type APIResponse struct {
	// StatusCode is meaningful only when HasStatusCode is true.
	StatusCode uint32
	// Description is required by Core for published metadata.
	Description string
	// Headers documents response fields.
	Headers []APIResponseHeader
	// Content documents response representations.
	Content []APIMediaType
	// HasStatusCode distinguishes an explicit status from the default response.
	HasStatusCode bool
}

// APISecurityUse references one application-owned scheme and its scopes.
type APISecurityUse struct {
	// SchemeReference is the application-owned scheme identity.
	SchemeReference string
	// Scopes are the required authorization scopes.
	Scopes []string
}

// APISecurityRequirement is one AND-group of security uses.
type APISecurityRequirement struct {
	// Uses must jointly satisfy this alternative.
	Uses []APISecurityUse
}

// RouteAPIMetadata is optional documentation stored beside one dispatch route.
// It never enters request dispatch or handler values.
type RouteAPIMetadata struct {
	// OperationID is the stable operation identity.
	OperationID string
	// Summary is short public documentation.
	Summary string
	// Description is full public documentation.
	Description string
	// Tags are unique operation group names.
	Tags []string
	// Parameters documents operation inputs.
	Parameters []APIParameter
	// RequestBody optionally documents one body.
	RequestBody *APIRequestBody
	// Responses documents status/default outputs.
	Responses []APIResponse
	// SecurityRequirements are alternative security AND-groups.
	SecurityRequirements []APISecurityRequirement
	// SchemaDefinitions are route-local reusable schemas.
	SchemaDefinitions []APISchemaDefinition
	// Deprecated marks the operation as deprecated.
	Deprecated bool
}

type metadataCloner struct {
	bytes uint64
}

type routeMetadataCloner struct {
	owner  *metadataCloner
	items  uint32
	nodes  uint32
	active map[*APISchema]bool
	memo   map[*APISchema]*APISchema
}

func (cloner *metadataCloner) retain(size uint64) error {
	if cloner.bytes > maximumRouteMetadataBytes ||
		size > maximumRouteMetadataBytes-cloner.bytes {
		return errors.New("route metadata exceeds the Go byte bound")
	}
	cloner.bytes += size
	return nil
}

func (cloner *metadataCloner) text(value string) error {
	if len(value) > maximumMetadataTextBytes {
		return errors.New("route metadata text exceeds the Go bound")
	}
	return cloner.retain(uint64(len(value)))
}

func (cloner *routeMetadataCloner) addItems(count int, size uintptr) error {
	if count < 0 || uint64(count) > uint64(maximumMetadataItems-cloner.items) {
		return errors.New("route metadata exceeds the Go item bound")
	}
	cloner.items += uint32(count)
	return cloner.owner.retain(uint64(count) * uint64(size))
}

func (cloner *routeMetadataCloner) textArray(source []string) ([]string, error) {
	if err := cloner.addItems(len(source), unsafe.Sizeof(string(""))); err != nil {
		return nil, err
	}
	clone := make([]string, len(source))
	for index, value := range source {
		if err := cloner.owner.text(value); err != nil {
			return nil, fmt.Errorf("text %d: %w", index, err)
		}
		clone[index] = value
	}
	return clone, nil
}

func (cloner *routeMetadataCloner) schema(source *APISchema, depth uint32) (*APISchema, error) {
	if source == nil {
		return nil, nil
	}
	if depth > maximumMetadataSchemaDepth {
		return nil, errors.New("route metadata schema exceeds the Go depth bound")
	}
	if cloner.active[source] {
		return nil, errors.New("route metadata schema must not contain a cycle")
	}
	if existing := cloner.memo[source]; existing != nil {
		return existing, nil
	}
	if cloner.nodes == maximumMetadataSchemaNodes {
		return nil, errors.New("route metadata schema exceeds the Go node bound")
	}
	if err := cloner.owner.retain(uint64(unsafe.Sizeof(APISchema{}))); err != nil {
		return nil, err
	}
	cloner.nodes++
	clone := *source
	cloner.memo[source] = &clone
	cloner.active[source] = true
	defer delete(cloner.active, source)
	if err := cloner.owner.text(source.Format); err != nil {
		return nil, err
	}
	if err := cloner.owner.text(source.Reference); err != nil {
		return nil, err
	}
	var err error
	if clone.EnumValues, err = cloner.textArray(source.EnumValues); err != nil {
		return nil, err
	}
	if clone.Items, err = cloner.schema(source.Items, depth+1); err != nil {
		return nil, err
	}
	if err = cloner.addItems(len(source.Properties), unsafe.Sizeof(APISchemaProperty{})); err != nil {
		return nil, err
	}
	clone.Properties = make([]APISchemaProperty, len(source.Properties))
	for index, property := range source.Properties {
		if err = cloner.owner.text(property.Name); err != nil {
			return nil, err
		}
		if err = cloner.owner.text(property.Description); err != nil {
			return nil, err
		}
		clone.Properties[index] = property
		if clone.Properties[index].Schema, err = cloner.schema(property.Schema, depth+1); err != nil {
			return nil, fmt.Errorf("property %d: %w", index, err)
		}
	}
	return &clone, nil
}

func (cloner *routeMetadataCloner) examples(source []APIExample) ([]APIExample, error) {
	if err := cloner.addItems(len(source), unsafe.Sizeof(APIExample{})); err != nil {
		return nil, err
	}
	clone := make([]APIExample, len(source))
	for index, example := range source {
		if err := cloner.owner.text(example.Name); err != nil {
			return nil, err
		}
		if err := cloner.owner.text(example.Summary); err != nil {
			return nil, err
		}
		if err := cloner.owner.retain(uint64(len(example.Value))); err != nil {
			return nil, err
		}
		clone[index] = example
		clone[index].Value = append([]byte(nil), example.Value...)
	}
	return clone, nil
}

func (cloner *routeMetadataCloner) mediaTypes(source []APIMediaType) ([]APIMediaType, error) {
	if err := cloner.addItems(len(source), unsafe.Sizeof(APIMediaType{})); err != nil {
		return nil, err
	}
	clone := make([]APIMediaType, len(source))
	for index, media := range source {
		if err := cloner.owner.text(media.MediaType); err != nil {
			return nil, err
		}
		clone[index] = media
		var err error
		if clone[index].Schema, err = cloner.schema(media.Schema, 1); err != nil {
			return nil, fmt.Errorf("media %d schema: %w", index, err)
		}
		if clone[index].Examples, err = cloner.examples(media.Examples); err != nil {
			return nil, fmt.Errorf("media %d examples: %w", index, err)
		}
	}
	return clone, nil
}

func (cloner *metadataCloner) route(source *RouteAPIMetadata) (*RouteAPIMetadata, error) {
	if source == nil {
		return nil, nil
	}
	if err := cloner.retain(uint64(unsafe.Sizeof(RouteAPIMetadata{}))); err != nil {
		return nil, err
	}
	route := routeMetadataCloner{
		owner: cloner, active: make(map[*APISchema]bool), memo: make(map[*APISchema]*APISchema),
	}
	clone := *source
	for _, value := range []string{source.OperationID, source.Summary, source.Description} {
		if err := cloner.text(value); err != nil {
			return nil, err
		}
	}
	var err error
	if clone.Tags, err = route.textArray(source.Tags); err != nil {
		return nil, err
	}
	if err = route.addItems(len(source.Parameters), unsafe.Sizeof(APIParameter{})); err != nil {
		return nil, err
	}
	clone.Parameters = make([]APIParameter, len(source.Parameters))
	for index, parameter := range source.Parameters {
		for _, value := range []string{parameter.Name, parameter.Description} {
			if err = cloner.text(value); err != nil {
				return nil, err
			}
		}
		clone.Parameters[index] = parameter
		if clone.Parameters[index].Schema, err = route.schema(parameter.Schema, 1); err != nil {
			return nil, fmt.Errorf("parameter %d schema: %w", index, err)
		}
		if clone.Parameters[index].Examples, err = route.examples(parameter.Examples); err != nil {
			return nil, fmt.Errorf("parameter %d examples: %w", index, err)
		}
	}
	if source.RequestBody != nil {
		if err = cloner.retain(uint64(unsafe.Sizeof(APIRequestBody{}))); err != nil {
			return nil, err
		}
		if err = cloner.text(source.RequestBody.Description); err != nil {
			return nil, err
		}
		body := *source.RequestBody
		if body.Content, err = route.mediaTypes(source.RequestBody.Content); err != nil {
			return nil, err
		}
		clone.RequestBody = &body
	}
	if err = route.addItems(len(source.Responses), unsafe.Sizeof(APIResponse{})); err != nil {
		return nil, err
	}
	clone.Responses = make([]APIResponse, len(source.Responses))
	for index, response := range source.Responses {
		if err = cloner.text(response.Description); err != nil {
			return nil, err
		}
		clone.Responses[index] = response
		if err = route.addItems(len(response.Headers), unsafe.Sizeof(APIResponseHeader{})); err != nil {
			return nil, err
		}
		clone.Responses[index].Headers = make([]APIResponseHeader, len(response.Headers))
		for headerIndex, header := range response.Headers {
			for _, value := range []string{header.Name, header.Description} {
				if err = cloner.text(value); err != nil {
					return nil, err
				}
			}
			clone.Responses[index].Headers[headerIndex] = header
			if clone.Responses[index].Headers[headerIndex].Schema, err = route.schema(header.Schema, 1); err != nil {
				return nil, fmt.Errorf("response %d header %d schema: %w", index, headerIndex, err)
			}
			if clone.Responses[index].Headers[headerIndex].Examples, err = route.examples(header.Examples); err != nil {
				return nil, fmt.Errorf("response %d header %d examples: %w", index, headerIndex, err)
			}
		}
		if clone.Responses[index].Content, err = route.mediaTypes(response.Content); err != nil {
			return nil, fmt.Errorf("response %d content: %w", index, err)
		}
	}
	if err = route.addItems(len(source.SecurityRequirements), unsafe.Sizeof(APISecurityRequirement{})); err != nil {
		return nil, err
	}
	clone.SecurityRequirements = make([]APISecurityRequirement, len(source.SecurityRequirements))
	for index, requirement := range source.SecurityRequirements {
		if err = route.addItems(len(requirement.Uses), unsafe.Sizeof(APISecurityUse{})); err != nil {
			return nil, err
		}
		clone.SecurityRequirements[index].Uses = make([]APISecurityUse, len(requirement.Uses))
		for useIndex, use := range requirement.Uses {
			if err = cloner.text(use.SchemeReference); err != nil {
				return nil, err
			}
			clone.SecurityRequirements[index].Uses[useIndex] = use
			if clone.SecurityRequirements[index].Uses[useIndex].Scopes, err = route.textArray(use.Scopes); err != nil {
				return nil, err
			}
		}
	}
	if err = route.addItems(len(source.SchemaDefinitions), unsafe.Sizeof(APISchemaDefinition{})); err != nil {
		return nil, err
	}
	clone.SchemaDefinitions = make([]APISchemaDefinition, len(source.SchemaDefinitions))
	for index, definition := range source.SchemaDefinitions {
		if err = cloner.text(definition.Name); err != nil {
			return nil, err
		}
		clone.SchemaDefinitions[index] = definition
		if clone.SchemaDefinitions[index].Schema, err = route.schema(definition.Schema, 1); err != nil {
			return nil, fmt.Errorf("schema definition %d: %w", index, err)
		}
	}
	return &clone, nil
}

func cloneRouteAPIMetadata(source *RouteAPIMetadata) (*RouteAPIMetadata, error) {
	return (&metadataCloner{}).route(source)
}

func cloneRoutes(source []Route) ([]Route, error) {
	cloner := metadataCloner{}
	clone := make([]Route, len(source))
	for index, route := range source {
		clone[index] = route
		var err error
		if clone[index].APIMetadata, err = cloner.route(route.APIMetadata); err != nil {
			return nil, fmt.Errorf("route %d: %w", index, err)
		}
	}
	return clone, nil
}
