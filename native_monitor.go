package coakkahttp

/*
#include "bridge.h"
*/
import "C"

import (
	"errors"
	"unsafe"
)

func nativeMonitorPolicy(api *C.coakka_http_go_api_t, source MonitorPolicy) C.coakka_http_host_monitor_policy_t {
	var value C.coakka_http_host_monitor_policy_t
	C.coakka_http_go_monitor_policy_init(api, &value)
	value.collection = C.coakka_http_host_monitor_collection_t(source.Collection)
	value.notification = C.coakka_http_host_monitor_notification_t(source.Notification)
	value.latency = C.coakka_http_host_monitor_latency_t(source.Latency)
	value.active_event_capacity = C.uint32_t(source.ActiveEventCapacity)
	value.aggregate_categories = nativeMonitorCategories(source.AggregateCategories)
	value.event_categories = nativeMonitorCategories(source.EventCategories)
	return value
}

func copyMonitorPolicy(source C.coakka_http_host_monitor_policy_t) MonitorPolicy {
	return MonitorPolicy{
		Collection:          MonitorCollection(source.collection),
		Notification:        MonitorNotification(source.notification),
		Latency:             MonitorLatency(source.latency),
		ActiveEventCapacity: uint32(source.active_event_capacity),
		AggregateCategories: MonitorCategory(source.aggregate_categories),
		EventCategories:     MonitorCategory(source.event_categories),
	}
}

func copyMonitorConfig(source C.coakka_http_host_monitor_config_t) MonitorConfig {
	return MonitorConfig{
		Generation:                  uint64(source.generation),
		CollectionEpoch:             uint64(source.collection_epoch),
		ChangeSequence:              uint64(source.change_sequence),
		EpochStartedMonotonicNanos:  uint64(source.epoch_started_monotonic_ns),
		PolicyAppliedMonotonicNanos: uint64(source.policy_applied_monotonic_ns),
		SupportedCategories:         MonitorCategory(source.supported_categories),
		ReservedEventCapacity:       uint32(source.reserved_event_capacity),
		MaxEventsPerRead:            uint32(source.max_events_per_read),
		FixedLatencyBuckets:         source.fixed_latency_buckets != 0,
		SignalReserved:              source.signal_reserved != 0,
		Policy:                      copyMonitorPolicy(source.policy),
	}
}

func (native *nativeService) monitorConfig() (MonitorConfig, error) {
	var value C.coakka_http_host_monitor_config_t
	C.coakka_http_go_monitor_config_init(native.api, &value)
	err := resultError(C.coakka_http_go_monitor_config(native.api, native.service, &value))
	return copyMonitorConfig(value), err
}

func (native *nativeService) monitorSnapshot() (MonitorSnapshot, error) {
	var source C.coakka_http_host_monitor_snapshot_t
	C.coakka_http_go_monitor_snapshot_init(native.api, &source)
	if err := resultError(C.coakka_http_go_monitor_snapshot(native.api, native.service, &source)); err != nil {
		return MonitorSnapshot{}, err
	}
	failureCount, err := checkedCount(source.failure_count)
	if err != nil || failureCount > len(source.failure_counts) {
		return MonitorSnapshot{}, errors.New("native monitor failure count exceeds its ABI bound")
	}
	latencyCount, err := checkedCount(source.latency_bucket_count)
	if err != nil || latencyCount > len(source.latency_buckets) {
		return MonitorSnapshot{}, errors.New("native monitor latency count exceeds its ABI bound")
	}
	failures := make([]MonitorFailureCount, failureCount)
	for index := range failures {
		failures[index] = MonitorFailureCount{
			Reason: uint32(source.failure_counts[index].reason),
			Count:  uint64(source.failure_counts[index].count),
		}
	}
	latencies := make([]MonitorLatencyBucket, latencyCount)
	for index := range latencies {
		latencies[index] = MonitorLatencyBucket{
			UpperBoundNanos: uint64(source.latency_buckets[index].upper_bound_ns),
			Count:           uint64(source.latency_buckets[index].count),
		}
	}
	var statuses [5]uint64
	for index := range statuses {
		statuses[index] = uint64(source.response_status_family[index])
	}
	return MonitorSnapshot{
		Collecting:              source.collecting != 0,
		Stale:                   source.stale != 0,
		Partial:                 source.partial != 0,
		SnapshotSequence:        uint64(source.snapshot_sequence),
		ObservedMonotonicNanos:  uint64(source.observed_monotonic_ns),
		CollectedChangeSequence: uint64(source.collected_change_sequence),
		InboundAdmittedRequests: uint64(source.admitted_requests),
		InboundRequestBytes:     uint64(source.request_bytes),
		ExchangeCompleted:       uint64(source.exchange_completed),
		ExchangeFailed:          uint64(source.exchange_failed),
		ExchangeTimedOut:        uint64(source.exchange_timed_out),
		ExchangeDisconnected:    uint64(source.exchange_disconnected),
		ExchangeCancelled:       uint64(source.exchange_cancelled),
		ResponseStatusFamily:    statuses,
		RetainedEvents:          uint64(source.retained_events),
		EventOverwriteCount:     uint64(source.overwritten_events),
		EventDropCount:          uint64(source.dropped_events),
		SignalNotifyCount:       uint64(source.signal_notifications),
		SignalCoalescedCount:    uint64(source.coalesced_signals),
		SignalFailureCount:      uint64(source.signal_failures),
		Config:                  copyMonitorConfig(source.config),
		Health:                  copyHealth(source.health),
		FailureCounts:           failures,
		LatencyBuckets:          latencies,
	}, nil
}

