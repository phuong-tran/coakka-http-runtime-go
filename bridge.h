#ifndef COAKKA_HTTP_GO_BRIDGE_H
#define COAKKA_HTTP_GO_BRIDGE_H

#include <coakka/http/host.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct coakka_http_go_api coakka_http_go_api_t;

/* Cold coherent route observation; Core owns capacity, identity and refusal. */
void coakka_http_go_route_identity_init(
    const coakka_http_go_api_t *, coakka_http_host_route_identity_t *);
coakka_http_host_status_t coakka_http_go_route_snapshot(
    const coakka_http_go_api_t *, coakka_http_host_service_t *,
    coakka_http_host_route_state_t *, uint32_t,
    coakka_http_host_route_identity_t *);

/* The bridge links directly to the installed host library. This lightweight
 * binding token only verifies ABI compatibility; all transport and backend
 * decisions remain native-owned. */
coakka_http_host_status_t coakka_http_go_open(coakka_http_go_api_t **);
void coakka_http_go_close(coakka_http_go_api_t **);
coakka_http_host_capabilities_t
coakka_http_go_features(const coakka_http_go_api_t *);

void coakka_http_go_service_plan_request_init(
    const coakka_http_go_api_t *, coakka_http_host_service_plan_request_t *);
void coakka_http_go_service_plan_init(const coakka_http_go_api_t *,
                                      coakka_http_host_service_plan_t *);
void coakka_http_go_issue_init(const coakka_http_go_api_t *,
                               coakka_http_host_issue_t *);
coakka_http_host_status_t
coakka_http_go_plan_service(const coakka_http_go_api_t *,
                            const coakka_http_host_service_plan_request_t *,
                            coakka_http_host_service_plan_t *);

void coakka_http_go_configuration_init(const coakka_http_go_api_t *,
                                       coakka_http_host_configuration_t *);
void coakka_http_go_limits_init(const coakka_http_go_api_t *,
                                coakka_http_host_limits_t *);
void coakka_http_go_route_init(const coakka_http_go_api_t *,
                               coakka_http_host_route_t *);
void coakka_http_go_api_example_init(
    const coakka_http_go_api_t *, coakka_http_host_api_example_t *);
void coakka_http_go_api_schema_property_init(
    const coakka_http_go_api_t *, coakka_http_host_api_schema_property_t *);
void coakka_http_go_api_schema_init(
    const coakka_http_go_api_t *, coakka_http_host_api_schema_t *);
void coakka_http_go_api_schema_definition_init(
    const coakka_http_go_api_t *, coakka_http_host_api_schema_definition_t *);
void coakka_http_go_api_parameter_init(
    const coakka_http_go_api_t *, coakka_http_host_api_parameter_t *);
void coakka_http_go_api_media_type_init(
    const coakka_http_go_api_t *, coakka_http_host_api_media_type_t *);
void coakka_http_go_api_request_body_init(
    const coakka_http_go_api_t *, coakka_http_host_api_request_body_t *);
void coakka_http_go_api_header_init(
    const coakka_http_go_api_t *, coakka_http_host_api_header_t *);
void coakka_http_go_api_response_init(
    const coakka_http_go_api_t *, coakka_http_host_api_response_t *);
void coakka_http_go_api_security_use_init(
    const coakka_http_go_api_t *, coakka_http_host_api_security_use_t *);
void coakka_http_go_api_security_requirement_init(
    const coakka_http_go_api_t *, coakka_http_host_api_security_requirement_t *);
void coakka_http_go_route_api_metadata_init(
    const coakka_http_go_api_t *, coakka_http_host_route_api_metadata_t *);
void coakka_http_go_static_mount_init(const coakka_http_go_api_t *,
                                      coakka_http_host_static_mount_t *);
void coakka_http_go_file_authority_init(const coakka_http_go_api_t *,
                                        coakka_http_host_file_authority_t *);
