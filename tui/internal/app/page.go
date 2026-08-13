package app

// Page identifies the currently displayed top-level view.
type Page int

const (
	// PageChat is the main conversation view. V1 only ships this page;
	// Session/Skill/Agent pages are reserved for later Tasks.
	PageChat Page = iota
)

// String returns a stable identifier for the page.
func (p Page) String() string {
	switch p {
	case PageChat:
		return "chat"
	default:
		return "unknown"
	}
}
