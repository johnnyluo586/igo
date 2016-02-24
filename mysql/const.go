// Go MySQL Driver - A MySQL-Driver for Go's database/sql package
//
// Copyright 2012 The Go-MySQL-Driver Authors. All rights reserved.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

package mysql

const (
	MinProtocolVersion byte = 10
	MaxPacketSize           = 1<<24 - 1
	TimeFormat              = "2006-01-02 15:04:05.999999"
)

// MySQL constants documentation:
// http://dev.mysql.com/doc/internals/en/client-server-protocol.html

const (
	HeaderOK          byte = 0x00
	HeaderLocalInFile byte = 0xfb
	HeaderEOF         byte = 0xfe
	HeaderERR         byte = 0xff
)

// https://dev.mysql.com/doc/internals/en/capability-flags.html#packet-Protocol::CapabilityFlags
type ClientFlag uint32

const (
	ClientLongPassword ClientFlag = 1 << iota
	ClientFoundRows
	ClientLongFlag
	ClientConnectWithDB
	ClientNoSchema
	ClientCompress
	ClientODBC
	ClientLocalFiles
	ClientIgnoreSpace
	ClientProtocol41
	ClientInteractive
	ClientSSL
	ClientIgnoreSIGPIPE
	ClientTransactions
	ClientReserved
	ClientSecureConn
	ClientMultiStatements
	ClientMultiResults
	ClientPSMultiResults
	ClientPluginAuth
	ClientConnectAttrs
	ClientPluginAuthLenEncClientData
	ClientCanHandleExpiredPasswords
	ClientSessionTrack
	ClientDeprecateEOF
)

const (
	ComQuit byte = iota + 1
	ComInitDB
	ComQuery
	ComFieldList
	ComCreateDB
	ComDropDB
	ComRefresh
	ComShutdown
	ComStatistics
	ComProcessInfo
	ComConnect
	ComProcessKill
	ComDebug
	ComPing
	ComTime
	ComDelayedInsert
	ComChangeUser
	ComBinlogDump
	ComTableDump
	ComConnectOut
	ComRegisterSlave
	ComStmtPrepare
	ComStmtExecute
	ComStmtSendLongData
	ComStmtClose
	ComStmtReset
	ComSetOption
	ComStmtFetch
)

// https://dev.mysql.com/doc/internals/en/com-query-response.html#packet-Protocol::ColumnType
const (
	FieldTypeDecimal byte = iota
	FieldTypeTiny
	FieldTypeShort
	FieldTypeLong
	FieldTypeFloat
	FieldTypeDouble
	FieldTypeNULL
	FieldTypeTimestamp
	FieldTypeLongLong
	FieldTypeInt24
	FieldTypeDate
	FieldTypeTime
	FieldTypeDateTime
	FieldTypeYear
	FieldTypeNewDate
	FieldTypeVarChar
	FieldTypeBit
)
const (
	FieldTypeJSON byte = iota + 0xf5
	FieldTypeNewDecimal
	FieldTypeEnum
	FieldTypeSet
	FieldTypeTinyBLOB
	FieldTypeMediumBLOB
	FieldTypeLongBLOB
	FieldTypeBLOB
	FieldTypeVarString
	FieldTypeString
	FieldTypeGeometry
)

type FieldFlag uint16

const (
	FlagNotNULL FieldFlag = 1 << iota
	FlagPriKey
	FlagUniqueKey
	FlagMultipleKey
	FlagBLOB
	FlagUnsigned
	FlagZeroFill
	FlagBinary
	FlagEnum
	FlagAutoIncrement
	FlagTimestamp
	FlagSet
	FlagUnknown1
	FlagUnknown2
	FlagUnknown3
	FlagUnknown4
)

// http://dev.mysql.com/doc/internals/en/status-flags.html
type StatusFlag uint16

const (
	StatusInTrans StatusFlag = 1 << iota
	StatusInAutocommit
	StatusReserved // Not in documentation
	StatusMoreResultsExists
	StatusNoGoodIndexUsed
	StatusNoIndexUsed
	StatusCursorExists
	StatusLastRowSent
	StatusDbDropped
	StatusNoBackslashEscapes
	StatusMetadataChanged
	StatusQueryWasSlow
	StatusPsOutParams
	StatusInTransReadonly
	StatusSessionStateChanged
)
