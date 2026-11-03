#include "bridge.h"

_Static_assert(COAKKA_HTTP_HOST_ABI_VERSION == 10U,
               "review the Go projection for the new Core contract");
_Static_assert(COAKKA_HTTP_HOST_API_PARAMETER_PATH == 1U &&
                   COAKKA_HTTP_HOST_API_PARAMETER_QUERY == 2U &&
                   COAKKA_HTTP_HOST_API_PARAMETER_HEADER == 3U &&
                   COAKKA_HTTP_HOST_API_PARAMETER_COOKIE == 4U,
               "review Go APIParameterLocation values");
_Static_assert(COAKKA_HTTP_HOST_API_SCHEMA_BOOLEAN == 1U &&
                   COAKKA_HTTP_HOST_API_SCHEMA_INTEGER == 2U &&
                   COAKKA_HTTP_HOST_API_SCHEMA_NUMBER == 3U &&
                   COAKKA_HTTP_HOST_API_SCHEMA_STRING == 4U &&
                   COAKKA_HTTP_HOST_API_SCHEMA_BYTES == 5U &&
                   COAKKA_HTTP_HOST_API_SCHEMA_OBJECT == 6U &&
                   COAKKA_HTTP_HOST_API_SCHEMA_ARRAY == 7U &&
                   COAKKA_HTTP_HOST_API_SCHEMA_REFERENCE == 8U,
               "review Go APISchemaKind values");
_Static_assert(COAKKA_HTTP_HOST_API_ADDITIONAL_PROPERTIES_UNSPECIFIED == 0U &&
                   COAKKA_HTTP_HOST_API_ADDITIONAL_PROPERTIES_FORBID == 1U &&
                   COAKKA_HTTP_HOST_API_ADDITIONAL_PROPERTIES_ALLOW == 2U,
               "review Go APIAdditionalProperties values");
_Static_assert(COAKKA_HTTP_HOST_API_RESPONSE_STATUS == 1U &&
                   COAKKA_HTTP_HOST_API_RESPONSE_DEFAULT == 2U,
               "review Go APIResponse selector projection");

struct coakka_http_go_api {
  uint32_t abi_version;
};

static coakka_http_go_api_t coakka_http_go_api = {
    COAKKA_HTTP_HOST_ABI_VERSION,
};

coakka_http_host_status_t coakka_http_go_open(coakka_http_go_api_t **out_api) {
  if (out_api == NULL || *out_api != NULL ||
      coakka_http_go_api.abi_version != coakka_http_host_abi_version()) {
    return COAKKA_HTTP_HOST_INVALID_ARGUMENT;
  }
  *out_api = &coakka_http_go_api;
  return COAKKA_HTTP_HOST_OK;
}

void coakka_http_go_close(coakka_http_go_api_t **api) {
  if (api != NULL) {
    *api = NULL;
  }
}

coakka_http_host_capabilities_t
coakka_http_go_features(const coakka_http_go_api_t *api) {
  return api == NULL ? 0U : coakka_http_host_features();
}

#define FORWARD_INIT(go_name, host_name, type)                                 \
  void go_name(const coakka_http_go_api_t *api, type *value) {                 \
    (void)api;                                                                 \
    host_name(value);                                                          \
  }

FORWARD_INIT(coakka_http_go_service_plan_request_init,
             coakka_http_host_service_plan_request_init,
             coakka_http_host_service_plan_request_t)
FORWARD_INIT(coakka_http_go_service_plan_init,
             coakka_http_host_service_plan_init,
             coakka_http_host_service_plan_t)
FORWARD_INIT(coakka_http_go_issue_init, coakka_http_host_issue_init,
             coakka_http_host_issue_t)
