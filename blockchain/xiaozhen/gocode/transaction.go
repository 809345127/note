package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// Transaction 交易结构体
type Transaction struct {
	ID        string     // 交易ID（哈希值）
	Inputs    []TxInput  // 交易输入
	Outputs   []TxOutput // 交易输出
	Timestamp time.Time  // 交易时间戳
}

// TxInput 交易输入
type TxInput struct {
	TxID      string // 引用的交易ID
	OutIndex  int    // 引用的输出索引
	Signature string // 输入签名
	PubKey    string // 公钥
}

// TxOutput 交易输出
type TxOutput struct {
	Value      int    // 比特币数量
	PubKeyHash string // 接收方公钥哈希（地址）
}

// calculateTransactionHash 计算交易哈希
func (tx *Transaction) calculateHash() string {
	// 使用交易的实际内容计算哈希，避免循环依赖
	inputsData := ""
	for _, input := range tx.Inputs {
		inputsData += fmt.Sprintf("%s%d%s%s", input.TxID, input.OutIndex, input.Signature, input.PubKey)
	}

	outputsData := ""
	for _, output := range tx.Outputs {
		outputsData += fmt.Sprintf("%d%s", output.Value, output.PubKeyHash)
	}

	record := fmt.Sprintf("%s%s%s", inputsData, outputsData, tx.Timestamp.String())
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

// VerifyTransaction 验证交易
func (tx *Transaction) VerifyTransaction(isCoinbase bool) bool {
	// 铸币交易特殊处理：没有输入是合法的
	if isCoinbase {
		if len(tx.Inputs) != 0 {
			return false // 铸币交易不应该有输入
		}
		if len(tx.Outputs) == 0 {
			return false
		}
		// 检查输出总额
		for _, output := range tx.Outputs {
			if output.Value <= 0 {
				return false
			}
		}
		return true
	}

	// 普通交易验证
	if len(tx.Inputs) == 0 || len(tx.Outputs) == 0 {
		return false
	}

	// 检查输出总额是否合理
	totalOutput := 0
	for _, output := range tx.Outputs {
		if output.Value <= 0 {
			return false
		}
		totalOutput += output.Value
	}

	// 检查签名（简化版本）
	for _, input := range tx.Inputs {
		if input.Signature == "" || input.PubKey == "" {
			return false
		}
	}

	return true
}

// CreateTransaction 创建新交易
func CreateTransaction(inputs []TxInput, outputs []TxOutput) *Transaction {
	tx := &Transaction{
		ID:        "",
		Inputs:    inputs,
		Outputs:   outputs,
		Timestamp: time.Now(),
	}
	
	// 计算交易ID
	tx.ID = tx.calculateHash()
	return tx
}

// =============================================================================
// Pay to Script Hash (P2SH) 数据结构
// =============================================================================

// P2SH版本的交易输出 - 包含脚本哈希而不是公钥哈希
type TxOutputP2SH struct {
	Value      int    // 比特币数量
	ScriptHash string // 脚本哈希（P2SH地址）
}

// P2SH版本的交易输入 - 必须提供脚本和满足脚本的数据
type TxInputP2SH struct {
	TxID         string // 引用的交易ID
	OutIndex     int    // 引用的输出索引
	ScriptSig    string // 脚本签名数据（包含满足脚本的参数）
	RedeemScript string // 原始脚本（其哈希值等于输出中的ScriptHash）
}

// P2SH交易结构体
type TransactionP2SH struct {
	ID        string        // 交易ID（哈希值）
	Inputs    []TxInputP2SH // P2SH交易输入
	Outputs   []TxOutputP2SH // P2SH交易输出
	Timestamp time.Time     // 交易时间戳
}