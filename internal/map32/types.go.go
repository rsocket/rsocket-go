package map32

// Map32 is a safe map for uint32 key.
type Map32 interface {
	// Destroy destroy current map.
	Destroy()
	// Range visits all items.
	Range(fn func(uint32, interface{}) bool)
	// Load loads the value of key.
	Load(key uint32) (v interface{}, ok bool)
	// Store stores the key and value.
	Store(key uint32, value interface{})
	// Delete deletes the key.
	Delete(key uint32)
}