void coakka_http_go_compression_init(const coakka_http_go_api_t *,
                                     coakka_http_host_compression_t *);
void coakka_http_go_monitor_options_init(const coakka_http_go_api_t *,
                                         coakka_http_host_monitor_options_t *);
void coakka_http_go_outbound_endpoint_init(
    const coakka_http_go_api_t *, coakka_http_host_outbound_endpoint_t *);
void coakka_http_go_outbound_target_init(const coakka_http_go_api_t *,
                                         coakka_http_host_outbound_target_t *);
void coakka_http_go_outbound_trust_init(const coakka_http_go_api_t *,
                                        coakka_http_host_outbound_trust_t *);
void coakka_http_go_outbound_identity_init(
    const coakka_http_go_api_t *, coakka_http_host_outbound_identity_t *);

coakka_http_host_status_t
coakka_http_go_service_create(const coakka_http_go_api_t *,
                              const coakka_http_host_configuration_t *,
                              const coakka_http_host_inspection_options_t *,
                              const coakka_http_host_route_t *, uint32_t,
                              coakka_http_host_service_t **, uint32_t *);
coakka_http_host_status_t
coakka_http_go_service_start(const coakka_http_go_api_t *,
                             coakka_http_host_service_t *,
                             coakka_http_host_issue_t *);
coakka_http_host_status_t
coakka_http_go_service_port(const coakka_http_go_api_t *,
                            const coakka_http_host_service_t *, uint16_t *);
coakka_http_host_status_t
coakka_http_go_service_info(const coakka_http_go_api_t *,
                            const coakka_http_host_service_t *,
                            coakka_http_host_service_info_t *);
void coakka_http_go_service_info_init(const coakka_http_go_api_t *,
                                      coakka_http_host_service_info_t *);
/* Cold, pointer-free effective limits copied from Core; no connector defaults. */
coakka_http_host_status_t coakka_http_go_service_limits(
    const coakka_http_go_api_t *, const coakka_http_host_service_t *,
    coakka_http_host_limits_t *);
coakka_http_host_status_t
coakka_http_go_service_drain(const coakka_http_go_api_t *,
                             coakka_http_host_service_t *);
/* Core-issued completion/cause; never inferred from an empty reader queue. */
coakka_http_host_status_t coakka_http_go_service_wait_drained(
    const coakka_http_go_api_t *, coakka_http_host_service_t *, uint64_t,
    coakka_http_host_drain_state_t *);
coakka_http_host_status_t
coakka_http_go_service_stop(const coakka_http_go_api_t *,
                            coakka_http_host_service_t *);
coakka_http_host_status_t coakka_http_go_service_destroy(const coakka_http_go_api_t *,
                                    coakka_http_host_service_t **);
coakka_http_host_status_t
coakka_http_go_interrupt_requests(const coakka_http_go_api_t *,
                                  coakka_http_host_service_t *);

/* Initializes unleased output through Core, then attempts one bounded take.
 * Success transfers one lease to the caller; failures transfer none. The
 * caller must release successful output before reusing this storage. */
coakka_http_host_status_t
coakka_http_go_take_request(const coakka_http_go_api_t *,
                            coakka_http_host_service_t *, uint64_t,
                            coakka_http_host_request_event_t *);
coakka_http_host_status_t
coakka_http_go_release_request(const coakka_http_go_api_t *,
                               coakka_http_host_service_t *,
                               coakka_http_host_request_event_t *);

/* One private by-value result projects Core's authoritative immutable request
 * view without a Go output pointer. Every byte range borrows the live event
 * lease; Go copies it before release. Core owns validation and view layout. */
typedef struct coakka_http_go_request_view_result {
  coakka_http_host_request_view_t view;
  coakka_http_host_status_t status;
} coakka_http_go_request_view_result_t;

coakka_http_go_request_view_result_t coakka_http_go_request_view(
    const coakka_http_go_api_t *, const coakka_http_host_request_event_t *);
