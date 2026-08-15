package app

// Page identifies the currently displayed top-level view.
type Page int

const (
	// PageChat is the main conversation view.
	PageChat Page = iota
	// PageSkills is the skill management view (view/enable/disable/refresh).
	PageSkills
)

// String returns a stable identifier for the page.
func (p Page) String() string {
	switch p {
	case PageChat:
		return "chat"
	case PageSkills:
		return "skills"
	default:
		return "unknown"
	}
}
