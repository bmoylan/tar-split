package storage

import (
	"hash"
	"hash/crc64"
)

// CRCTable is the default table used for crc64 sum calculations
var CRCTable = crc64.MakeTable(crc64.ISO)

var NewHash = func() hash.Hash {
	return crc64.New(CRCTable)
}
