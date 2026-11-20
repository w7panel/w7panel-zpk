package lock

import "sync"

var resources sync.Map

// Acquire serializes local mutations while coordinating database, object storage and remote authorization.
// It does not claim to provide a transaction across services or distributed locking.
func Acquire(resource string) func() {
	value, _ := resources.LoadOrStore(resource, &sync.Mutex{})
	mutex := value.(*sync.Mutex)
	mutex.Lock()
	return mutex.Unlock
}
