/*
 * Copyright (C) 2018 The ontology Authors
 * This file is part of The ontology library.
 *
 * The ontology is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The ontology is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The ontology.  If not, see <http://www.gnu.org/licenses/>.
 */
package utils

import (
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	types2 "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	oComm "github.com/ontio/ontology/common"
	sysconfig "github.com/ontio/ontology/common/config"
	"github.com/ontio/ontology/core/types"
	types3 "github.com/ontio/ontology/http/ethrpc/types"
)

func EthBlockFromOntology(block *types.Block, fullTx bool, bloom types2.Bloom) map[string]interface{} {
	if block == nil {
		return nil
	}
	hash := block.Hash()
	transactions, gasUsed, ethTxs := EthTransactionsFromOntology(block.Transactions, common.BytesToHash(hash.ToArray()), uint64(block.Header.Height))

	var blockTxs interface{}

	if fullTx {
		blockTxs = ethTxs
	} else {
		blockTxs = transactions
	}
	return FormatBlock(*block, 0, gasUsed, blockTxs, bloom)
}

func RawEthBlockFromOntology(block *types.Block, bloom types2.Bloom) *types2.Block {
	if block == nil {
		return nil
	}
	hash := block.Hash()
	gasUsed, ethTxs := RawEthTransactionsFromOntology(block.Transactions, common.BytesToHash(hash.ToArray()), uint64(block.Header.Height))

	h := &types2.Header{
		ParentHash:  common.Hash(block.Header.PrevBlockHash),
		UncleHash:   common.Hash{},
		Coinbase:    common.Address{},
		Root:        common.Hash{},
		TxHash:      common.Hash(block.Header.TransactionsRoot),
		ReceiptHash: common.Hash{},
		Bloom:       bloom,
		Difficulty:  big.NewInt(0),
		Number:      big.NewInt(int64(block.Header.Height)),
		GasLimit:    0,
		GasUsed:     gasUsed.Uint64(),
		Time:        uint64(block.Header.Timestamp),
		Extra:       []byte{},
		MixDigest:   common.Hash{},
		Nonce:       types2.BlockNonce{},
	}
	return types2.NewBlock(h, ethTxs, nil, nil, new(trie.Trie))
}

func EthTransactionsFromOntology(txs []*types.Transaction, blockHash common.Hash, blockNumber uint64) ([]common.Hash, *big.Int, []*types3.Transaction) {
	var transactionHashes []common.Hash
	var transactions []*types3.Transaction
	gasUsed := big.NewInt(0)
	for idx, tx := range txs {
		hash := tx.Hash()
		rpcTx, err := OntTxToEthTx(*tx, blockHash, blockNumber, uint64(idx))
		if err != nil {
			continue
		}
		gasUsed.Add(gasUsed, big.NewInt(int64(rpcTx.Gas)))
		transactionHashes = append(transactionHashes, common.BytesToHash(hash.ToArray()))
		transactions = append(transactions, rpcTx)
	}
	return transactionHashes, gasUsed, transactions
}

func RawEthTransactionsFromOntology(txs []*types.Transaction, blockHash common.Hash, blockNumber uint64) (*big.Int, []*types2.Transaction) {
	var transactions []*types2.Transaction
	gasUsed := big.NewInt(0)
	for _, tx := range txs {
		if tx.IsEipTx() {
			eipTx, err := tx.GetEIP155Tx()
			if err != nil {
				continue
			}
			gasUsed.Add(gasUsed, big.NewInt(int64(eipTx.Gas())))
			transactions = append(transactions, eipTx)
		}
	}
	return gasUsed, transactions
}

func OntTxToEthTx(tx types.Transaction, blockHash common.Hash, blockNumber, index uint64) (*types3.Transaction, error) {
	eip155Tx, err := tx.GetEIP155Tx()
	if err != nil {
		return nil, err
	}
	return NewTransaction(eip155Tx, common.Hash(tx.Hash()), blockHash, blockNumber, index)
}

func FormatBlock(block types.Block, gasLimit uint64, gasUsed *big.Int, transactions interface{}, bloom types2.Bloom) map[string]interface{} {
	size := len(block.ToArray())
	header := block.Header
	hash := header.Hash()
	transactionsRoot := types2.EmptyRootHash
	if oComm.UINT256_EMPTY != header.TransactionsRoot {
		transactionsRoot = common.BytesToHash(header.TransactionsRoot[:])
	}
	ret := map[string]interface{}{
		"number":           hexutil.Uint64(header.Height),
		"hash":             common.BytesToHash(hash[:]),
		"parentHash":       common.BytesToHash(header.PrevBlockHash[:]),
		"nonce":            types2.BlockNonce{}, // PoW specific
		"sha3Uncles":       types2.EmptyUncleHash,
		"logsBloom":        bloom,
		"transactionsRoot": transactionsRoot,
		"stateRoot":        common.Hash{},
		"miner":            common.Address{},
		"mixHash":          common.Hash{},
		"difficulty":       hexutil.Uint64(0),
		"totalDifficulty":  hexutil.Uint64(0),
		"extraData":        hexutil.Bytes{},
		"size":             hexutil.Uint64(size),
		"gasLimit":         hexutil.Uint64(gasLimit),
		"gasUsed":          (*hexutil.Big)(gasUsed),
		"timestamp":        hexutil.Uint64(header.Timestamp),
		"uncles":           []common.Hash{},
		"receiptsRoot":     types2.EmptyRootHash,
	}
	if !reflect.ValueOf(transactions).IsNil() {
		switch transactions.(type) {
		case []common.Hash:
			ret["transactions"] = transactions.([]common.Hash)
		case []*types3.Transaction:
			ret["transactions"] = transactions.([]*types3.Transaction)
		}
	} else {
		ret["transactions"] = []common.Hash{}
	}
	return ret
}

func EthToOntAddr(address common.Address) oComm.Address {
	return oComm.Address(address)
}

func EthToOntHash(txHash common.Hash) oComm.Uint256 {
	return oComm.Uint256(txHash)
}

func OntToEthHash(txHash oComm.Uint256) common.Hash {
	return common.Hash(txHash)
}

func NewTransaction(tx *types2.Transaction, txHash, blockHash common.Hash, blockNumber, index uint64) (*types3.Transaction, error) {
	signer := types2.NewEIP155Signer(big.NewInt(int64(GetChainId())))
	from, err := signer.Sender(tx)
	if err != nil {
		return nil, err
	}
	v, r, s := tx.RawSignatureValues()
	rpcTx := &types3.Transaction{
		From:     from,
		Gas:      hexutil.Uint64(tx.Gas()),
		GasPrice: (*hexutil.Big)(tx.GasPrice()),
		Hash:     txHash,
		Input:    hexutil.Bytes(tx.Data()),
		Nonce:    hexutil.Uint64(tx.Nonce()),
		To:       tx.To(),
		Value:    (*hexutil.Big)(tx.Value()),
		V:        (*hexutil.Big)(v),
		R:        (*hexutil.Big)(r),
		S:        (*hexutil.Big)(s),
	}

	if blockHash != (common.Hash{}) {
		rpcTx.BlockHash = &blockHash
		rpcTx.BlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(blockNumber))
		rpcTx.TransactionIndex = (*hexutil.Uint64)(&index)
	}

	return rpcTx, nil
}

func GetChainId() uint32 {
	return sysconfig.DefConfig.P2PNode.EVMChainId
}