void coakka_http_go_configuration_init(const coakka_http_go_api_t *api,
                                       coakka_http_host_configuration_t *value) {
  /* Go's qualified scheduling profile is private construction intent. Core
   * copies it and owns admission/selection; no capability probe or retry here.
   * Immutable C storage avoids retaining a Go pointer across native calls. */
  static const coakka_http_execution_options_t execution = {
      sizeof(coakka_http_execution_options_t),
      COAKKA_HTTP_EXECUTION_PREFER_SUPPORTED, 1U, 0U};
  (void)api;
  coakka_http_host_configuration_init(value);
  if (value != NULL) {
    value->execution = &execution;
  }
}
FORWARD_INIT(coakka_http_go_limits_init, coakka_http_host_limits_init,
             coakka_http_host_limits_t)
FORWARD_INIT(coakka_http_go_route_init, coakka_http_host_route_init,
             coakka_http_host_route_t)
FORWARD_INIT(coakka_http_go_api_example_init,
             coakka_http_host_api_example_init,
             coakka_http_host_api_example_t)
FORWARD_INIT(coakka_http_go_api_schema_property_init,
             coakka_http_host_api_schema_property_init,
             coakka_http_host_api_schema_property_t)
FORWARD_INIT(coakka_http_go_api_schema_init,
             coakka_http_host_api_schema_init,
             coakka_http_host_api_schema_t)
FORWARD_INIT(coakka_http_go_api_schema_definition_init,
             coakka_http_host_api_schema_definition_init,
             coakka_http_host_api_schema_definition_t)
FORWARD_INIT(coakka_http_go_api_parameter_init,
             coakka_http_host_api_parameter_init,
             coakka_http_host_api_parameter_t)
FORWARD_INIT(coakka_http_go_api_media_type_init,
             coakka_http_host_api_media_type_init,
             coakka_http_host_api_media_type_t)
FORWARD_INIT(coakka_http_go_api_request_body_init,
             coakka_http_host_api_request_body_init,
             coakka_http_host_api_request_body_t)
FORWARD_INIT(coakka_http_go_api_header_init,
             coakka_http_host_api_header_init,
             coakka_http_host_api_header_t)
FORWARD_INIT(coakka_http_go_api_response_init,
             coakka_http_host_api_response_init,
             coakka_http_host_api_response_t)
FORWARD_INIT(coakka_http_go_api_security_use_init,
             coakka_http_host_api_security_use_init,
             coakka_http_host_api_security_use_t)
FORWARD_INIT(coakka_http_go_api_security_requirement_init,
             coakka_http_host_api_security_requirement_init,
             coakka_http_host_api_security_requirement_t)
FORWARD_INIT(coakka_http_go_route_api_metadata_init,
             coakka_http_host_route_api_metadata_init,
             coakka_http_host_route_api_metadata_t)
FORWARD_INIT(coakka_http_go_static_mount_init,
             coakka_http_host_static_mount_init,
             coakka_http_host_static_mount_t)
FORWARD_INIT(coakka_http_go_file_authority_init,
             coakka_http_host_file_authority_init,
             coakka_http_host_file_authority_t)
FORWARD_INIT(coakka_http_go_compression_init, coakka_http_host_compression_init,
             coakka_http_host_compression_t)
FORWARD_INIT(coakka_http_go_monitor_options_init,
             coakka_http_host_monitor_options_init,
             coakka_http_host_monitor_options_t)
FORWARD_INIT(coakka_http_go_outbound_endpoint_init,
             coakka_http_host_outbound_endpoint_init,
             coakka_http_host_outbound_endpoint_t)
FORWARD_INIT(coakka_http_go_outbound_target_init,
             coakka_http_host_outbound_target_init,
             coakka_http_host_outbound_target_t)
FORWARD_INIT(coakka_http_go_outbound_trust_init,
             coakka_http_host_outbound_trust_init,
             coakka_http_host_outbound_trust_t)
FORWARD_INIT(coakka_http_go_outbound_identity_init,
             coakka_http_host_outbound_identity_init,
             coakka_http_host_outbound_identity_t)
FORWARD_INIT(coakka_http_go_service_info_init,
             coakka_http_host_service_info_init,
             coakka_http_host_service_info_t)