/* Private projection granularity, unrelated to Core notification batches.
 * Fixed by-value results need no heap scratch or Go output pointer passed to C.
 * Only count entries are usable; any getter failure returns count zero. All
 * byte views remain borrowed from the live event, including across chunks. */
enum { COAKKA_HTTP_GO_ENTRY_BATCH_CAPACITY = 8 };
typedef struct coakka_http_go_header_batch {
  coakka_http_host_header_t entries[COAKKA_HTTP_GO_ENTRY_BATCH_CAPACITY];
  uint32_t count;
} coakka_http_go_header_batch_t;
typedef struct coakka_http_go_path_batch {
  coakka_http_host_path_parameter_t entries[COAKKA_HTTP_GO_ENTRY_BATCH_CAPACITY];
  uint32_t count;
} coakka_http_go_path_batch_t;
typedef struct coakka_http_go_query_batch {
  coakka_http_host_query_parameter_t entries[COAKKA_HTTP_GO_ENTRY_BATCH_CAPACITY];
  uint32_t count;
} coakka_http_go_query_batch_t;

coakka_http_go_header_batch_t coakka_http_go_request_headers(
    const coakka_http_go_api_t *, const coakka_http_host_request_event_t *,
    uint32_t start, uint32_t count);
coakka_http_go_header_batch_t coakka_http_go_request_trailers(
    const coakka_http_go_api_t *, const coakka_http_host_request_event_t *,
    uint32_t start, uint32_t count);
coakka_http_go_header_batch_t coakka_http_go_request_stream_trailers(
    const coakka_http_go_api_t *, const coakka_http_host_request_event_t *,
    uint32_t start, uint32_t count);
coakka_http_go_path_batch_t coakka_http_go_request_path_parameters(
    const coakka_http_go_api_t *, const coakka_http_host_request_event_t *,
    uint32_t start, uint32_t count);
coakka_http_go_query_batch_t coakka_http_go_request_query_parameters(
    const coakka_http_go_api_t *, const coakka_http_host_request_event_t *,
    uint32_t start, uint32_t count);
uint32_t coakka_http_go_request_stream_trailer_count(
    const coakka_http_go_api_t *, const coakka_http_host_request_event_t *);
coakka_http_host_status_t
coakka_http_go_request_cancelled(const coakka_http_go_api_t *,
                                 coakka_http_host_service_t *,
                                 coakka_http_host_exchange_t, uint8_t *);

void coakka_http_go_response_init(const coakka_http_go_api_t *,
                                  coakka_http_host_response_t *);
void coakka_http_go_file_response_init(const coakka_http_go_api_t *,
                                       coakka_http_host_file_response_t *);
void coakka_http_go_failure_init(const coakka_http_go_api_t *,
                                 coakka_http_host_failure_t *);
void coakka_http_go_sse_event_init(const coakka_http_go_api_t *,
                                   coakka_http_host_sse_event_t *);
/* Borrow body only for this synchronous call. Core copies accepted bytes before
 * returning; neither this bridge nor Core retains the caller's body pointer. */
coakka_http_host_status_t coakka_http_go_respond_buffered(
    const coakka_http_go_api_t *, coakka_http_host_service_t *,
    coakka_http_host_exchange_t, uint32_t, const coakka_http_host_header_t *,
    uint32_t, const uint8_t *, size_t);
/* Go-owned pointer-free wire bytes are borrowed only for this synchronous
 * call. The bridge validates its private extents, builds bounded stack views,
 * and Core validates/copies HTTP values before return. */
coakka_http_host_status_t coakka_http_go_respond_prepared(
    const coakka_http_go_api_t *, coakka_http_host_service_t *,
    coakka_http_host_exchange_t, uint32_t, const uint8_t *, size_t, uint32_t,
    size_t);
coakka_http_host_status_t coakka_http_go_respond_file(
    const coakka_http_go_api_t *, coakka_http_host_service_t *,
    coakka_http_host_exchange_t, const coakka_http_host_file_response_t *);
