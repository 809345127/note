package main

import (
	"fmt"
	"time"
)

// Blockchain 区块链结构
type Blockchain struct {
	Blocks   []*Block
	UTXOSet  UTXOSet // 添加UTXO集合
}

// AddBlock 添加区块到链
func (bc *Blockchain) AddBlock(transactions []Transaction) {
	prevBlock := bc.Blocks[len(bc.Blocks)-1]
	newBlock := NewBlock(prevBlock.Header.Index+1, transactions, prevBlock.Header.Hash)
	newBlock.MineBlock(2) // 难度设置为2
	bc.Blocks = append(bc.Blocks, newBlock)
	
	// 更新UTXO集合
	bc.UpdateUTXOSet(transactions)
}

// UpdateUTXOSet 更新UTXO集合
func (bc *Blockchain) UpdateUTXOSet(transactions []Transaction) {
	// 处理每笔交易
	for _, tx := range transactions {
		// 如果是铸币交易（没有输入），只添加输出
		if len(tx.Inputs) == 0 {
			for i, output := range tx.Outputs {
				bc.UTXOSet.AddUTXO(tx.ID, i, output.Value, output.PubKeyHash)
			}
			continue
		}

		// 处理普通交易
		// 1. 花费输入（从UTXO集合中移除）
		for _, input := range tx.Inputs {
			success := bc.UTXOSet.SpendUTXO(input.TxID, input.OutIndex)
			if !success {
				// 注意：实际系统中应该拒绝这种交易（双重花费）
				fmt.Printf("警告: 尝试花费不存在的UTXO: %s:%d (可能是双重花费)\n", input.TxID, input.OutIndex)
			}
		}

		// 2. 添加新的UTXO
		for i, output := range tx.Outputs {
			bc.UTXOSet.AddUTXO(tx.ID, i, output.Value, output.PubKeyHash)
		}
	}
}

// CreateCoinbaseTransaction 创建铸币交易（包含区块奖励和手续费）
func CreateCoinbaseTransaction(transactions []Transaction, minerAddress string, blockReward int, utxoSet *UTXOSet) Transaction {
	// 计算所有交易的手续费总和
	totalFees := 0

	for _, tx := range transactions {
		// 跳过铸币交易，它们没有输入
		if len(tx.Inputs) == 0 {
			continue
		}

		totalInput := 0
		totalOutput := 0

		// 从UTXO集合查询输入的实际金额
		for _, input := range tx.Inputs {
			utxo, found := utxoSet.FindUTXO(input.TxID, input.OutIndex)
			if found {
				totalInput += utxo.Value
			} else {
				// 注意：实际系统中应该拒绝引用不存在UTXO的交易
				fmt.Printf("警告: 交易引用了不存在的UTXO: %s:%d\n", input.TxID, input.OutIndex)
			}
		}

		// 计算该交易的总输出
		for _, output := range tx.Outputs {
			totalOutput += output.Value
		}

		// 手续费 = 输入总额 - 输出总额
		fee := totalInput - totalOutput
		if fee > 0 {
			totalFees += fee
		}
	}

	// 创建铸币交易
	coinbaseTx := Transaction{
		ID:      "coinbase_" + fmt.Sprintf("%d", time.Now().Unix()),
		Inputs:  []TxInput{}, // 铸币交易无输入
		Outputs: []TxOutput{
			{
				Value:      blockReward + totalFees, // 区块奖励 + 所有手续费
				PubKeyHash: minerAddress,            // 矿工地址
			},
		},
		Timestamp: time.Now(),
	}

	return coinbaseTx
}

// CreateGenesisBlock 创建创世区块
func CreateGenesisBlock() *Block {
	// 创世区块包含一个特殊的交易
	genesisTx := Transaction{
		ID: "genesis_transaction",
		Inputs: []TxInput{}, // 创世交易没有输入
		Outputs: []TxOutput{
			{
				Value:      50, // 创世区块奖励
				PubKeyHash: "alice_address", // 给 alice，让她可以花费
			},
		},
		Timestamp: time.Now(),
	}

	return NewBlock(0, []Transaction{genesisTx}, "")
}