FORWARD_INIT(coakka_http_go_response_init, coakka_http_host_response_init,
             coakka_http_host_response_t)
FORWARD_INIT(coakka_http_go_file_response_init,
             coakka_http_host_file_response_init,
             coakka_http_host_file_response_t)
FORWARD_INIT(coakka_http_go_failure_init, coakka_http_host_failure_init,
             coakka_http_host_failure_t)
FORWARD_INIT(coakka_http_go_sse_event_init, coakka_http_host_sse_event_init,
             coakka_http_host_sse_event_t)
FORWARD_INIT(coakka_http_go_websocket_event_init,
             coakka_http_host_websocket_event_init,
             coakka_http_host_websocket_event_t)
FORWARD_INIT(coakka_http_go_outbound_request_init,
             coakka_http_host_outbound_request_init,
             coakka_http_host_outbound_request_t)
FORWARD_INIT(coakka_http_go_outbound_event_init,
             coakka_http_host_outbound_event_init,
             coakka_http_host_outbound_event_t)
FORWARD_INIT(coakka_http_go_health_init, coakka_http_host_health_init,
             coakka_http_host_health_t)
FORWARD_INIT(coakka_http_go_route_identity_init,
             coakka_http_host_route_identity_init,
             coakka_http_host_route_identity_t)
FORWARD_INIT(coakka_http_go_rebind_init, coakka_http_host_rebind_request_init,
             coakka_http_host_rebind_request_t)
FORWARD_INIT(coakka_http_go_rebind_outcome_init,
             coakka_http_host_rebind_outcome_init,
             coakka_http_host_rebind_outcome_t)
FORWARD_INIT(coakka_http_go_publication_init,
             coakka_http_host_publication_request_init,
             coakka_http_host_publication_request_t)
FORWARD_INIT(coakka_http_go_publication_outcome_init,
             coakka_http_host_publication_outcome_init,
             coakka_http_host_publication_outcome_t)
FORWARD_INIT(coakka_http_go_monitor_policy_init,
             coakka_http_host_monitor_policy_init,
             coakka_http_host_monitor_policy_t)
FORWARD_INIT(coakka_http_go_monitor_config_init,
             coakka_http_host_monitor_config_init,
             coakka_http_host_monitor_config_t)
FORWARD_INIT(coakka_http_go_monitor_apply_outcome_init,
             coakka_http_host_monitor_apply_outcome_init,
             coakka_http_host_monitor_apply_outcome_t)
FORWARD_INIT(coakka_http_go_monitor_snapshot_init,
             coakka_http_host_monitor_snapshot_init,
             coakka_http_host_monitor_snapshot_t)
FORWARD_INIT(coakka_http_go_monitor_event_init,
             coakka_http_host_monitor_event_init,
             coakka_http_host_monitor_event_t)
FORWARD_INIT(coakka_http_go_monitor_page_init,
             coakka_http_host_monitor_page_init,
             coakka_http_host_monitor_page_t)

#undef FORWARD_INIT

coakka_http_host_status_t coakka_http_go_plan_service(
    const coakka_http_go_api_t *api,
    const coakka_http_host_service_plan_request_t *request,
    coakka_http_host_service_plan_t *out_plan) {
  return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
                     : coakka_http_host_plan_service(request, out_plan);
}

coakka_http_host_status_t coakka_http_go_service_create(
    const coakka_http_go_api_t *api,
    const coakka_http_host_configuration_t *configuration,
    const coakka_http_host_inspection_options_t *inspection,
    const coakka_http_host_route_t *routes, uint32_t route_count,
    coakka_http_host_service_t **out_service, uint32_t *failed_route_index) {
  return api == NULL
             ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
             : coakka_http_host_service_create_with_inspection(
                   configuration, inspection, routes, route_count, out_service,
                   failed_route_index);
}