coakka_http_host_status_t
coakka_http_go_fail(const coakka_http_go_api_t *, coakka_http_host_service_t *,
                    coakka_http_host_exchange_t,
                    const coakka_http_host_failure_t *);
coakka_http_host_status_t coakka_http_go_start_response_stream(
    const coakka_http_go_api_t *, coakka_http_host_service_t *,
    coakka_http_host_exchange_t, const coakka_http_host_response_t *);
coakka_http_host_status_t coakka_http_go_write_response_stream(
    const coakka_http_go_api_t *, coakka_http_host_service_t *,
    coakka_http_host_exchange_t, coakka_http_host_bytes_t);
coakka_http_host_status_t coakka_http_go_finish_response_stream(
    const coakka_http_go_api_t *, coakka_http_host_service_t *,
    coakka_http_host_exchange_t, const coakka_http_host_header_t *, uint32_t);
coakka_http_host_status_t coakka_http_go_start_sse(
    const coakka_http_go_api_t *, coakka_http_host_service_t *,
    coakka_http_host_exchange_t, const coakka_http_host_header_t *, uint32_t);
coakka_http_host_status_t coakka_http_go_write_sse(
    const coakka_http_go_api_t *, coakka_http_host_service_t *,
    coakka_http_host_exchange_t, const coakka_http_host_sse_event_t *);

void coakka_http_go_websocket_event_init(const coakka_http_go_api_t *,
                                         coakka_http_host_websocket_event_t *);
coakka_http_host_status_t coakka_http_go_accept_websocket(
    const coakka_http_go_api_t *, coakka_http_host_service_t *,
    coakka_http_host_exchange_t, coakka_http_host_bytes_t);
coakka_http_host_status_t
coakka_http_go_take_websocket(const coakka_http_go_api_t *,
                              coakka_http_host_service_t *, uint64_t,
                              coakka_http_host_websocket_event_t *);
coakka_http_host_status_t
coakka_http_go_release_websocket(const coakka_http_go_api_t *,
                                 coakka_http_host_service_t *,
                                 coakka_http_host_websocket_event_t *);
coakka_http_host_status_t coakka_http_go_send_websocket(
    const coakka_http_go_api_t *, coakka_http_host_service_t *,
    coakka_http_host_websocket_t, uint32_t, coakka_http_host_bytes_t);
coakka_http_host_status_t coakka_http_go_close_websocket(
    const coakka_http_go_api_t *, coakka_http_host_service_t *,
    coakka_http_host_websocket_t, uint32_t, coakka_http_host_bytes_t);

void coakka_http_go_outbound_request_init(
    const coakka_http_go_api_t *, coakka_http_host_outbound_request_t *);
void coakka_http_go_outbound_event_init(const coakka_http_go_api_t *,
                                        coakka_http_host_outbound_event_t *);
coakka_http_host_status_t
coakka_http_go_submit_outbound(const coakka_http_go_api_t *,
                               coakka_http_host_service_t *,
                               const coakka_http_host_outbound_request_t *,
                               coakka_http_host_outbound_call_t *);
coakka_http_host_status_t
coakka_http_go_cancel_outbound(const coakka_http_go_api_t *,
                               coakka_http_host_service_t *,
                               coakka_http_host_outbound_call_t);
coakka_http_host_status_t
coakka_http_go_interrupt_outbound(const coakka_http_go_api_t *,
                                  coakka_http_host_service_t *);
coakka_http_host_status_t
coakka_http_go_take_outbound(const coakka_http_go_api_t *,
                             coakka_http_host_service_t *, uint64_t,
                             coakka_http_host_outbound_event_t *);
uint8_t
coakka_http_go_outbound_header(const coakka_http_go_api_t *,
                               coakka_http_host_service_t *,
                               const coakka_http_host_outbound_event_t *,
                               uint32_t, coakka_http_host_header_t *);
