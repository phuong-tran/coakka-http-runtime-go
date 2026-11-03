package coakkahttp

/*
#include "bridge.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

// nativeMetadataProjection owns the call-wide projection-byte budget. The arena
// owns every C address until the synchronous service-create/publication call
// returns; Core copies accepted metadata before that return.
type nativeMetadataProjection struct {
	api           *C.coakka_http_go_api_t
	arena         *nativeArena
	retainedBytes uint64
}

type nativeRouteMetadataProjection struct {
	owner     *nativeMetadataProjection
	items     uint32
	nodes     uint32
	active    map[*APISchema]bool
	projected map[*APISchema]*C.coakka_http_host_api_schema_t
}

func (projection *nativeMetadataProjection) retain(size uint64) error {
	if projection.retainedBytes > maximumRouteMetadataBytes ||
		size > maximumRouteMetadataBytes-projection.retainedBytes {
		return errors.New("route metadata exceeds the Go byte bound")
	}
	projection.retainedBytes += size
	return nil
}

func (projection *nativeMetadataProjection) allocate(size uintptr) (unsafe.Pointer, error) {
	if err := projection.retain(uint64(size)); err != nil {
		return nil, err
	}
	return projection.arena.allocate(size)
}

func (projection *nativeMetadataProjection) text(value string) (C.coakka_http_host_bytes_t, error) {
	if len(value) > maximumMetadataTextBytes {
		return C.coakka_http_host_bytes_t{}, errors.New("route metadata text exceeds the Go bound")
	}
	if err := projection.retain(uint64(len(value))); err != nil {
		return C.coakka_http_host_bytes_t{}, err
	}
	return projection.arena.text(value)
}

func (projection *nativeMetadataProjection) bytes(value []byte) (C.coakka_http_host_bytes_t, error) {
	if len(value) > maximumRouteMetadataBytes {
		return C.coakka_http_host_bytes_t{}, errors.New("route metadata example exceeds the Go bound")
	}
	if err := projection.retain(uint64(len(value))); err != nil {
		return C.coakka_http_host_bytes_t{}, err
	}
	return projection.arena.bytes(value)
}

func (projection *nativeRouteMetadataProjection) addItems(count int) error {
	if count < 0 || uint64(count) > uint64(maximumMetadataItems-projection.items) {
		return errors.New("route metadata exceeds the Go item bound")
	}
	projection.items += uint32(count)
	return nil
}

func (projection *nativeRouteMetadataProjection) textArray(values []string) (*C.coakka_http_host_bytes_t, C.uint32_t, error) {
	if err := projection.addItems(len(values)); err != nil {
		return nil, 0, err
	}
	if len(values) == 0 {
		return nil, 0, nil
	}
	memory, err := projection.owner.allocate(uintptr(len(values)) * unsafe.Sizeof(C.coakka_http_host_bytes_t{}))
	if err != nil {
		return nil, 0, err
	}
	targets := unsafe.Slice((*C.coakka_http_host_bytes_t)(memory), len(values))
	for index, value := range values {
		if targets[index], err = projection.owner.text(value); err != nil {
			return nil, 0, fmt.Errorf("text %d: %w", index, err)
		}
	}
	return (*C.coakka_http_host_bytes_t)(memory), C.uint32_t(len(values)), nil
}

func (projection *nativeRouteMetadataProjection) examples(values []APIExample) (*C.coakka_http_host_api_example_t, C.uint32_t, error) {
	if err := projection.addItems(len(values)); err != nil {
		return nil, 0, err
	}
	if len(values) == 0 {
		return nil, 0, nil
	}
	memory, err := projection.owner.allocate(uintptr(len(values)) * unsafe.Sizeof(C.coakka_http_host_api_example_t{}))
	if err != nil {
		return nil, 0, err
	}
	targets := unsafe.Slice((*C.coakka_http_host_api_example_t)(memory), len(values))
	for index, value := range values {
		C.coakka_http_go_api_example_init(projection.owner.api, &targets[index])
		if targets[index].name, err = projection.owner.text(value.Name); err != nil {
			return nil, 0, fmt.Errorf("example %d name: %w", index, err)
		}
		if targets[index].summary, err = projection.owner.text(value.Summary); err != nil {
			return nil, 0, fmt.Errorf("example %d summary: %w", index, err)
		}
		if targets[index].value, err = projection.owner.bytes(value.Value); err != nil {
			return nil, 0, fmt.Errorf("example %d value: %w", index, err)
		}
	}
	return (*C.coakka_http_host_api_example_t)(memory), C.uint32_t(len(values)), nil
}

func (projection *nativeRouteMetadataProjection) schema(source *APISchema, depth uint32) (*C.coakka_http_host_api_schema_t, error) {
	if source == nil {
		return nil, nil
	}
	if depth > maximumMetadataSchemaDepth {
		return nil, errors.New("route metadata schema exceeds the Go depth bound")
	}
	if projection.active[source] {
		return nil, errors.New("route metadata schema must not contain a cycle")
	}
	if existing := projection.projected[source]; existing != nil {
		return existing, nil
	}
	if projection.nodes == maximumMetadataSchemaNodes {
		return nil, errors.New("route metadata schema exceeds the Go node bound")
	}
	projection.nodes++
	memory, err := projection.owner.allocate(unsafe.Sizeof(C.coakka_http_host_api_schema_t{}))
	if err != nil {
		return nil, err
	}
	target := (*C.coakka_http_host_api_schema_t)(memory)
	C.coakka_http_go_api_schema_init(projection.owner.api, target)
	projection.projected[source] = target
	projection.active[source] = true
	defer delete(projection.active, source)
	target.kind = C.coakka_http_host_api_schema_kind_t(source.Kind)
	target.additional_properties = C.coakka_http_host_api_additional_properties_t(source.AdditionalProperties)
	if target.format, err = projection.owner.text(source.Format); err != nil {
		return nil, err
	}
	if target.reference, err = projection.owner.text(source.Reference); err != nil {
		return nil, err
	}
	if target.enum_values, target.enum_value_count, err = projection.textArray(source.EnumValues); err != nil {
		return nil, err
	}
	if target.items, err = projection.schema(source.Items, depth+1); err != nil {
		return nil, err
	}
	if target.properties, target.property_count, err = projection.schemaProperties(source.Properties, depth+1); err != nil {
		return nil, err
	}
	if source.HasNullable {
		target.has_nullable = 1
		target.nullable = boolByte(source.Nullable)
	}
	return target, nil
}

func (projection *nativeRouteMetadataProjection) schemaProperties(values []APISchemaProperty, depth uint32) (*C.coakka_http_host_api_schema_property_t, C.uint32_t, error) {
	if err := projection.addItems(len(values)); err != nil {
		return nil, 0, err
	}
	if len(values) == 0 {
		return nil, 0, nil
	}
	memory, err := projection.owner.allocate(uintptr(len(values)) * unsafe.Sizeof(C.coakka_http_host_api_schema_property_t{}))
	if err != nil {
		return nil, 0, err
	}
	targets := unsafe.Slice((*C.coakka_http_host_api_schema_property_t)(memory), len(values))
	for index, value := range values {
		C.coakka_http_go_api_schema_property_init(projection.owner.api, &targets[index])
		targets[index].required = boolByte(value.Required)
		if targets[index].name, err = projection.owner.text(value.Name); err != nil {
			return nil, 0, err
		}
		if targets[index].description, err = projection.owner.text(value.Description); err != nil {
			return nil, 0, err
		}
		if targets[index].schema, err = projection.schema(value.Schema, depth); err != nil {
			return nil, 0, fmt.Errorf("property %d: %w", index, err)
		}
	}
	return (*C.coakka_http_host_api_schema_property_t)(memory), C.uint32_t(len(values)), nil
}

func (projection *nativeRouteMetadataProjection) mediaTypes(values []APIMediaType) (*C.coakka_http_host_api_media_type_t, C.uint32_t, error) {
	if err := projection.addItems(len(values)); err != nil {
		return nil, 0, err
	}
	if len(values) == 0 {
		return nil, 0, nil
	}
	memory, err := projection.owner.allocate(uintptr(len(values)) * unsafe.Sizeof(C.coakka_http_host_api_media_type_t{}))
	if err != nil {
		return nil, 0, err
	}
	targets := unsafe.Slice((*C.coakka_http_host_api_media_type_t)(memory), len(values))
	for index, value := range values {
		C.coakka_http_go_api_media_type_init(projection.owner.api, &targets[index])
		if targets[index].media_type, err = projection.owner.text(value.MediaType); err != nil {
			return nil, 0, err
		}
		if targets[index].schema, err = projection.schema(value.Schema, 1); err != nil {
			return nil, 0, fmt.Errorf("media %d schema: %w", index, err)
		}
		if targets[index].examples, targets[index].example_count, err = projection.examples(value.Examples); err != nil {
			return nil, 0, fmt.Errorf("media %d examples: %w", index, err)
		}
	}
	return (*C.coakka_http_host_api_media_type_t)(memory), C.uint32_t(len(values)), nil
}

func (projection *nativeRouteMetadataProjection) parameters(values []APIParameter) (*C.coakka_http_host_api_parameter_t, C.uint32_t, error) {
	if err := projection.addItems(len(values)); err != nil {
		return nil, 0, err
	}
	if len(values) == 0 {
		return nil, 0, nil
	}
	memory, err := projection.owner.allocate(uintptr(len(values)) * unsafe.Sizeof(C.coakka_http_host_api_parameter_t{}))
	if err != nil {
		return nil, 0, err
	}
	targets := unsafe.Slice((*C.coakka_http_host_api_parameter_t)(memory), len(values))
	for index, value := range values {
		C.coakka_http_go_api_parameter_init(projection.owner.api, &targets[index])
		targets[index].location = C.coakka_http_host_api_parameter_location_t(value.Location)
		targets[index].required = boolByte(value.Required)
		if targets[index].name, err = projection.owner.text(value.Name); err != nil {
			return nil, 0, err
		}
		if targets[index].description, err = projection.owner.text(value.Description); err != nil {
			return nil, 0, err
		}
		if targets[index].schema, err = projection.schema(value.Schema, 1); err != nil {
			return nil, 0, fmt.Errorf("parameter %d schema: %w", index, err)
		}
		if targets[index].examples, targets[index].example_count, err = projection.examples(value.Examples); err != nil {
			return nil, 0, fmt.Errorf("parameter %d examples: %w", index, err)
		}
	}
	return (*C.coakka_http_host_api_parameter_t)(memory), C.uint32_t(len(values)), nil
}

func (projection *nativeRouteMetadataProjection) requestBody(source *APIRequestBody) (*C.coakka_http_host_api_request_body_t, error) {
	if source == nil {
		return nil, nil
	}
	memory, err := projection.owner.allocate(unsafe.Sizeof(C.coakka_http_host_api_request_body_t{}))
	if err != nil {
		return nil, err
	}
	target := (*C.coakka_http_host_api_request_body_t)(memory)
	C.coakka_http_go_api_request_body_init(projection.owner.api, target)
	target.required = boolByte(source.Required)
	if target.description, err = projection.owner.text(source.Description); err != nil {
		return nil, err
	}
	if target.content, target.content_count, err = projection.mediaTypes(source.Content); err != nil {
		return nil, err
	}
	return target, nil
}

func (projection *nativeRouteMetadataProjection) responseHeaders(values []APIResponseHeader) (*C.coakka_http_host_api_header_t, C.uint32_t, error) {
	if err := projection.addItems(len(values)); err != nil {
		return nil, 0, err
	}
	if len(values) == 0 {
		return nil, 0, nil
	}
	memory, err := projection.owner.allocate(uintptr(len(values)) * unsafe.Sizeof(C.coakka_http_host_api_header_t{}))
	if err != nil {
		return nil, 0, err
	}
	targets := unsafe.Slice((*C.coakka_http_host_api_header_t)(memory), len(values))
	for index, value := range values {
		C.coakka_http_go_api_header_init(projection.owner.api, &targets[index])
		if targets[index].name, err = projection.owner.text(value.Name); err != nil {
			return nil, 0, err
		}
		if targets[index].description, err = projection.owner.text(value.Description); err != nil {
			return nil, 0, err
		}
		if targets[index].schema, err = projection.schema(value.Schema, 1); err != nil {
			return nil, 0, fmt.Errorf("response header %d schema: %w", index, err)
		}
		if targets[index].examples, targets[index].example_count, err = projection.examples(value.Examples); err != nil {
			return nil, 0, fmt.Errorf("response header %d examples: %w", index, err)
		}
	}
	return (*C.coakka_http_host_api_header_t)(memory), C.uint32_t(len(values)), nil
}

func (projection *nativeRouteMetadataProjection) responses(values []APIResponse) (*C.coakka_http_host_api_response_t, C.uint32_t, error) {
	if err := projection.addItems(len(values)); err != nil {
		return nil, 0, err
	}
	if len(values) == 0 {
		return nil, 0, nil
	}
	memory, err := projection.owner.allocate(uintptr(len(values)) * unsafe.Sizeof(C.coakka_http_host_api_response_t{}))
	if err != nil {
		return nil, 0, err
	}
	targets := unsafe.Slice((*C.coakka_http_host_api_response_t)(memory), len(values))
	for index, value := range values {
		C.coakka_http_go_api_response_init(projection.owner.api, &targets[index])
		if value.HasStatusCode {
			targets[index].selector = C.COAKKA_HTTP_HOST_API_RESPONSE_STATUS
			targets[index].status_code = C.uint32_t(value.StatusCode)
		} else {
			targets[index].selector = C.COAKKA_HTTP_HOST_API_RESPONSE_DEFAULT
		}
		if targets[index].description, err = projection.owner.text(value.Description); err != nil {
			return nil, 0, err
		}
		if targets[index].headers, targets[index].header_count, err = projection.responseHeaders(value.Headers); err != nil {
			return nil, 0, fmt.Errorf("response %d headers: %w", index, err)
		}
		if targets[index].content, targets[index].content_count, err = projection.mediaTypes(value.Content); err != nil {
			return nil, 0, fmt.Errorf("response %d content: %w", index, err)
		}
	}
	return (*C.coakka_http_host_api_response_t)(memory), C.uint32_t(len(values)), nil
}

func (projection *nativeRouteMetadataProjection) securityUses(values []APISecurityUse) (*C.coakka_http_host_api_security_use_t, C.uint32_t, error) {
	if err := projection.addItems(len(values)); err != nil {
		return nil, 0, err
	}
	if len(values) == 0 {
		return nil, 0, nil
	}
	memory, err := projection.owner.allocate(uintptr(len(values)) * unsafe.Sizeof(C.coakka_http_host_api_security_use_t{}))
	if err != nil {
		return nil, 0, err
	}
	targets := unsafe.Slice((*C.coakka_http_host_api_security_use_t)(memory), len(values))
	for index, value := range values {
		C.coakka_http_go_api_security_use_init(projection.owner.api, &targets[index])
		if targets[index].scheme_reference, err = projection.owner.text(value.SchemeReference); err != nil {
			return nil, 0, err
		}
		if targets[index].scopes, targets[index].scope_count, err = projection.textArray(value.Scopes); err != nil {
			return nil, 0, fmt.Errorf("security use %d scopes: %w", index, err)
		}
	}
	return (*C.coakka_http_host_api_security_use_t)(memory), C.uint32_t(len(values)), nil
}

func (projection *nativeRouteMetadataProjection) securityRequirements(values []APISecurityRequirement) (*C.coakka_http_host_api_security_requirement_t, C.uint32_t, error) {
	if err := projection.addItems(len(values)); err != nil {
		return nil, 0, err
	}
	if len(values) == 0 {
		return nil, 0, nil
	}
	memory, err := projection.owner.allocate(uintptr(len(values)) * unsafe.Sizeof(C.coakka_http_host_api_security_requirement_t{}))
	if err != nil {
		return nil, 0, err
	}
	targets := unsafe.Slice((*C.coakka_http_host_api_security_requirement_t)(memory), len(values))
	for index, value := range values {
		C.coakka_http_go_api_security_requirement_init(projection.owner.api, &targets[index])
		if targets[index].uses, targets[index].use_count, err = projection.securityUses(value.Uses); err != nil {
			return nil, 0, fmt.Errorf("security requirement %d: %w", index, err)
		}
	}
	return (*C.coakka_http_host_api_security_requirement_t)(memory), C.uint32_t(len(values)), nil
}

func (projection *nativeRouteMetadataProjection) schemaDefinitions(values []APISchemaDefinition) (*C.coakka_http_host_api_schema_definition_t, C.uint32_t, error) {
	if err := projection.addItems(len(values)); err != nil {
		return nil, 0, err
	}
	if len(values) == 0 {
		return nil, 0, nil
	}
	memory, err := projection.owner.allocate(uintptr(len(values)) * unsafe.Sizeof(C.coakka_http_host_api_schema_definition_t{}))
	if err != nil {
		return nil, 0, err
	}
	targets := unsafe.Slice((*C.coakka_http_host_api_schema_definition_t)(memory), len(values))
	for index, value := range values {
		C.coakka_http_go_api_schema_definition_init(projection.owner.api, &targets[index])
		if targets[index].name, err = projection.owner.text(value.Name); err != nil {
			return nil, 0, err
		}
		if targets[index].schema, err = projection.schema(value.Schema, 1); err != nil {
			return nil, 0, fmt.Errorf("schema definition %d: %w", index, err)
		}
	}
	return (*C.coakka_http_host_api_schema_definition_t)(memory), C.uint32_t(len(values)), nil
}

func (projection *nativeMetadataProjection) route(source *RouteAPIMetadata) (*C.coakka_http_host_route_api_metadata_t, error) {
	if source == nil {
		return nil, nil
	}
	route := nativeRouteMetadataProjection{
		owner: projection, active: make(map[*APISchema]bool),
		projected: make(map[*APISchema]*C.coakka_http_host_api_schema_t),
	}
	memory, err := projection.allocate(unsafe.Sizeof(C.coakka_http_host_route_api_metadata_t{}))
	if err != nil {
		return nil, err
	}
	target := (*C.coakka_http_host_route_api_metadata_t)(memory)
	C.coakka_http_go_route_api_metadata_init(projection.api, target)
	target.deprecated = boolByte(source.Deprecated)
	if target.operation_id, err = projection.text(source.OperationID); err != nil {
		return nil, err
	}
	if target.summary, err = projection.text(source.Summary); err != nil {
		return nil, err
	}
	if target.description, err = projection.text(source.Description); err != nil {
		return nil, err
	}
	if target.tags, target.tag_count, err = route.textArray(source.Tags); err != nil {
		return nil, err
	}
	if target.parameters, target.parameter_count, err = route.parameters(source.Parameters); err != nil {
		return nil, err
	}
	if target.request_body, err = route.requestBody(source.RequestBody); err != nil {
		return nil, err
	}
	if target.responses, target.response_count, err = route.responses(source.Responses); err != nil {
		return nil, err
	}
	if target.security_requirements, target.security_requirement_count, err = route.securityRequirements(source.SecurityRequirements); err != nil {
		return nil, err
	}
	if target.schema_definitions, target.schema_definition_count, err = route.schemaDefinitions(source.SchemaDefinitions); err != nil {
		return nil, err
	}
	return target, nil
}