#define FORWARD_SERVICE_STATUS(go_name, host_name)                             \
  coakka_http_host_status_t go_name(const coakka_http_go_api_t *api,           \
                                    coakka_http_host_service_t *service) {     \
    return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT                     \
                       : host_name(service);                                   \
  }

FORWARD_SERVICE_STATUS(coakka_http_go_service_drain,
                       coakka_http_host_service_begin_drain)
FORWARD_SERVICE_STATUS(coakka_http_go_service_stop,
                       coakka_http_host_service_stop)
FORWARD_SERVICE_STATUS(coakka_http_go_interrupt_requests,
                       coakka_http_host_interrupt_requests)
FORWARD_SERVICE_STATUS(coakka_http_go_interrupt_outbound,
                       coakka_http_host_interrupt_outbound)
FORWARD_SERVICE_STATUS(coakka_http_go_monitor_interrupt,
                       coakka_http_host_monitor_interrupt)

#undef FORWARD_SERVICE_STATUS

coakka_http_host_status_t coakka_http_go_service_wait_drained(
    const coakka_http_go_api_t *api, coakka_http_host_service_t *service,
    uint64_t timeout_ms, coakka_http_host_drain_state_t *state) {
  return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
                    : coakka_http_host_service_wait_drained(service, timeout_ms, state);
}

coakka_http_host_status_t
coakka_http_go_service_start(const coakka_http_go_api_t *api,
                             coakka_http_host_service_t *service,
                             coakka_http_host_issue_t *issue) {
  return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
                     : coakka_http_host_service_start(service, issue);
}

coakka_http_host_status_t coakka_http_go_service_destroy(const coakka_http_go_api_t *api,
                                    coakka_http_host_service_t **service) {
  return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
                     : coakka_http_host_service_destroy(service);
}

coakka_http_host_status_t
coakka_http_go_service_port(const coakka_http_go_api_t *api,
                            const coakka_http_host_service_t *service,
                            uint16_t *out_port) {
  return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
                     : coakka_http_host_service_port(service, out_port);
}

coakka_http_host_status_t
coakka_http_go_service_info(const coakka_http_go_api_t *api,
                            const coakka_http_host_service_t *service,
                            coakka_http_host_service_info_t *out_info) {
  return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
                     : coakka_http_host_service_info(service, out_info);
}

coakka_http_host_status_t coakka_http_go_service_limits(
    const coakka_http_go_api_t *api, const coakka_http_host_service_t *service,
    coakka_http_host_limits_t *out_limits) {
  if (api == NULL || out_limits == NULL)
    return COAKKA_HTTP_HOST_INVALID_ARGUMENT;
  coakka_http_host_limits_init(out_limits);
  return coakka_http_host_service_limits(service, out_limits);
}

coakka_http_host_status_t coakka_http_go_take_request(
    const coakka_http_go_api_t *api, coakka_http_host_service_t *service,
    uint64_t timeout_ms, coakka_http_host_request_event_t *out_event) {
  if (api == NULL || out_event == NULL) {
    return COAKKA_HTTP_HOST_INVALID_ARGUMENT;
  }
  /* Keep Core's initialization and wait semantics, but pay one Go/C crossing.
   * Output has no live lease on entry. Never initialize/reuse leased storage. */
  coakka_http_host_request_event_init(out_event);
  return coakka_http_host_take_request(service, timeout_ms, out_event);
}

coakka_http_host_status_t
coakka_http_go_release_request(const coakka_http_go_api_t *api,
                               coakka_http_host_service_t *service,
                               coakka_http_host_request_event_t *event) {
  return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
                     : coakka_http_host_release_request(service, event);
}

#define FORWARD_REQUEST_VALUE(result_type, go_name, host_name)                 \
  result_type go_name(const coakka_http_go_api_t *api,                         \
                      const coakka_http_host_request_event_t *event) {         \
    (void)api;                                                                 \
    return host_name(event);                                                   \
  }

