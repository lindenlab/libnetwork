package bitseq

import (
	"github.com/docker/libnetwork/datastore"
	"github.com/docker/libnetwork/types"
)

// Key provides the Key to be used in KV Store
func (h *Handle) Key() []string {
	h.Lock()
	defer h.Unlock()
	return []string{h.app, h.id}
}

// KeyPrefix returns the immediate parent key that can be used for tree walk
func (h *Handle) KeyPrefix() []string {
	h.Lock()
	defer h.Unlock()
	return []string{h.app}
}

// Value marshals the data to be stored in the KV store
func (h *Handle) Value() []byte {
	b, err := h.ToByteArray()
	if err != nil {
		return []byte{}
	}
	return b
}

// Index returns the latest DB Index as seen by this object
func (h *Handle) Index() uint64 {
	h.Lock()
	defer h.Unlock()
	return h.dbIndex
}

// SetIndex method allows the datastore to store the latest DB Index into this object
func (h *Handle) SetIndex(index uint64) {
	h.Lock()
	h.dbIndex = index
	h.Unlock()
}

func (h *Handle) watchForChanges() error {
	h.Lock()
	store := h.store
	h.Unlock()

	if store == nil {
		return nil
	}

	kvpChan, err := store.KVStore().Watch(datastore.Key(h.Key()...), nil)
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case kvPair := <-kvpChan:
				// Only process remote update
				if kvPair != nil && (kvPair.LastIndex != h.getDBIndex()) {
					h.Lock()
					h.dbIndex = kvPair.LastIndex
					h.Unlock()
					h.FromByteArray(kvPair.Value)
				}
			}
		}
	}()
	return nil
}

func (h *Handle) writeToStore() error {
	h.Lock()
	store := h.store
	h.Unlock()
	if store == nil {
		return nil
	}
	err := store.PutObjectAtomic(h)
	if err == datastore.ErrKeyModified {
		return types.RetryErrorf("failed to perform atomic write (%v). retry might fix the error", err)
	}
	return err
}

func (h *Handle) deleteFromStore() error {
	h.Lock()
	store := h.store
	h.Unlock()
	if store == nil {
		return nil
	}
	return store.DeleteObjectAtomic(h)
}