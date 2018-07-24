// Copyright (c) 2018 Yuriy Lisovskiy
// Distributed under the BSD 3-Clause software license, see the accompanying
// file LICENSE or https://opensource.org/licenses/BSD-3-Clause.

package db

import (
	"os"
)

type file interface {
	Fd() uintptr
	ReadAt(b []byte, off int64) (n int, err error)
	Stat() (fi os.FileInfo, err error)
	WriteAt(b []byte, off int64) (n int, err error)
}
