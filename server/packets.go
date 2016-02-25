// Go MySQL Driver - A MySQL-Driver for Go's database/sql package
//
// Copyright 2012 The Go-MySQL-Driver Authors. All rights reserved.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

package server

import (
	"database/sql/driver"
	"igo/log"
	. "igo/mysql"
)

/******************************************************************************
*                             Command Packets                                 *
******************************************************************************/

func (c *Client) writeCommandPacket(command byte) error {
	// Reset Packet Sequence
	c.sequence = 0

	data := c.buf.takeSmallBuffer(4 + 1)
	if data == nil {
		// can not take the buffer. Something must be wrong with the connection
		log.Error(ErrBusyBuffer)
		return driver.ErrBadConn
	}

	// Add command byte
	data[4] = command

	// Send CMD packet
	return c.writePacket(data)
}

func (c *Client) writeCommandPacketStr(command byte, arg string) error {
	// Reset Packet Sequence
	c.sequence = 0

	pktLen := 1 + len(arg)
	data := c.buf.takeBuffer(pktLen + 4)
	if data == nil {
		// can not take the buffer. Something must be wrong with the connection
		log.Error(ErrBusyBuffer)
		return driver.ErrBadConn
	}

	// Add command byte
	data[4] = command

	// Add arg
	copy(data[5:], arg)

	// Send CMD packet
	return c.writePacket(data)
}

func (c *Client) writeCommandPacketUint32(command byte, arg uint32) error {
	// Reset Packet Sequence
	c.sequence = 0

	data := c.buf.takeSmallBuffer(4 + 1 + 4)
	if data == nil {
		// can not take the buffer. Something must be wrong with the connection
		log.Error(ErrBusyBuffer)
		return driver.ErrBadConn
	}

	// Add command byte
	data[4] = command

	// Add arg [32 bit]
	data[5] = byte(arg)
	data[6] = byte(arg >> 8)
	data[7] = byte(arg >> 16)
	data[8] = byte(arg >> 24)

	// Send CMD packet
	return c.writePacket(data)
}
