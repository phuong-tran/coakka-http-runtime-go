package coakkahttp

/*
#include "bridge.h"
*/
import "C"

import "errors"

// TLSReloadMode selects how Core handles connections from earlier generations.
type TLSReloadMode uint32

const (
	// TLSReloadGraceful keeps connections using their captured credentials.
	TLSReloadGraceful TLSReloadMode = C.COAKKA_HTTP_HOST_TLS_RELOAD_GRACEFUL
	// TLSReloadDrainExisting retires connections from earlier generations.
	TLSReloadDrainExisting TLSReloadMode = C.COAKKA_HTTP_HOST_TLS_RELOAD_DRAIN_EXISTING
)

// TLSReloadReason is the causal outcome reported by Core.
type TLSReloadReason uint32

const (
	// TLSReloadApplied reports that Core published the credential candidate.
	TLSReloadApplied TLSReloadReason = C.COAKKA_HTTP_HOST_TLS_RELOAD_APPLIED
	// TLSReloadInvalidCredential reports an invalid credential declaration.
	TLSReloadInvalidCredential TLSReloadReason = C.COAKKA_HTTP_HOST_TLS_RELOAD_INVALID_CREDENTIAL
	// TLSReloadSecurityChange reports a change to the listener security mode.
	TLSReloadSecurityChange TLSReloadReason = C.COAKKA_HTTP_HOST_TLS_RELOAD_SECURITY_CHANGE
	// TLSReloadStaleGeneration reports a candidate generation that does not advance.
	TLSReloadStaleGeneration TLSReloadReason = C.COAKKA_HTTP_HOST_TLS_RELOAD_STALE_GENERATION
	// TLSReloadGenerationConflict reports a mismatch with the expected generation.
	TLSReloadGenerationConflict TLSReloadReason = C.COAKKA_HTTP_HOST_TLS_RELOAD_GENERATION_CONFLICT
	// TLSReloadBusy reports another credential preparation in progress.
	TLSReloadBusy TLSReloadReason = C.COAKKA_HTTP_HOST_TLS_RELOAD_BUSY
	// TLSReloadRetainedCapacity reports that old connections pin all retirement slots.
	TLSReloadRetainedCapacity TLSReloadReason = C.COAKKA_HTTP_HOST_TLS_RELOAD_RETAINED_CAPACITY
	// TLSReloadPreparationRejected reports that Core rejected the file-backed material.
	TLSReloadPreparationRejected TLSReloadReason = C.COAKKA_HTTP_HOST_TLS_RELOAD_PREPARATION_REJECTED
	// TLSReloadOutOfMemory reports an allocation failure in Core.
	TLSReloadOutOfMemory TLSReloadReason = C.COAKKA_HTTP_HOST_TLS_RELOAD_OUT_OF_MEMORY
	// TLSReloadClosed reports that Core closed credential publication.
	TLSReloadClosed TLSReloadReason = C.COAKKA_HTTP_HOST_TLS_RELOAD_CLOSED
	// TLSReloadInternal reports an internal credential control failure.
	TLSReloadInternal TLSReloadReason = C.COAKKA_HTTP_HOST_TLS_RELOAD_INTERNAL
)

// TLSReloadRequest supplies one complete file-backed credential candidate.
type TLSReloadRequest struct {
	// Mode selects whether to retire earlier connections.
	Mode TLSReloadMode
	// Security identifies the listener security mode.
	Security TransportSecurity
	// ExpectedGeneration must match the effective generation at publication.
	ExpectedGeneration uint64
	// CredentialGeneration identifies the candidate credential generation.
	CredentialGeneration uint64
	// CredentialID labels the credential material.
	CredentialID string
	// CertificateChainFile names the chain file read by Core.
	CertificateChainFile string
	// PrivateKeyFile names the key file read by Core.
	PrivateKeyFile string
	// TrustRootsFile names the client trust file for mutual TLS.
	TrustRootsFile string
}

// TLSCredentials is Core's effective listener credential observation.
type TLSCredentials struct {
	// Security identifies the listener security mode.
	Security TransportSecurity
	// ReloadMode is the mode of the latest accepted generation.
	ReloadMode TLSReloadMode
	// Generation identifies the effective credentials.
	Generation uint64
	// CredentialID labels the credential material.
	CredentialID string
	// DrainBeforeGeneration is the first generation that can still accept work.
	DrainBeforeGeneration uint64
	// RetiredGenerations counts retained generations in Core.
	RetiredGenerations uint32
	// MaximumRetiredGenerations is the Core retirement capacity.
	MaximumRetiredGenerations uint32
	// Closed reports whether Core closed credential publication.
	Closed bool
	// Preparing reports an in-progress preparation.
	Preparing bool
}