FORWARD_REQUEST_VALUE(uint32_t, coakka_http_go_request_stream_trailer_count,
                      coakka_http_host_request_stream_trailer_count)

#undef FORWARD_REQUEST_VALUE

coakka_http_go_request_view_result_t coakka_http_go_request_view(
    const coakka_http_go_api_t *api,
    const coakka_http_host_request_event_t *event) {
  coakka_http_go_request_view_result_t result = {0};
  coakka_http_host_request_view_init(&result.view);
  result.status = api == NULL
                      ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
                      : coakka_http_host_request_view(event, &result.view);
  return result;
}

/* Core still owns entry bounds and protocol truth. Bound the private return
 * storage and arithmetic before calling it, then fail without publishing a
 * partial count. Zero initialization also avoids returning uninitialized tail
 * slots across cgo. No shared scratch, retained pointer or allocation. */
#define REQUEST_ENTRY_BATCH(go_name, host_name, batch_type)                    \
  batch_type go_name(const coakka_http_go_api_t *api,                          \
                     const coakka_http_host_request_event_t *event,           \
                     uint32_t start, uint32_t count) {                        \
    (void)api;                                                               \
    batch_type result = {0};                                                 \
    if (count > COAKKA_HTTP_GO_ENTRY_BATCH_CAPACITY ||                         \
        start > UINT32_MAX - count) {                                         \
      return result;                                                         \
    }                                                                        \
    for (uint32_t i = 0U; i < count; ++i) {                                    \
      if (host_name(event, start + i, &result.entries[i]) == 0U) {             \
        return result;                                                       \
      }                                                                      \
    }                                                                        \
    result.count = count;                                                    \
    return result;                                                           \
  }

REQUEST_ENTRY_BATCH(coakka_http_go_request_headers,
                    coakka_http_host_request_header, coakka_http_go_header_batch_t)
REQUEST_ENTRY_BATCH(coakka_http_go_request_trailers,
                    coakka_http_host_request_trailer, coakka_http_go_header_batch_t)
REQUEST_ENTRY_BATCH(coakka_http_go_request_stream_trailers,
                    coakka_http_host_request_stream_trailer, coakka_http_go_header_batch_t)
REQUEST_ENTRY_BATCH(coakka_http_go_request_path_parameters,
                    coakka_http_host_request_path_parameter, coakka_http_go_path_batch_t)
REQUEST_ENTRY_BATCH(coakka_http_go_request_query_parameters,
                    coakka_http_host_request_query_parameter, coakka_http_go_query_batch_t)

#undef REQUEST_ENTRY_BATCH

coakka_http_host_status_t coakka_http_go_request_cancelled(
    const coakka_http_go_api_t *api, coakka_http_host_service_t *service,
    coakka_http_host_exchange_t exchange, uint8_t *out_cancelled) {
  return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
                     : coakka_http_host_request_cancelled(service, exchange,
                                                          out_cancelled);
}

#define FORWARD_EXCHANGE_STATUS(go_name, host_name, value_type)                \
  coakka_http_host_status_t go_name(                                           \
      const coakka_http_go_api_t *api, coakka_http_host_service_t *service,    \
      coakka_http_host_exchange_t exchange, const value_type *value) {         \
    return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT                     \
                       : host_name(service, exchange, value);                  \
  }

FORWARD_EXCHANGE_STATUS(coakka_http_go_respond_file,
                        coakka_http_host_respond_file,
                        coakka_http_host_file_response_t)
FORWARD_EXCHANGE_STATUS(coakka_http_go_fail, coakka_http_host_fail,
                        coakka_http_host_failure_t)
FORWARD_EXCHANGE_STATUS(coakka_http_go_start_response_stream,
                        coakka_http_host_start_response_stream,
                        coakka_http_host_response_t)
FORWARD_EXCHANGE_STATUS(coakka_http_go_write_sse, coakka_http_host_write_sse,
                        coakka_http_host_sse_event_t)

#undef FORWARD_EXCHANGE_STATUS

