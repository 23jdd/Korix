package Koris

import "github.com/23jdd/Koris/storage"

type (
	Store    = storage.Store
	Tx       = storage.Tx
	Iterator = storage.Iterator
)

var NewMemoryStore = storage.NewMemoryStore
var OpenBboltStore = storage.OpenBbolt
