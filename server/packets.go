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
	"encoding/binary"
	"igo/log"
	"igo/mysql"
	"time"
)

//Packeter the packet interface
type Packeter interface {
	writePacket([]byte) error
	readPacket() ([]byte, error)
}

/******************************************************************************
*                           Packets Process                                   *
******************************************************************************/
// Packets documentation:
// http://dev.mysql.com/doc/internals/en/client-server-protocol.html

// Read packet to buffer 'data'
func (c *Client) readPacket() ([]byte, error) {
	var payload []byte
	for {
		// Read packet header
		data, err := c.buf.readNext(4)
		if err != nil {
			log.Error(err)
			c.close()
			return nil, driver.ErrBadConn
		}

		// Packet Length [24 bit]
		pktLen := int(uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16)

		if pktLen < 1 {
			log.Error(mysql.ErrMalformPkt)
			c.close()
			return nil, driver.ErrBadConn
		}

		// Check Packet Sync [8 bit]
		// log.Debugf("read Client seq: %v, %v", data[3], c.sequence)
		if data[3] != c.sequence {
			if data[3] > c.sequence {
				return nil, mysql.ErrPktSyncMul
			}
			return nil, mysql.ErrPktSync
		}
		c.sequence++

		// Read packet body [pktLen bytes]
		data, err = c.buf.readNext(pktLen)
		if err != nil {
			log.Error(err)
			c.close()
			return nil, driver.ErrBadConn
		}

		isLastPacket := (pktLen < mysql.MaxPacketSize)

		// Zero allocations for non-splitting packets
		if isLastPacket && payload == nil {
			return data, nil
		}

		payload = append(payload, data...)

		if isLastPacket {
			log.Debug("payload: ", string(payload))
			return payload, nil
		}
	}
}

// Write packet buffer 'data'
func (c *Client) writePacket(data []byte) error {
	pktLen := len(data) - 4
	if pktLen > c.maxPacketAllowed {
		return mysql.ErrPktTooLarge
	}

	for {
		var size int
		if pktLen >= mysql.MaxPacketSize {
			data[0] = 0xff
			data[1] = 0xff
			data[2] = 0xff
			size = mysql.MaxPacketSize
		} else {
			data[0] = byte(pktLen)
			data[1] = byte(pktLen >> 8)
			data[2] = byte(pktLen >> 16)
			size = pktLen
		}
		data[3] = c.sequence

		// Write packet
		if c.writeTimeout > 0 {
			if err := c.netConn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
				return err
			}
		}
		log.Debug("client write packet:", data)
		n, err := c.netConn.Write(data[:4+size])
		if err == nil && n == 4+size {
			c.sequence++
			if size != mysql.MaxPacketSize {
				return nil
			}
			pktLen -= size
			data = data[size:]
			continue
		}

		// Handle error
		if err == nil { // n != len(data)
			log.Error(mysql.ErrMalformPkt)
		} else {
			log.Error(err)
		}
		return driver.ErrBadConn
	}
}

/******************************************************************************
*                             Command Packets                                 *
******************************************************************************/