coakka_http_host_status_t coakka_http_go_respond_buffered(
    const coakka_http_go_api_t *api, coakka_http_host_service_t *service,
    coakka_http_host_exchange_t exchange, uint32_t status_code,
    const coakka_http_host_header_t *headers, uint32_t header_count,
    const uint8_t *body, size_t body_size) {
  if (api == NULL) {
    return COAKKA_HTTP_HOST_INVALID_ARGUMENT;
  }
  /* Build the borrowed view on the C stack: Go passes its pointer-free byte
   * backing array directly, implicitly pinned for the call. Core validates and
   * copies before publication; every return path ends this borrow, including
   * refusal. Never store this view or schedule work with the caller's pointer. */
  coakka_http_host_response_t response;
  coakka_http_host_response_init(&response);
  response.status_code = status_code;
  response.headers = headers;
  response.header_count = header_count;
  response.body.data = body;
  response.body.size = body_size;
  return coakka_http_host_respond(service, exchange, &response);
}

enum { COAKKA_HTTP_GO_MAX_PREPARED_HEADERS = 100 };

static uint32_t coakka_http_go_load_u32_le(const uint8_t *value) {
  return (uint32_t)value[0] | ((uint32_t)value[1] << 8U) |
         ((uint32_t)value[2] << 16U) | ((uint32_t)value[3] << 24U);
}

coakka_http_host_status_t coakka_http_go_respond_prepared(
    const coakka_http_go_api_t *api, coakka_http_host_service_t *service,
    coakka_http_host_exchange_t exchange, uint32_t status_code,
    const uint8_t *wire, size_t wire_size, uint32_t header_count,
    size_t body_offset) {
  if (api == NULL || header_count > COAKKA_HTTP_GO_MAX_PREPARED_HEADERS ||
      body_offset > wire_size || (wire_size != 0U && wire == NULL)) {
    return COAKKA_HTTP_HOST_INVALID_ARGUMENT;
  }
  const size_t table_size = (size_t)header_count * 8U;
  if (table_size > body_offset) {
    return COAKKA_HTTP_HOST_INVALID_ARGUMENT;
  }
  coakka_http_host_header_t headers[COAKKA_HTTP_GO_MAX_PREPARED_HEADERS] = {0};
  size_t length_offset = 0U;
  size_t byte_offset = table_size;
  for (uint32_t index = 0U; index < header_count; ++index) {
    const uint32_t name_size = coakka_http_go_load_u32_le(wire + length_offset);
    const uint32_t value_size =
        coakka_http_go_load_u32_le(wire + length_offset + 4U);
    length_offset += 8U;
    if ((size_t)name_size > body_offset - byte_offset) {
      return COAKKA_HTTP_HOST_INVALID_ARGUMENT;
    }
    if (name_size != 0U) {
      headers[index].name.data = wire + byte_offset;
      headers[index].name.size = (size_t)name_size;
    }
    byte_offset += (size_t)name_size;
    if ((size_t)value_size > body_offset - byte_offset) {
      return COAKKA_HTTP_HOST_INVALID_ARGUMENT;
    }
    if (value_size != 0U) {
      headers[index].value.data = wire + byte_offset;
      headers[index].value.size = (size_t)value_size;
    }
    byte_offset += (size_t)value_size;
  }
  if (byte_offset != body_offset) {
    return COAKKA_HTTP_HOST_INVALID_ARGUMENT;
  }
  coakka_http_host_response_t response;
  coakka_http_host_response_init(&response);
  response.status_code = status_code;
  response.headers = header_count == 0U ? NULL : headers;
  response.header_count = header_count;
  if (wire_size != body_offset) {
    response.body.data = wire + body_offset;
    response.body.size = wire_size - body_offset;
  }
  return coakka_http_host_respond(service, exchange, &response);
}