coakka_http_host_status_t
coakka_http_go_release_outbound(const coakka_http_go_api_t *,
                                coakka_http_host_service_t *,
                                coakka_http_host_outbound_event_t *);

void coakka_http_go_health_init(const coakka_http_go_api_t *,
                                coakka_http_host_health_t *);
coakka_http_host_status_t coakka_http_go_health(const coakka_http_go_api_t *,
                                                coakka_http_host_service_t *,
                                                coakka_http_host_health_t *);
coakka_http_host_status_t
coakka_http_go_probe_liveness(const coakka_http_go_api_t *,
                              coakka_http_host_service_t *, uint64_t,
                              coakka_http_host_health_t *);

void coakka_http_go_rebind_init(const coakka_http_go_api_t *,
                                coakka_http_host_rebind_request_t *);
void coakka_http_go_rebind_outcome_init(const coakka_http_go_api_t *,
                                        coakka_http_host_rebind_outcome_t *);
coakka_http_host_status_t
coakka_http_go_rebind(const coakka_http_go_api_t *,
                      coakka_http_host_service_t *,
                      const coakka_http_host_rebind_request_t *,
                      coakka_http_host_rebind_outcome_t *);
void coakka_http_go_publication_init(const coakka_http_go_api_t *,
                                     coakka_http_host_publication_request_t *);
void coakka_http_go_publication_outcome_init(
    const coakka_http_go_api_t *, coakka_http_host_publication_outcome_t *);
coakka_http_host_status_t
coakka_http_go_publish_routes(const coakka_http_go_api_t *,
                              coakka_http_host_service_t *,
                              const coakka_http_host_publication_request_t *,
                              coakka_http_host_publication_outcome_t *);

void coakka_http_go_monitor_policy_init(const coakka_http_go_api_t *,
                                        coakka_http_host_monitor_policy_t *);
void coakka_http_go_monitor_config_init(const coakka_http_go_api_t *,
                                        coakka_http_host_monitor_config_t *);
void coakka_http_go_monitor_apply_outcome_init(
    const coakka_http_go_api_t *, coakka_http_host_monitor_apply_outcome_t *);
void coakka_http_go_monitor_snapshot_init(
    const coakka_http_go_api_t *, coakka_http_host_monitor_snapshot_t *);
void coakka_http_go_monitor_event_init(const coakka_http_go_api_t *,
                                       coakka_http_host_monitor_event_t *);
void coakka_http_go_monitor_page_init(const coakka_http_go_api_t *,
                                      coakka_http_host_monitor_page_t *);
coakka_http_host_status_t
coakka_http_go_monitor_config(const coakka_http_go_api_t *,
                              coakka_http_host_service_t *,
                              coakka_http_host_monitor_config_t *);
coakka_http_host_status_t
coakka_http_go_monitor_apply(const coakka_http_go_api_t *,
                             coakka_http_host_service_t *, uint64_t,
                             const coakka_http_host_monitor_policy_t *,
                             coakka_http_host_monitor_apply_outcome_t *);
coakka_http_host_status_t
coakka_http_go_monitor_snapshot(const coakka_http_go_api_t *,
                                coakka_http_host_service_t *,
                                coakka_http_host_monitor_snapshot_t *);
coakka_http_host_status_t
coakka_http_go_monitor_read(const coakka_http_go_api_t *,
                            coakka_http_host_service_t *, uint64_t, uint32_t,
                            coakka_http_host_monitor_event_t *, uint32_t,
                            coakka_http_host_monitor_page_t *);
coakka_http_host_status_t
coakka_http_go_monitor_wait(const coakka_http_go_api_t *,
                            coakka_http_host_service_t *, uint64_t);
coakka_http_host_status_t
coakka_http_go_monitor_interrupt(const coakka_http_go_api_t *,
                                 coakka_http_host_service_t *);

#ifdef __cplusplus
}
#endif

#endif
