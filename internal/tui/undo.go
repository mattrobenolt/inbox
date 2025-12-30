package tui

import "go.withmatt.com/inbox/internal/gmail"

func (m *Model) undoAvailable() bool {
	return len(m.inbox.undo.threads) > 0
}

func (m *Model) undoRefs() []threadRef {
	if len(m.inbox.undo.threads) == 0 {
		return nil
	}
	refs := make([]threadRef, 0, len(m.inbox.undo.threads))
	for _, thread := range m.inbox.undo.threads {
		refs = append(refs, threadRef{threadID: thread.ThreadID, accountIndex: thread.AccountIndex})
	}
	return refs
}

func (m *Model) threadsForRefs(refs []threadRef) []gmail.Thread {
	if len(refs) == 0 {
		return nil
	}
	want := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		want[threadKey(ref.threadID, ref.accountIndex)] = struct{}{}
	}
	threads := make([]gmail.Thread, 0, len(refs))
	for _, thread := range m.inbox.threads {
		if _, ok := want[threadKey(thread.ThreadID, thread.AccountIndex)]; ok {
			threads = append(threads, thread)
		}
	}
	return threads
}