coakka_http_host_status_t coakka_http_go_write_response_stream(
    const coakka_http_go_api_t *api, coakka_http_host_service_t *service,
    coakka_http_host_exchange_t exchange, coakka_http_host_bytes_t data) {
  return api == NULL
             ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
             : coakka_http_host_write_response_stream(service, exchange, data);
}

coakka_http_host_status_t coakka_http_go_finish_response_stream(
    const coakka_http_go_api_t *api, coakka_http_host_service_t *service,
    coakka_http_host_exchange_t exchange,
    const coakka_http_host_header_t *trailers, uint32_t trailer_count) {
  return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
                     : coakka_http_host_finish_response_stream(
                           service, exchange, trailers, trailer_count);
}

coakka_http_host_status_t coakka_http_go_start_sse(
    const coakka_http_go_api_t *api, coakka_http_host_service_t *service,
    coakka_http_host_exchange_t exchange,
    const coakka_http_host_header_t *headers, uint32_t header_count) {
  return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
                     : coakka_http_host_start_sse(service, exchange, headers,
                                                  header_count);
}

coakka_http_host_status_t
coakka_http_go_accept_websocket(const coakka_http_go_api_t *api,
                                coakka_http_host_service_t *service,
                                coakka_http_host_exchange_t exchange,
                                coakka_http_host_bytes_t selected_subprotocol) {
  return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
                     : coakka_http_host_accept_websocket(service, exchange,
                                                         selected_subprotocol);
}

coakka_http_host_status_t coakka_http_go_take_websocket(
    const coakka_http_go_api_t *api, coakka_http_host_service_t *service,
    uint64_t timeout_ms, coakka_http_host_websocket_event_t *out_event) {
  return api == NULL
             ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
             : coakka_http_host_take_websocket(service, timeout_ms, out_event);
}

coakka_http_host_status_t
coakka_http_go_release_websocket(const coakka_http_go_api_t *api,
                                 coakka_http_host_service_t *service,
                                 coakka_http_host_websocket_event_t *event) {
  return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
                     : coakka_http_host_release_websocket(service, event);
}

coakka_http_host_status_t
coakka_http_go_send_websocket(const coakka_http_go_api_t *api,
                              coakka_http_host_service_t *service,
                              coakka_http_host_websocket_t session,
                              uint32_t kind, coakka_http_host_bytes_t data) {
  return api == NULL
             ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
             : coakka_http_host_send_websocket(service, session, kind, data);
}

coakka_http_host_status_t
coakka_http_go_close_websocket(const coakka_http_go_api_t *api,
                               coakka_http_host_service_t *service,
                               coakka_http_host_websocket_t session,
                               uint32_t code, coakka_http_host_bytes_t reason) {
  return api == NULL
             ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
             : coakka_http_host_close_websocket(service, session, code, reason);
}

coakka_http_host_status_t coakka_http_go_submit_outbound(
    const coakka_http_go_api_t *api, coakka_http_host_service_t *service,
    const coakka_http_host_outbound_request_t *request,
    coakka_http_host_outbound_call_t *out_call) {
  return api == NULL
             ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
             : coakka_http_host_submit_outbound(service, request, out_call);
}

coakka_http_host_status_t
coakka_http_go_cancel_outbound(const coakka_http_go_api_t *api,
                               coakka_http_host_service_t *service,
                               coakka_http_host_outbound_call_t call) {
  return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
                     : coakka_http_host_cancel_outbound(service, call);
}

coakka_http_host_status_t coakka_http_go_take_outbound(
    const coakka_http_go_api_t *api, coakka_http_host_service_t *service,
    uint64_t timeout_ms, coakka_http_host_outbound_event_t *out_event) {
  return api == NULL
             ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
             : coakka_http_host_take_outbound(service, timeout_ms, out_event);
}

uint8_t coakka_http_go_outbound_header(
    const coakka_http_go_api_t *api, coakka_http_host_service_t *service,
    const coakka_http_host_outbound_event_t *event, uint32_t index,
    coakka_http_host_header_t *out_header) {
  return api == NULL ? 0U
                     : coakka_http_host_outbound_header(service, event, index,
                                                        out_header);
}

