package utils

import (
	"bytes"
	"encoding/binary"
	"math"
)

func UintToBytes(val uint, index int) []byte {
	b := make([]byte, index)
	for i := 0; i < index; i++ {
		b[i] = byte(val >> (8 * (index - i - 1)))
	}
	return b
}

func Float64ToByte(float float64) []byte {
	bits := math.Float64bits(float)
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, bits)

	return bytes
}

// 返回 uint32
func BytesToUint32(val []byte) uint32 {
	return uint32(val[0]<<24) + uint32(val[1]<<16) + uint32(val[2]<<8) + uint32(val[3])
}

func AmfStringToBytes(b *bytes.Buffer, val string) {
	b.Write(UintToBytes(uint(len(val)), 2))
	b.Write([]byte(val))
}

func AmfDoubleToBytes(b *bytes.Buffer, val float64) {
	b.WriteByte(0x00)
	b.Write(Float64ToByte(val))
}
