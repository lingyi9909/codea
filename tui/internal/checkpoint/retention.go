package checkpoint

func pruneCheckpointRecords(items []Checkpoint, limit int, recovery *RestoreState) []Checkpoint {
	if limit <= 0 || len(items) <= limit {
		return append([]Checkpoint(nil), items...)
	}

	keep := make(map[string]struct{}, limit+4)
	start := len(items) - limit
	for _, cp := range items[start:] {
		keep[cp.ID] = struct{}{}
	}

	activeTaskID := ""
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].TaskID != "" {
			activeTaskID = items[i].TaskID
			break
		}
	}
	if activeTaskID != "" {
		for _, cp := range items[:start] {
			if cp.TaskID == activeTaskID && (cp.Kind == KindBaseline || cp.Kind == KindSafety) {
				keep[cp.ID] = struct{}{}
			}
		}
	}
	if recovery != nil {
		keep[recovery.TargetID] = struct{}{}
		keep[recovery.SafetyID] = struct{}{}
	}

	out := make([]Checkpoint, 0, len(keep))
	for _, cp := range items {
		if _, ok := keep[cp.ID]; ok {
			out = append(out, cp)
		}
	}
	return out
}
