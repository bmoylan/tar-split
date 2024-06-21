package storage

import (
	"hash"
	"hash/crc64"
)

// NewHash can be overridden to change the hashing algorithm used for tar entries.
var NewHash = NewCRC64

// CRCTable is the default table used for crc64 sum calculations
var CRCTable = crc64.MakeTable(crc64.ISO)

// NewCRC64 is the default function if NewHash is not overridden.
func NewCRC64() hash.Hash {
	return crc64.New(CRCTable)
}
