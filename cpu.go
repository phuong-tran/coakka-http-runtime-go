package coakkahttp

// CPUPolicy selects Core's startup placement for the HTTP dispatcher and the
// native threads it creates. It is not a process CPU quota or Go worker count.
type CPUPolicy uint32

const (
	// CPUAuto (the zero value) asks Core for up to two allowed logical CPUs.
	// On unsupported platforms it preserves inherited placement and reports it.
	CPUAuto CPUPolicy = 0
	// CPUSingle asks Core for one allowed logical CPU. Unsupported platforms
	// reject this explicit request rather than reporting a placement not applied.
	CPUSingle CPUPolicy = 1
)

// CPUPlacement is Core's startup placement outcome. Unknown values are retained.
type CPUPlacement uint32

const (
	// CPUPlacementApplied means Core verified the requested logical CPU set.
	CPUPlacementApplied CPUPlacement = 0
	// CPUPlacementSingleAvailable means AUTO found only one allowed logical CPU.
	CPUPlacementSingleAvailable CPUPlacement = 1
	// CPUPlacementUnsupported means AUTO retained platform placement unchanged.
	CPUPlacementUnsupported CPUPlacement = 2
)

// CPUInfo is an immutable, pointer-free copy of Core's dispatcher entry facts.
// It is not live CPU utilization, a process-wide affinity report or a CPU quota.
type CPUInfo struct {
	// RequestedPolicy is the preference Core accepted at construction.
	RequestedPolicy CPUPolicy
	// Placement distinguishes verified selection, one-CPU fallback and unsupported.
	Placement CPUPlacement
	// AvailableCPUCount counts inherited allowed logical CPUs; zero means unknown.
	AvailableCPUCount uint32
	// SelectedCPUCount is Core's selected set length, at most two; zero means unknown.
	SelectedCPUCount uint32
	// SelectedCPUIDs contains Core's logical IDs; unused entries are ^uint32(0).
	SelectedCPUIDs [2]uint32
}
