// Copyright (c) 2018 Yuriy Lisovskiy
// Distributed under the BSD 3-Clause software license, see the accompanying
// file LICENSE or https://opensource.org/licenses/BSD-3-Clause.

package rpc

const (
	C_TX         = "tx"
	C_INV        = "inv"
	C_PING       = "ping"
	C_PONG       = "pong"
	C_ADDR       = "addr"
	C_BLOCK      = "block"
	C_ERROR      = "error"
	C_VERSION    = "version"
	C_GET_DATA   = "getdata"
	C_GET_BLOCKS = "getblocks"
)

const (
	PROTOCOL       = "tcp"
	NODE_VERSION   = 1
	COMMAND_LENGTH = 12
)