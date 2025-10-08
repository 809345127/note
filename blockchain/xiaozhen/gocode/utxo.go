package main

// UTXO 未花费交易输出
type UTXO struct {
	TxID   string // 交易ID
	Index  int    // 输出索引
	Value  int    // 比特币数量
	Owner  string // 所有者地址
}

// UTXOSet UTXO集合
type UTXOSet struct {
	UTXOs []UTXO
}

// AddUTXO 添加UTXO
func (set *UTXOSet) AddUTXO(txID string, index int, value int, owner string) {
	set.UTXOs = append(set.UTXOs, UTXO{
		TxID:  txID,
		Index: index,
		Value: value,
		Owner: owner,
	})
}

// SpendUTXO 花费UTXO
func (set *UTXOSet) SpendUTXO(txID string, index int) bool {
	for i, utxo := range set.UTXOs {
		if utxo.TxID == txID && utxo.Index == index {
			// 从UTXO集合中移除
			set.UTXOs = append(set.UTXOs[:i], set.UTXOs[i+1:]...)
			return true
		}
	}
	return false
}

// FindUTXO 查找UTXO
func (set *UTXOSet) FindUTXO(txID string, index int) (UTXO, bool) {
	for _, utxo := range set.UTXOs {
		if utxo.TxID == txID && utxo.Index == index {
			return utxo, true
		}
	}
	return UTXO{}, false
}