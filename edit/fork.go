package edit

// Fork returns an independent working branch with the same current snapshot,
// version and original/committed baseline. It performs no parsing or I/O. Plans
// from either working object are foreign to the other. The captured immutable
// source set and explicitly supplied registry are shared; mutable file maps are
// copied. Like Working, Fork requires exclusive access while it is called.
func (w *Working) Fork() *Working {
	return &Working{owner: new(byte), version: w.version, root: w.root, o: w.o,
		set: w.set, files: cloneFiles(w.files), original: cloneFiles(w.original)}
}

// Guards returns the original/last-committed physical baselines in name order,
// including unchanged sources. Returned records are independent values. Merely
// inspecting guards neither acknowledges edits nor acquires filesystem access.
func (w *Working) Guards() []Guard {
	guards := make([]Guard, 0, len(w.original))
	for _, name := range w.Names() {
		value := w.original[name]
		guards = append(guards, Guard{Name: name, Digest: digest(value), Bytes: len(value)})
	}
	return guards
}