coakka_http_host_status_t
coakka_http_go_release_outbound(const coakka_http_go_api_t *api,
                                coakka_http_host_service_t *service,
                                coakka_http_host_outbound_event_t *event) {
  return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
                     : coakka_http_host_release_outbound(service, event);
}

coakka_http_host_status_t
coakka_http_go_health(const coakka_http_go_api_t *api,
                      coakka_http_host_service_t *service,
                      coakka_http_host_health_t *out_health) {
  return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
                     : coakka_http_host_service_health(service, out_health);
}

coakka_http_host_status_t coakka_http_go_route_snapshot(
    const coakka_http_go_api_t *api, coakka_http_host_service_t *service,
    coakka_http_host_route_state_t *routes, uint32_t capacity,
    coakka_http_host_route_identity_t *identity) {
  return api == NULL
             ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
             : coakka_http_host_route_snapshot(service, routes, capacity, identity);
}

coakka_http_host_status_t coakka_http_go_probe_liveness(
    const coakka_http_go_api_t *api, coakka_http_host_service_t *service,
    uint64_t timeout_ms, coakka_http_host_health_t *out_health) {
  return api == NULL
             ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
             : coakka_http_host_probe_liveness(service, timeout_ms, out_health);
}

coakka_http_host_status_t
coakka_http_go_rebind(const coakka_http_go_api_t *api,
                      coakka_http_host_service_t *service,
                      const coakka_http_host_rebind_request_t *request,
                      coakka_http_host_rebind_outcome_t *out_outcome) {
  return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
                     : coakka_http_host_rebind(service, request, out_outcome);
}

coakka_http_host_status_t coakka_http_go_publish_routes(
    const coakka_http_go_api_t *api, coakka_http_host_service_t *service,
    const coakka_http_host_publication_request_t *request,
    coakka_http_host_publication_outcome_t *out_outcome) {
  return api == NULL
             ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
             : coakka_http_host_publish_routes(service, request, out_outcome);
}

coakka_http_host_status_t
coakka_http_go_monitor_config(const coakka_http_go_api_t *api,
                              coakka_http_host_service_t *service,
                              coakka_http_host_monitor_config_t *out_config) {
  return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
                     : coakka_http_host_monitor_config(service, out_config);
}

coakka_http_host_status_t coakka_http_go_monitor_apply(
    const coakka_http_go_api_t *api, coakka_http_host_service_t *service,
    uint64_t expected_generation,
    const coakka_http_host_monitor_policy_t *policy,
    coakka_http_host_monitor_apply_outcome_t *out_outcome) {
  return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
                     : coakka_http_host_monitor_apply(
                           service, expected_generation, policy, out_outcome);
}

coakka_http_host_status_t coakka_http_go_monitor_snapshot(
    const coakka_http_go_api_t *api, coakka_http_host_service_t *service,
    coakka_http_host_monitor_snapshot_t *out_snapshot) {
  return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
                     : coakka_http_host_monitor_snapshot(service, out_snapshot);
}

coakka_http_host_status_t coakka_http_go_monitor_read(
    const coakka_http_go_api_t *api, coakka_http_host_service_t *service,
    uint64_t after_sequence, uint32_t maximum_events,
    coakka_http_host_monitor_event_t *events, uint32_t event_capacity,
    coakka_http_host_monitor_page_t *out_page) {
  return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
                     : coakka_http_host_monitor_read(service, after_sequence,
                                                     maximum_events, events,
                                                     event_capacity, out_page);
}

coakka_http_host_status_t
coakka_http_go_monitor_wait(const coakka_http_go_api_t *api,
                            coakka_http_host_service_t *service,
                            uint64_t timeout_ms) {
  return api == NULL ? COAKKA_HTTP_HOST_INVALID_ARGUMENT
                     : coakka_http_host_monitor_wait(service, timeout_ms);
}
