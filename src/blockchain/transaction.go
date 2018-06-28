package blockchain

import (
	"fmt"
	"log"
	"bytes"
	"strings"
	"math/big"
	"crypto/rand"
	"crypto/ecdsa"
	"encoding/gob"
	"encoding/hex"
	"crypto/sha256"
	"crypto/elliptic"
	w "github.com/YuriyLisovskiy/blockchain-go/src/wallet"
	txPkg "github.com/YuriyLisovskiy/blockchain-go/src/tx"
)

type Transaction struct {
	ID   []byte
	VIn  []txPkg.TXInput
	VOut []txPkg.TXOutput
}

func (tx Transaction) IsCoinBase() bool {
	return len(tx.VIn) == 1 && len(tx.VIn[0].TxId) == 0 && tx.VIn[0].VOut == -1
}

func (tx Transaction) Serialize() []byte {
	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}
	return encoded.Bytes()
}

func (tx *Transaction) Hash() []byte {
	var hash [32]byte
	txCopy := *tx
	txCopy.ID = []byte{}
	hash = sha256.Sum256(txCopy.Serialize())
	return hash[:]
}

func (tx *Transaction) Sign(privateKey ecdsa.PrivateKey, prevTXs map[string]Transaction) {
	if tx.IsCoinBase() {
		return
	}
	for _, vin := range tx.VIn {
		if prevTXs[hex.EncodeToString(vin.TxId)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}
	txCopy := tx.TrimmedCopy()
	for inID, vin := range txCopy.VIn {
		prevTx := prevTXs[hex.EncodeToString(vin.TxId)]
		txCopy.VIn[inID].Signature = nil
		txCopy.VIn[inID].PubKey = prevTx.VOut[vin.VOut].PubKeyHash
		dataToSign := fmt.Sprintf("%x\n", txCopy)
		r, s, err := ecdsa.Sign(rand.Reader, &privateKey, []byte(dataToSign))
		if err != nil {
			log.Panic(err)
		}
		signature := append(r.Bytes(), s.Bytes()...)
		tx.VIn[inID].Signature = signature
		txCopy.VIn[inID].PubKey = nil
	}
}

func (tx Transaction) String() string {
	var lines []string
	lines = append(lines, fmt.Sprintf("--- Transaction %x:", tx.ID))
	for i, input := range tx.VIn {
		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       TXID:      %x", input.TxId))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.VOut))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubKey))
	}
	for i, output := range tx.VOut {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PubKeyHash))
	}
	return strings.Join(lines, "\n")
}

func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []txPkg.TXInput
	var outputs []txPkg.TXOutput
	for _, vIn := range tx.VIn {
		inputs = append(inputs, txPkg.TXInput{vIn.TxId, vIn.VOut, nil, nil})
	}
	for _, vOut := range tx.VOut {
		outputs = append(outputs, txPkg.TXOutput{vOut.Value, vOut.PubKeyHash})
	}
	txCopy := Transaction{tx.ID, inputs, outputs}
	return txCopy
}

func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	if tx.IsCoinBase() {
		return true
	}
	for _, vin := range tx.VIn {
		if prevTXs[hex.EncodeToString(vin.TxId)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}
	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()
	for inID, vin := range tx.VIn {
		prevTx := prevTXs[hex.EncodeToString(vin.TxId)]
		txCopy.VIn[inID].Signature = nil
		txCopy.VIn[inID].PubKey = prevTx.VOut[vin.VOut].PubKeyHash
		r := big.Int{}
		s := big.Int{}
		sigLen := len(vin.Signature)
		r.SetBytes(vin.Signature[:(sigLen / 2)])
		s.SetBytes(vin.Signature[(sigLen / 2):])
		x := big.Int{}
		y := big.Int{}
		keyLen := len(vin.PubKey)
		x.SetBytes(vin.PubKey[:(keyLen / 2)])
		y.SetBytes(vin.PubKey[(keyLen / 2):])
		dataToVerify := fmt.Sprintf("%x\n", txCopy)
		rawPubKey := ecdsa.PublicKey{curve, &x, &y}
		if ecdsa.Verify(&rawPubKey, []byte(dataToVerify), &r, &s) == false {
			return false
		}
		txCopy.VIn[inID].PubKey = nil
	}
	return true
}

func NewCoinBaseTX(to, data string) *Transaction {
	if data == "" {
		randData := make([]byte, 20)
		_, err := rand.Read(randData)
		if err != nil {
			log.Panic(err)
		}
		data = fmt.Sprintf("%x", randData)
	}
	txIn := txPkg.TXInput{[]byte{}, -1, nil, []byte(data)}
	txOut := txPkg.NewTXOutput(subsidy, to)
	tx := Transaction{nil, []txPkg.TXInput{txIn}, []txPkg.TXOutput{*txOut}}
	tx.ID = tx.Hash()
	return &tx
}

func NewUTXOTransaction(wallet *w.Wallet, to string, amount int, UTXOSet *UTXOSet) *Transaction {
	var inputs []txPkg.TXInput
	var outputs []txPkg.TXOutput
	pubKeyHash := w.HashPubKey(wallet.PublicKey)
	acc, validOutputs := UTXOSet.FindSpendableOutputs(pubKeyHash, amount)
	if acc < amount {
		log.Panic("ERROR: Not enough funds")
	}
	for txId, outs := range validOutputs {
		txID, err := hex.DecodeString(txId)
		if err != nil {
			log.Panic(err)
		}
		for _, out := range outs {
			input := txPkg.TXInput{txID, out, nil, wallet.PublicKey}
			inputs = append(inputs, input)
		}
	}
	from := fmt.Sprintf("%s", wallet.GetAddress())
	outputs = append(outputs, *txPkg.NewTXOutput(amount, to))
	if acc > amount {
		outputs = append(outputs, *txPkg.NewTXOutput(acc-amount, from)) // a change
	}
	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()
	UTXOSet.BlockChain.SignTransaction(&tx, wallet.PrivateKey)
	return &tx
}

func DeserializeTransaction(data []byte) Transaction {
	var transaction Transaction
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&transaction)
	if err != nil {
		log.Panic(err)
	}
	return transaction
}