// TLSReloadOutcome carries the accepted or refused update and its effective state.
type TLSReloadOutcome struct {
	// Reason is the typed outcome from Core.
	Reason TLSReloadReason
	// Changed reports whether Core published new credentials.
	Changed bool
	// Effective is the observed state after the update attempt.
	Effective TLSCredentials
}

func copyTLSCredentials(value C.coakka_http_host_tls_credentials_t) TLSCredentials {
	return TLSCredentials{
		Security: TransportSecurity(value.security), ReloadMode: TLSReloadMode(value.reload_mode),
		Generation: uint64(value.generation), CredentialID: C.GoString(&value.credential_id[0]),
		DrainBeforeGeneration:     uint64(value.drain_before_generation),
		RetiredGenerations:        uint32(value.retired_generations),
		MaximumRetiredGenerations: uint32(value.maximum_retired_generations),
		Closed:                    value.closed != 0, Preparing: value.preparing != 0,
	}
}

// TLSCredentials returns Core's current credential generation and retirement state.
func (runtime *Runtime) TLSCredentials() (TLSCredentials, error) {
	native, done, err := runtime.acquire()
	if err != nil {
		return TLSCredentials{}, err
	}
	defer done.release()
	var value C.coakka_http_host_tls_credentials_t
	C.coakka_http_host_tls_credentials_init(&value)
	if err := resultError(C.coakka_http_host_tls_credentials(native.service, &value)); err != nil {
		return TLSCredentials{}, err
	}
	return copyTLSCredentials(value), nil
}

// ReloadTLS prepares and conditionally publishes credentials through Core.
// Check the outcome reason even when err is nil.
func (runtime *Runtime) ReloadTLS(request TLSReloadRequest) (TLSReloadOutcome, error) {
	native, done, err := runtime.acquire()
	if err != nil {
		return TLSReloadOutcome{}, err
	}
	defer done.release()
	var value C.coakka_http_host_tls_reload_request_t
	C.coakka_http_host_tls_reload_request_init(&value)
	value.mode = C.coakka_http_host_tls_reload_mode_t(request.Mode)
	value.security = C.coakka_http_host_security_t(request.Security)
	value.expected_generation = C.uint64_t(request.ExpectedGeneration)
	value.credential_generation = C.uint64_t(request.CredentialGeneration)
	var arena nativeArena
	defer arena.close()
	for _, field := range []struct {
		text string
		out  *C.coakka_http_host_bytes_t
	}{
		{request.CredentialID, &value.credential_id},
		{request.CertificateChainFile, &value.certificate_chain_file},
		{request.PrivateKeyFile, &value.private_key_file},
		{request.TrustRootsFile, &value.trust_roots_file},
	} {
		*field.out, err = arena.text(field.text)
		if err != nil {
			return TLSReloadOutcome{}, err
		}
	}
	var outcome C.coakka_http_host_tls_reload_outcome_t
	C.coakka_http_host_tls_reload_outcome_init(&outcome)
	if err := resultError(C.coakka_http_host_tls_reload(native.service, &value, &outcome)); err != nil {
		return TLSReloadOutcome{}, err
	}
	return TLSReloadOutcome{
		Reason: TLSReloadReason(outcome.reason), Changed: outcome.changed != 0,
		Effective: copyTLSCredentials(outcome.effective),
	}, nil
}

// TLSCredentials returns the copied effective listener credentials.
func (service *Service) TLSCredentials() (TLSCredentials, error) {
	if service == nil || service.runtime == nil {
		return TLSCredentials{}, errors.New("service is closed")
	}
	return service.runtime.TLSCredentials()
}

// ReloadTLS submits one credential update to Core.
func (service *Service) ReloadTLS(request TLSReloadRequest) (TLSReloadOutcome, error) {
	if service == nil || service.runtime == nil {
		return TLSReloadOutcome{}, errors.New("service is closed")
	}
	return service.runtime.ReloadTLS(request)
}
