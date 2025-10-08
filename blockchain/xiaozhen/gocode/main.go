package main

import "fmt"

func main() {
	// 初始化区块链
	bc := Blockchain{}
	genesisBlock := CreateGenesisBlock()
	bc.Blocks = append(bc.Blocks, genesisBlock)

	// 初始化UTXO集合（添加创世区块的UTXO）
	for _, tx := range genesisBlock.Body.Transactions {
		for i, output := range tx.Outputs {
			bc.UTXOSet.AddUTXO(tx.ID, i, output.Value, output.PubKeyHash)
		}
	}

	// 创建示例交易
	tx1 := CreateTransaction(
		[]TxInput{
			{
				TxID:      "genesis_transaction",
				OutIndex:  0,
				Signature: "signature1",
				PubKey:    "alice_public_key",
			},
		},
		[]TxOutput{
			{
				Value:      30,
				PubKeyHash: "bob_address",
			},
			{
				Value:      19,
				PubKeyHash: "alice_address", // 找零
			},
		},
	)

	tx2 := CreateTransaction(
		[]TxInput{
			{
				TxID:      tx1.ID,
				OutIndex:  0,
				Signature: "signature2",
				PubKey:    "bob_public_key",
			},
		},
		[]TxOutput{
			{
				Value:      25,
				PubKeyHash: "charlie_address",
			},
			{
				Value:      4,
				PubKeyHash: "bob_address", // 找零
			},
		},
	)

	// 创建第一个区块，包含铸币交易和普通交易
	coinbaseTx1 := CreateCoinbaseTransaction([]Transaction{*tx1}, "miner1_address", 25, &bc.UTXOSet)
	bc.AddBlock([]Transaction{coinbaseTx1, *tx1})

	// 创建第二个区块，包含铸币交易和普通交易
	coinbaseTx2 := CreateCoinbaseTransaction([]Transaction{*tx2}, "miner2_address", 25, &bc.UTXOSet)
	bc.AddBlock([]Transaction{coinbaseTx2, *tx2})

	// 打印区块链
	fmt.Println("=== 区块链信息 ===")
	for _, block := range bc.Blocks {
		fmt.Printf("区块 #%d\n", block.Header.Index)
		fmt.Printf("  时间戳: %s\n", block.Header.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("  前一个区块哈希: %s\n", block.Header.PreviousHash)
		fmt.Printf("  当前区块哈希: %s\n", block.Header.Hash)
		fmt.Printf("  Merkle根: %s\n", block.Header.MerkleRoot)
		fmt.Printf("  Nonce值: %d\n", block.Header.Nonce)
		fmt.Printf("  交易数量: %d\n", len(block.Body.Transactions))
		
		// 打印每笔交易的详细信息
		for i, tx := range block.Body.Transactions {
			fmt.Printf("    交易 #%d: %s\n", i, tx.ID)
			if tx.ID[:8] == "coinbase" {
				fmt.Printf("      类型: 铸币交易\n")
			} else {
				fmt.Printf("      类型: 普通交易\n")
			}
			fmt.Printf("      输入数量: %d, 输出数量: %d\n", len(tx.Inputs), len(tx.Outputs))
			
			// 打印交易输出详情
			for j, output := range tx.Outputs {
				fmt.Printf("      输出 #%d: %d 比特币 -> %s\n", j, output.Value, output.PubKeyHash)
			}
		}
		fmt.Println("-------------------")
	}
	
	// 验证区块链完整性
	fmt.Println("\n=== 区块链验证 ===")
	for i := 1; i < len(bc.Blocks); i++ {
		prevBlock := bc.Blocks[i-1]
		currentBlock := bc.Blocks[i]
		
		if currentBlock.Header.PreviousHash != prevBlock.Header.Hash {
			fmt.Printf("错误: 区块 #%d 的前一个区块哈希不匹配\n", currentBlock.Header.Index)
		} else {
			fmt.Printf("区块 #%d 的哈希链接验证通过\n", currentBlock.Header.Index)
		}
	}
	
	// 展示UTXO集合
	fmt.Println("\n=== UTXO集合 ===")
	fmt.Printf("当前UTXO总数: %d\n", len(bc.UTXOSet.UTXOs))
	for i, utxo := range bc.UTXOSet.UTXOs {
		fmt.Printf("UTXO #%d: 交易ID=%s, 输出索引=%d, 价值=%d 比特币, 所有者=%s\n", 
			i, utxo.TxID, utxo.Index, utxo.Value, utxo.Owner)
	}
	
	// 计算总供应量
	totalSupply := 0
	for _, utxo := range bc.UTXOSet.UTXOs {
		totalSupply += utxo.Value
	}
	fmt.Printf("\n比特币总供应量: %d\n", totalSupply)
}