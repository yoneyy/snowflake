package snowflake

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"sync"
	"time"
)

// 64bit 00000000 00000000 00000000 00000000 00000000 00000000 00000000 00000000

type Snowflake struct {
	mu sync.Mutex

	// 起始时间戳
	epoch time.Time

	// 保存 Node 的位数 Node 和 sequence 总共只有22bit位共享
	nodeBits uint8

	// 保存 sequence 序列的位数 Node 和 sequence 总共只有22bit位共享
	sequenceBits uint8

	// 节点ID最大边界数
	maxNodeID int64

	// 序列号最大边界数
	maxSequence int64

	// 节点ID
	nodeID int64

	// 序列号
	sequence int64

	// 最后一次生成snowflake的时间戳
	lastTimestamp int64

	// 节点ID左移位数
	nodeShift uint8

	// 时间戳左移位数
	timestampShift uint8
}

type SnowflakeOption struct {
	// 起始时间戳，用于计算相对时间
	Epoch int64
	// 节点ID
	NodeID int64
}

func NewSnowflake(option *SnowflakeOption) (*Snowflake, error) {
	nodeBits := uint8(10)
	sequenceBits := uint8(12)

	if option.Epoch == 0 {
		option.Epoch = time.Date(2024, 9, 10, 15, 17, 0, 0, time.Local).UnixNano() / int64(time.Millisecond)
	}

	now := time.Now()
	epoch := now.Add(time.Unix(option.Epoch/1000, (option.Epoch%1000)*1000000).Sub(now))

	// 原码转反码
	// 正数的原码与反码、补码一致
	// 负数的原码转反码最高位不变其余都取反（0变1，1变0）

	// 反码转补码
	// 正数的原码与反码、补码一致
	// 负数的反码转补码最高位不变最后位+1 需要有进位的进位

	// 计算节点ID最大边界
	// -1 原码 1000000000000000000000000000000000000000000000000000000000000001
	// -1 原码转反码 1111111111111111111111111111111111111111111111111111111111111110
	// -1 反码转补码（反码转补码+1 补码转反码-1） 1111111111111111111111111111111111111111111111111111111111111111
	// int64(-1) << int64(machineIDBits) --> -1 * 1024 = -1024
	// -1024 原码 1000000000000000000000000000000000000000000000000000010000000000
	// -1024 原码转反码 1111111111111111111111111111111111111111111111111111101111111111
	// -1024 反码转补码（反码转补码+1 补码转反码-1）1111111111111111111111111111111111111111111111111111110000000000
	// ^ 计算异或（两个位相同为0，相异为1）
	// -1 	 	 --> 1111111111111111111111111111111111111111111111111111111111111111
	// ^
	// -1024 	 --> 1111111111111111111111111111111111111111111111111111110000000000
	// maxNodeID = 0000000000000000000000000000000000000000000000000000001111111111
	// maxNodeID --> 转10进制为 1023 即机器ID最大不超过1023
	maxNodeID := int64(-1) ^ (int64(-1) << nodeBits) // 1023

	// 计算序列号最大边界
	// -1 原码 1000000000000000000000000000000000000000000000000000000000000001
	// -1 原码转反码 1111111111111111111111111111111111111111111111111111111111111110
	// -1 反码转补码（反码转补码+1 补码转反码-1） 1111111111111111111111111111111111111111111111111111111111111111
	// int64(-1) << sequenceBits --> -1 * 4096 = -4096
	// -4096 原码 1000000000000000000000000000000000000000000000000001000000000000
	// -4096 原码转反码 1111111111111111111111111111111111111111111111111110111111111111
	// -4096 反码转补码 1111111111111111111111111111111111111111111111111111000000000000
	// -1 	 		 --> 1111111111111111111111111111111111111111111111111111111111111111
	// ^
	// -4096 		 --> 1111111111111111111111111111111111111111111111111111000000000000
	// maxSequence = 0000000000000000000000000000000000000000000000000000111111111111
	// maxSequence --> 转10进制为 4095 即序列号最大不超过4095
	maxSequence := int64(-1) ^ (int64(-1) << int64(sequenceBits)) // 4095

	// Check whether the node id ranges from 0 to maxNodeID
	if option.NodeID < 0 || option.NodeID > maxNodeID {
		return nil, fmt.Errorf("node id must be between 0 and %d", maxNodeID)
	}

	// 节点ID左移位数
	nodeShift := sequenceBits

	// 时间戳左移位数
	timestampShift := nodeBits + sequenceBits

	snowflake := &Snowflake{
		mu:             sync.Mutex{},
		epoch:          epoch,
		nodeBits:       nodeBits,
		sequenceBits:   sequenceBits,
		maxNodeID:      maxNodeID,
		maxSequence:    maxSequence,
		nodeID:         option.NodeID,
		sequence:       0,
		lastTimestamp:  -1,
		nodeShift:      nodeShift,
		timestampShift: timestampShift,
	}
	return snowflake, nil
}

