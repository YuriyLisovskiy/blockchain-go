// Copyright (c) 2018 Yuriy Lisovskiy
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"

	"github.com/YuriyLisovskiy/blockchain-go/src/core/types"
	"github.com/YuriyLisovskiy/blockchain-go/src/core/types/tx_io"
	"github.com/YuriyLisovskiy/blockchain-go/src/core/vars"
	db_pkg "github.com/YuriyLisovskiy/blockchain-go/src/db"
)

type UTXOSet struct {
	BlockChain BlockChain
}

func (u UTXOSet) FindSpendableOutputs(pubkeyHash []byte, amount float64) (float64, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	accumulated := float64(0)
	db := u.BlockChain.db
	err := db.View(func(tx *db_pkg.Tx) error {
		b := tx.Bucket(vars.UTXO_BUCKET)
		if b == nil {
			return errors.New(fmt.Sprintf("bucket '%x' does not exist", vars.UTXO_BUCKET))
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			txID := hex.EncodeToString(k)
			outs := tx_io.DeserializeOutputs(v)
			for outIdx, out := range outs.Outputs {
				if out.IsLockedWithKey(pubkeyHash) && accumulated < amount {
					accumulated += out.Value
					unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	return accumulated, unspentOutputs
}

func (u UTXOSet) FindUTXO(pubKeyHash []byte) []tx_io.TXOutput {
	db := u.BlockChain.db
	var UTXOs []tx_io.TXOutput
	err := db.View(func(tx *db_pkg.Tx) error {
		b := tx.Bucket(vars.UTXO_BUCKET)
		if b == nil {
			return errors.New(fmt.Sprintf("bucket '%x' does not exist", vars.UTXO_BUCKET))
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			outs := tx_io.DeserializeOutputs(v)
			for _, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) {
					UTXOs = append(UTXOs, out)
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	return UTXOs
}

func (u UTXOSet) CountTransactions() int {
	db := u.BlockChain.db
	counter := 0
	err := db.View(func(tx *db_pkg.Tx) error {
		b := tx.Bucket(vars.UTXO_BUCKET)
		if b == nil {
			return errors.New(fmt.Sprintf("bucket '%x' does not exist", vars.UTXO_BUCKET))
		}
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			counter++
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	return counter
}

func (u UTXOSet) Reindex() {
	db := u.BlockChain.db
	err := db.Batch(func(tx *db_pkg.Tx) error {
		err := tx.DeleteBucket(vars.UTXO_BUCKET)
		if err != nil && err != db_pkg.ErrBucketNotFound {
			log.Panic(err)
		}
		_, err = tx.CreateBucket(vars.UTXO_BUCKET)
		if err != nil {
			log.Panic(err)
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	UTXO := u.BlockChain.FindUTXO()
	err = db.Batch(func(tx *db_pkg.Tx) error {
		b := tx.Bucket(vars.UTXO_BUCKET)
		if b == nil {
			return errors.New(fmt.Sprintf("bucket '%x' does not exist", vars.UTXO_BUCKET))
		}
		for txID, outs := range UTXO {
			key, err := hex.DecodeString(txID)
			if err != nil {
				log.Panic(err)
			}
			err = b.Put(key, outs.Serialize())
			if err != nil {
				log.Panic(err)
			}
		}
		return nil
	})
}

func (u UTXOSet) Update(block types.Block) {
	db := u.BlockChain.db
	vars.DBMutex.Lock()
	err := db.Batch(func(tx *db_pkg.Tx) error {
		b := tx.Bucket([]byte(vars.UTXO_BUCKET))
		if b == nil {
			return errors.New(fmt.Sprintf("bucket '%x' does not exist", vars.UTXO_BUCKET))
		}
		for _, tx := range block.Transactions {
			if tx.IsCoinBase() == false {
				for _, vin := range tx.VIn {
					updatedOuts := tx_io.TXOutputs{}
					outsBytes := b.Get(vin.PreviousTx)
					outs := tx_io.DeserializeOutputs(outsBytes)
					for outIdx, out := range outs.Outputs {
						if outIdx != vin.VOut {
							updatedOuts.Outputs = append(updatedOuts.Outputs, out)
						}
					}
					if len(updatedOuts.Outputs) == 0 {
						err := b.Delete(vin.PreviousTx)
						if err != nil {
							log.Panic(err)
						}
					} else {
						err := b.Put(vin.PreviousTx, updatedOuts.Serialize())
						if err != nil {
							log.Panic(err)
						}
					}
				}
			}
			newOutputs := tx_io.TXOutputs{}
			for _, out := range tx.VOut {
				newOutputs.Outputs = append(newOutputs.Outputs, out)
			}
			err := b.Put(tx.Hash, newOutputs.Serialize())
			if err != nil {
				log.Panic(err)
			}
		}
		return nil
	})
	vars.DBMutex.Unlock()
	if err != nil {
		log.Panic(err)
	}
}
