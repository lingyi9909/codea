package runtime

// Model is the Codea-owned view of one model currently exposed by a Runtime.
// It deliberately carries only safe display/routing metadata; credentials,
// provider options, SDK details, and other vendor configuration stay below the
// Runtime adapter boundary.
type Model struct {
	Ref          ModelRef
	Name         string
	ProviderName string
	Default      bool
}