// Generate snowflake ID
// @author yoneyy (y.tianyuan)
func (s *Snowflake) ID() SnowflakeID {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.currentTimestamp()

	// 如果当前时间戳小于最后一次生成snowflakeID的时间戳说明时间发生了回拨
	if now < s.lastTimestamp {
		// 延迟等待 补偿机制
		offset := s.lastTimestamp - now
		if offset <= 5 {
			// 等待一会 使当前时间戳能追赶上最后一次生成snowflakeID的时间
			// 1 << 1 = 2
			// 2 << 1 = 4
			// 3 << 1 = 6
			// 4 << 1 = 8
			// 5 << 1 = 10
			// 假设当前偏移了5ms 则等待10ms后去补偿少的那(10/2)=5ms
			time.Sleep(time.Millisecond * time.Duration(offset<<1))
			// 尝试生成新的时间戳
			now = s.currentTimestamp()
			// 如果生成新的时间戳还是小于最后一次生成snowflakeID的时间则直接抛出错误，终止生成ID
			if now < s.lastTimestamp {
				return -1
			}
		} else {
			// 如果时间回拨超过5ms，抛出错误，终止生成ID
			return -1
		}
	}

	if now == s.lastTimestamp {
		// 递增序列号
		// 这里 & s.maxSequence 是为了保证不超出边界
		// 假设 s.sequence 当前为 4095
		// 4095+1后 为 4096
		// 4096的原码 --> 0000000000000000000000000000000000000000000000000001000000000000
		// & 与 运算 （两个位都为1时，结果才为1。）
		// 4095的原码 --> 0000000000000000000000000000000000000000000000000000111111111111
		// ------------> 0000000000000000000000000000000000000000000000000000000000000000
		// 转 10 进制后 为 0 这样避免当前数超出最大序列号边界
		s.sequence = (s.sequence + 1) & s.maxSequence

		// 0 也说明了s.sequence + 1后超过了最大序列号边界
		// 则等待下一毫秒，通过循环不断获取当前时间戳，直到大于上次生成ID的时间戳
		if s.sequence == 0 {
			for now <= s.lastTimestamp {
				now = s.currentTimestamp()
			}
		}
	} else {
		// 如果当前时间与最后一次获取snowflake的时间不同，说明进入了新的毫秒，重置序列号为0
		s.sequence = 0
	}

	// 更新最后一次获取snowflake的时间戳
	s.lastTimestamp = now

	return s.build()
}

// Build Snowflake ID
// @author yoneyy (y.tianyuan)
func (s *Snowflake) build() SnowflakeID {
	// | 或运算 运算过程 r 表示 result
	// |（或运算 两个位都为0时，结果才为0）
	// 	  0110010011011001111010101100011010010000100000000000000000000000
	//  | 0000000000000000000000000000000000000000001111111111000000000000
	// r: 0110010011011001111010101100011010010000101111111111000000000000
	//  | 0000000000000000000000000000000000000000000000000000000000000110
	// r: 0110010011011001111010101100011010010000101111111111000000000110
	// id = 0110010011011001111010101100011010010000101111111111000000000110
	// id 转10进制为 7267097612291928070
	snowflakeID := SnowflakeID(s.lastTimestamp<<int64(s.timestampShift) |
		s.nodeID<<int64(s.nodeShift) |
		s.sequence)
	return snowflakeID
}

// Get current timestamp
// @author yoneyy (y.tianyuan)
func (s *Snowflake) currentTimestamp() int64 {
	return time.Since(s.epoch).Milliseconds()
}

// ParseInt64 converts an int64 into a snowflake ID
// @author yoneyy (y.tianyuan)
func ParseInt64(id int64) SnowflakeID {
	return SnowflakeID(id)
}

// ParseString converts a string into a snowflake ID
// @author yoneyy (y.tianyuan)
func ParseString(id string) (SnowflakeID, error) {
	snowflakeID, err := strconv.ParseInt(id, 10, 64)
	return SnowflakeID(snowflakeID), err
}

// ParseBytes converts a byte slice into a snowflake ID
// @author yoneyy (y.tianyuan)
func ParseBytes(id []byte) (SnowflakeID, error) {
	snowflakeID, err := strconv.ParseInt(string(id), 10, 64)
	return SnowflakeID(snowflakeID), err
}

// ParseBase64 converts a base64 string into a snowflake ID
// @author yoneyy (y.tianyuan)
func ParseBase64(id string) (SnowflakeID, error) {
	b, err := base64.StdEncoding.DecodeString(id)
	if err != nil {
		return -1, err
	}
	return ParseBytes(b)
}
