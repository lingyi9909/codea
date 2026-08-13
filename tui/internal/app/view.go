package app

// View renders the current application state. The full three-region layout
// (header / chat / input) is implemented in Step 6; this placeholder exists so
// *Model satisfies tea.Model while the subscription wiring lands first.
func (m *Model) View() string {
	return ""
}
