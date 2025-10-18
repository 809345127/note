package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// BlockHeader 定义区块头结构
type BlockHeader struct {
	Index        int       // 区块高度
	Timestamp    time.Time // 时间戳
	PreviousHash string    // 前一个区块的哈希
	Hash         string    // 当前区块的哈希
	Nonce        int       // 随机数(用于PoW)
	MerkleRoot   string    // Merkle Tree 根哈希
	Bits         uint32    // 难度目标的紧凑表示
}

// BlockBody 定义区块体结构
type BlockBody struct {
	Transactions []Transaction // 交易列表
}

// Block 组合区块头和区块体
type Block struct {
	Header BlockHeader
	Body   BlockBody
}

// calculateMerkleRoot 计算交易的Merkle根
func calculateMerkleRoot(transactions []Transaction) string {
	if len(transactions) == 0 {
		return ""
	}

	// 将交易ID作为叶子节点
	var leaves []string
	for _, tx := range transactions {
		leaves = append(leaves, tx.calculateHash())
	}

	// 构建Merkle树
	return buildMerkleTree(leaves)
}

// buildMerkleTree 构建Merkle树
func buildMerkleTree(leaves []string) string {
	if len(leaves) == 1 {
		return leaves[0]
	}

	var newLeaves []string
	for i := 0; i < len(leaves); i += 2 {
		if i+1 < len(leaves) {
			// 计算两个叶子节点的哈希
			combined := leaves[i] + leaves[i+1]
			hash := sha256.Sum256([]byte(combined))
			newLeaves = append(newLeaves, hex.EncodeToString(hash[:]))
		} else {
			// 奇数个叶子节点，最后一个重复
			combined := leaves[i] + leaves[i]
			hash := sha256.Sum256([]byte(combined))
			newLeaves = append(newLeaves, hex.EncodeToString(hash[:]))
		}
	}

	return buildMerkleTree(newLeaves)
}

// NewBlock 创建新区块
func NewBlock(index int, transactions []Transaction, previousHash string, bits uint32) *Block {
	block := &Block{
		Header: BlockHeader{
			Index:        index,
			Timestamp:    time.Now(),
			PreviousHash: previousHash,
			Nonce:        0,
			Bits:         bits, // 添加难度目标
		},
		Body: BlockBody{
			Transactions: transactions,
		},
	}

	// 计算Merkle根
	block.Header.MerkleRoot = calculateMerkleRoot(transactions)
	block.Header.Hash = block.calculateHash()
	return block
}

// calculateHash 计算区块哈希(SHA256)
func (b *Block) calculateHash() string {
	record := fmt.Sprintf("%d%s%s%s%d%d",
		b.Header.Index,
		b.Header.Timestamp.String(),
		b.Header.PreviousHash,
		b.Header.MerkleRoot,
		len(b.Body.Transactions),
		b.Header.Nonce)
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

// MineBlock 工作量证明(PoW)
func (b *Block) MineBlock() {
	target := bitsToTarget(b.Header.Bits)

	for !isHashValidTarget(b.Header.Hash, target) {
		b.Header.Nonce++
		b.Header.Hash = b.calculateHash()
	}
	fmt.Printf("区块挖矿成功: %s\n", b.Header.Hash)
}

// bitsToTarget 将Bits字段转换为256位目标值
func bitsToTarget(bits uint32) string {
	// 简化版：实际比特币实现更复杂
	// 这里使用前导零数量作为难度表示
	difficulty := bits >> 24 // 取高8位作为难度
	target := ""
	for i := 0; i < int(difficulty); i++ {
		target += "0"
	}
	for i := int(difficulty); i < 64; i++ {
		target += "f"
	}
	return target
}

// isHashValidTarget 验证哈希是否满足目标值要求
func isHashValidTarget(hash string, target string) bool {
	return hash <= target
}

// isHashValid 验证哈希是否满足难度要求（保留原函数以兼容）
func isHashValid(hash string, difficulty int) bool {
	prefix := ""
	for i := 0; i < difficulty; i++ {
		prefix += "0"
	}
	return hash[:difficulty] == prefix
}