func (native *nativeService) monitorApply(expectedGeneration uint64, policy MonitorPolicy) (MonitorApplyOutcome, error) {
	value := nativeMonitorPolicy(native.api, policy)
	var source C.coakka_http_host_monitor_apply_outcome_t
	C.coakka_http_go_monitor_apply_outcome_init(native.api, &source)
	if err := resultError(C.coakka_http_go_monitor_apply(
		native.api, native.service, C.uint64_t(expectedGeneration), &value, &source)); err != nil {
		return MonitorApplyOutcome{}, err
	}
	return MonitorApplyOutcome{
		Reason:    MonitorApplyReason(source.reason),
		Actual:    uint64(source.actual),
		Limit:     uint64(source.limit),
		Changed:   source.changed != 0,
		Effective: copyMonitorConfig(source.effective),
	}, nil
}

func (native *nativeService) monitorRead(afterSequence uint64, maximumEvents uint32) (MonitorEventPage, error) {
	if maximumEvents == 0 || uint64(maximumEvents) > maximumProjectionItems {
		return MonitorEventPage{}, errors.New("monitor page size is outside the Go bound")
	}
	arena := nativeArena{}
	defer arena.close()
	memory, err := arena.allocate(uintptr(maximumEvents) * unsafe.Sizeof(C.coakka_http_host_monitor_event_t{}))
	if err != nil {
		return MonitorEventPage{}, err
	}
	events := unsafe.Slice((*C.coakka_http_host_monitor_event_t)(memory), int(maximumEvents))
	for index := range events {
		C.coakka_http_go_monitor_event_init(native.api, &events[index])
	}
	var page C.coakka_http_host_monitor_page_t
	C.coakka_http_go_monitor_page_init(native.api, &page)
	if err = resultError(C.coakka_http_go_monitor_read(
		native.api, native.service, C.uint64_t(afterSequence), C.uint32_t(maximumEvents),
		(*C.coakka_http_host_monitor_event_t)(memory), C.uint32_t(maximumEvents), &page)); err != nil {
		return MonitorEventPage{}, err
	}
	count, err := checkedCount(page.count)
	if err != nil || count > len(events) {
		return MonitorEventPage{}, errors.New("native monitor page count exceeds its supplied capacity")
	}
	result := make([]MonitorEvent, count)
	for index := range result {
		source := events[index]
		result[index] = MonitorEvent{
			Kind:                   MonitorEventKind(source.kind),
			Sequence:               uint64(source.sequence),
			ChangeSequence:         uint64(source.change_sequence),
			ConfigGeneration:       uint64(source.config_generation),
			CollectionEpoch:        uint64(source.collection_epoch),
			ObservedMonotonicNanos: uint64(source.observed_monotonic_ns),
			Category:               uint64(source.category),
			FailureReason:          uint32(source.failure_reason),
			QueueScope:             uint32(source.queue_scope),
			QueueDepth:             uint32(source.queue_depth),
			QueueCapacity:          uint32(source.queue_capacity),
			Exchange: ExchangeID{
				Slot:       uint64(source.exchange.slot),
				Generation: uint64(source.exchange.generation),
			},
		}
	}
	return MonitorEventPage{
		Events:                  result,
		OldestAvailableSequence: uint64(page.oldest_sequence),
		LatestSequence:          uint64(page.latest_sequence),
		MissedEvents:            uint64(page.overwritten_events),
		RemainingEvents:         uint64(page.remaining_events),
	}, nil
}

func (native *nativeService) monitorWait(timeout C.uint64_t) error {
	return resultError(C.coakka_http_go_monitor_wait(native.api, native.service, timeout))
}

func (native *nativeService) monitorInterrupt() error {
	return resultError(C.coakka_http_go_monitor_interrupt(native.api, native.service))
}
