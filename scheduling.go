package coakkahttp

// NotificationProfile selects a startup notification cap owned by Core. It is
// not a minimum batch, delay, queue capacity or worker count. Partial turns
// always flush; changing a profile requires a new Runtime.
type NotificationProfile uint32

const (
	// NotificationAuto delegates the selected lane's default to Core.
	NotificationAuto NotificationProfile = 0
	// NotificationSmall selects a cap of one.
	NotificationSmall NotificationProfile = 1
	// NotificationMedium selects a cap of four.
	NotificationMedium NotificationProfile = 4
	// NotificationLarge selects a cap of eight.
	NotificationLarge NotificationProfile = 8
	// NotificationXLarge selects a cap of sixteen.
	NotificationXLarge NotificationProfile = 16
	// NotificationXXLarge selects a cap of thirty-two.
	NotificationXXLarge NotificationProfile = 32
	// NotificationUltra selects a cap of sixty-four.
	NotificationUltra NotificationProfile = 64
)

// NotificationProfiles groups the independent startup request and terminal
// doorbell policies. Its zero value delegates both defaults to Core. Unknown
// values are rejected by Core, not clamped or reinterpreted by Go.
type NotificationProfiles struct {
	// Request controls HTTP/1 request notifications per I/O turn.
	Request NotificationProfile
	// Terminal controls coalesced terminal wakeups, not terminal delivery.
	// Every terminal remains observable immediately in its bounded queue.
	Terminal NotificationProfile
}