func (c *Client) writeCommandPacket(command byte) error {
	// Reset Packet Sequence
	c.sequence = 0

	data := c.buf.takeSmallBuffer(4 + 1)
	if data == nil {
		// can not take the buffer. Something must be wrong with the connection
		log.Error(mysql.ErrBusyBuffer)
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
		log.Error(mysql.ErrBusyBuffer)
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
		log.Error(mysql.ErrBusyBuffer)
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

func (c *Client) writeResultPackets(payloads [][]byte) error {
	var err error

	for _, payload := range payloads {
		pktLen := len(payload)
		data := c.buf.takeBuffer(pktLen + 4)
		if data == nil {
			// can not take the buffer. Something must be wrong with the connection
			log.Error(mysql.ErrBusyBuffer)
			return driver.ErrBadConn
		}

		// Add arg
		copy(data[4:], payload)
		err = c.writePacket(data)
	}
	c.sequence = 0
	return err
}

// Read packet to buffer 'data'
func (mc *mysqlConn) readPacket() ([]byte, error) {
	var payload []byte
	for {
		// Read packet header
		data, err := mc.buf.readNext(4)
		if err != nil {
			log.Error(err)
			mc.Close()
			return nil, driver.ErrBadConn
		}

		// Packet Length [24 bit]
		pktLen := int(uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16)

		if pktLen < 1 {
			log.Error(mysql.ErrMalformPkt)
			mc.Close()
			return nil, driver.ErrBadConn
		}

		// Check Packet Sync [8 bit]
		// log.Debugf("read mysqlConn seq: %v, %v", data[3], mc.sequence)
		if data[3] != mc.sequence {
			if data[3] > mc.sequence {
				return nil, mysql.ErrPktSyncMul
			}
			return nil, mysql.ErrPktSync
		}
		mc.sequence++

		// Read packet body [pktLen bytes]
		data, err = mc.buf.readNext(pktLen)
		if err != nil {
			log.Error(err)
			mc.Close()
			return nil, driver.ErrBadConn
		}

		isLastPacket := (pktLen < mysql.MaxPacketSize)
		//log.Debug("mysqlConn read package: ", data, pktLen, mc.sequence)
		// Zero allocations for non-splitting packets
		if isLastPacket && payload == nil {
			return data, nil
		}

		payload = append(payload, data...)

		if isLastPacket {
			log.Debug("mysqlConn read package: ", payload)
			return payload, nil
		}
	}
}

// Write packet buffer 'data'
func (mc *mysqlConn) writePacket(data []byte) error {
	pktLen := len(data) - 4

	if pktLen > mc.maxPacketAllowed {
		return mysql.ErrPktTooLarge
	}

	for {
		var size int
		if pktLen >= mysql.MaxPacketSize {
			data[0] = 0xff
			data[1] = 0xff
			data[2] = 0xff
			size = mysql.MaxPacketSize
		} else {
			data[0] = byte(pktLen)
			data[1] = byte(pktLen >> 8)
			data[2] = byte(pktLen >> 16)
			size = pktLen
		}
		data[3] = mc.sequence
		// Write packet
		if mc.writeTimeout > 0 {
			if err := mc.netConn.SetWriteDeadline(time.Now().Add(mc.writeTimeout)); err != nil {
				return err
			}
		}
		//log.Debug("mysqlConn write packet:", data[:5], string(data[5:]))
		n, err := mc.netConn.Write(data[:4+size])
		if err == nil && n == 4+size {
			mc.sequence++
			if size != mysql.MaxPacketSize {
				return nil
			}
			pktLen -= size
			data = data[size:]
			continue
		}

		// Handle error
		if err == nil { // n != len(data)
			log.Error(mysql.ErrMalformPkt)
		} else {
			log.Error(err)
		}
		return driver.ErrBadConn
	}
}

// Ok Packet
// http://dev.mysql.com/doc/internals/en/generic-response-packets.html#packet-OK_Packet
func (mc *mysqlConn) handleOkPacket(data []byte) error {
	var n, m int

	// 0x00 [1 byte]

	// Affected rows [Length Coded Binary]
	mc.affectedRows, _, n = readLengthEncodedInteger(data[1:])

	// Insert id [Length Coded Binary]
	mc.insertID, _, m = readLengthEncodedInteger(data[1+n:])

	// server_status [2 bytes]
	mc.status = readStatus(data[1+n+m : 1+n+m+2])
	// if err := mc.discardResults(); err != nil {
	// 	return err
	// }

	// warning count [2 bytes]
	if !mc.strict {
		return nil
	}

	pos := 1 + n + m + 2
	if binary.LittleEndian.Uint16(data[pos:pos+2]) > 0 {
		log.Warn("mc.getWarnings()")
		return nil
	}
	return nil
}

/******************************************************************************
*                             Command Packets                                 *
******************************************************************************/

func (mc *mysqlConn) writeCommandPacket(command byte) error {
	// Reset Packet Sequence
	mc.sequence = 0

	data := mc.buf.takeSmallBuffer(4 + 1)
	if data == nil {
		// can not take the buffer. Something must be wrong with the connection
		log.Error(mysql.ErrBusyBuffer)
		return driver.ErrBadConn
	}

	// Add command byte
	data[4] = command

	// Send CMD packet
	return mc.writePacket(data)
}

func (mc *mysqlConn) writeCommandPacketStr(command byte, arg string) error {
	// Reset Packet Sequence
	mc.sequence = 0

	pktLen := 1 + len(arg)
	data := mc.buf.takeBuffer(pktLen + 4)
	if data == nil {
		// can not take the buffer. Something must be wrong with the connection
		log.Error(mysql.ErrBusyBuffer)
		return driver.ErrBadConn
	}

	// Add command byte
	data[4] = command

	// Add arg
	copy(data[5:], arg)

	// Send CMD packet
	return mc.writePacket(data)
}

func (mc *mysqlConn) writeCommandPacketUint32(command byte, arg uint32) error {
	// Reset Packet Sequence
	mc.sequence = 0

	data := mc.buf.takeSmallBuffer(4 + 1 + 4)
	if data == nil {
		// can not take the buffer. Something must be wrong with the connection
		log.Error(mysql.ErrBusyBuffer)
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
	return mc.writePacket(data)
}
