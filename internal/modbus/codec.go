package modbus

import (
	"bytes"
	"math"
)

//
// -------- Byte-order helpers (big-endian by byte) --------
//

// U8 returns the low byte.
func U8(b []byte) uint8 {
	if len(b) == 0 {
		return 0
	}
	if len(b) == 1 {
		return uint8(b[0])
	}
	return uint8(b[len(b)-1])
}

func U16(b []byte) uint16 {
	if len(b) < 2 {
		return 0
	}
	return uint16(b[0])<<8 | uint16(b[1])
}

func U32(b []byte) uint32 {
	if len(b) < 4 {
		return 0
	}
	return uint32(b[0])<<24 |
		uint32(b[1])<<16 |
		uint32(b[2])<<8 |
		uint32(b[3])
}

func S16(b []byte) int16 {
	return int16(U16(b))
}

func S32(b []byte) int32 {
	return int32(U32(b))
}

// UTF8 parses a C-style UTF-8 string (NUL-terminated or padded).
func UTF8(b []byte) string {
	if i := bytes.IndexByte(b, 0x00); i >= 0 {
		b = b[:i]
	}
	b = bytes.TrimRight(b, "\x00")
	return string(b)
}

//
// -------- Little-endian helpers (word-swapped) --------
//

func U32LE(b []byte) uint32 {
	if len(b) < 4 {
		return 0
	}
	low := U16(b[0:2])
	high := U16(b[2:4])
	return uint32(low) | (uint32(high) << 16)
}

func S32LE(b []byte) int32 {
	return int32(U32LE(b))
}

//
// -------- Float32 helpers --------
//

// F32BE parses IEEE754 float32, big-endian by byte (ABCD).
func F32BE(b []byte) float32 {
	if len(b) < 4 {
		return 0
	}
	u := U32(b)
	return math.Float32frombits(u)
}

// F32LE parses IEEE754 float32, little-endian by word (CDAB).
func F32LE(b []byte) float32 {
	if len(b) < 4 {
		return 0
	}
	u := U32LE(b)
	return math.Float32frombits(u)
}

// F32CDAB swaps 16-bit words: [AB][CD] -> [CD][AB].
func F32CDAB(b []byte) float32 {
	if len(b) < 4 {
		return 0
	}
	tmp := []byte{b[2], b[3], b[0], b[1]}
	return F32BE(tmp)
}

// F32BADC swaps bytes inside each word: [AB][CD] -> [BA][DC].
func F32BADC(b []byte) float32 {
	if len(b) < 4 {
		return 0
	}
	tmp := []byte{b[1], b[0], b[3], b[2]}
	return F32BE(tmp)
}

//
// -------- 64-bit helpers --------
//

func U64BE(b []byte) uint64 {
	if len(b) < 8 {
		return 0
	}
	return uint64(b[0])<<56 |
		uint64(b[1])<<48 |
		uint64(b[2])<<40 |
		uint64(b[3])<<32 |
		uint64(b[4])<<24 |
		uint64(b[5])<<16 |
		uint64(b[6])<<8 |
		uint64(b[7])
}

func S64BE(b []byte) int64 {
	return int64(U64BE(b))
}

func F64BE(b []byte) float64 {
	if len(b) < 8 {
		return 0
	}
	u := U64BE(b)
	return math.Float64frombits(u)
}
