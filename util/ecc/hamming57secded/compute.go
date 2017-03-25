//

/*
Package hamming57secded implements extended Hamming (64, 57, 4) code =
the Hamming code (63, 57, 3) + a global parity bit.
It's a Single-Error Correcting and Double-Error Detecting (SEC-DED) code.
*/
package hamming57secded

import (
	"fmt"
)

// Special bit positions in our 64-bit integer format
// (values/variables in this format are named 'packedCheckable' below).
// 'Bitp' stands for "Bit Position".
//
// Each bit position constant is given an explicit value to remind programmers
// that the parity/check bit positions are not intended to be changed casually
// and it makes no sense to add special positions.
//
const (
	globalParityBitp = 1

	checkBitp0 = 2
	checkBitp1 = 3
	checkBitp2 = 4
	checkBitp3 = 5
	checkBitp4 = 6
	checkBitp5 = 7
)

// Only the least significant 56 bits of the input value are used
func PackWithCheckBits(inputVal int64, tagBit int) (packedCheckable int64) {
	packed := inputVal<<8 | int64(tagBit&1)
	checkbits := computeHammingCheckBits(packed)
	v := packed | int64(checkbits)
	parity := computeParityInt64(v)
	return v | int64((parity&1)<<globalParityBitp)
}

func getHammingCheckBits(packedCheckable int64) uint8 {
	return uint8(packedCheckable & 0xfc)
}

func computeHammingCheckBits(packedVal int64) uint8 {
	u := uint64(packedVal)

	p5 := computeParity(u & 0xfffffffe00000000)
	p4 := computeParity(u & 0xffff0001fffc0000)
	p3 := computeParity(u & 0xff00ff01fe03f800)
	p2 := computeParity(u & 0xf0f0f0f1e1e3c700)

	p1 := computeParity(u & 0xcccccccd999b3601)
	p0 := computeParity(u & 0xaaaaaaab5556ad01)
	// The final byte '01' in the masks for check bits 1 and 0
	// is for selecting LSB = the tag bit.
	// The other seven bits in the least significant byte are reserved for
	//    - the check bits (computed in this function) and
	//    - the global parity bit (providing Double-Error Detection),
	// therefore they are not inputs for the Hamming code computation.

	cbs := (p5<<checkBitp5 | p4<<checkBitp4 | p3<<checkBitp3 |
		p2<<checkBitp2 | p1<<checkBitp1 | p0<<checkBitp0)
	return uint8(cbs)
}

func mapSyndromeToBitPos(syndrome uint8) (errBitPos uint8, isCheckBit bool) {
	switch {
	case syndrome > 63:
		panic(fmt.Sprintf("syndrome too big (%x)", syndrome))
	case syndrome > 32:
		return syndrome, false
	case syndrome == 32:
		return checkBitp5, true
	case syndrome > 16:
		return syndrome + 1, false
	case syndrome == 16:
		return checkBitp4, true
	case syndrome > 8:
		return syndrome + 2, false
	case syndrome == 8:
		return checkBitp3, true
	case syndrome > 4:
		return syndrome + 3, false
	case syndrome == 4:
		return checkBitp2, true
	case syndrome == 3: // corresponds to LSB = the tag bit:
		return 0, false
	case syndrome == 2:
		return checkBitp1, true
	case syndrome == 1:
		return checkBitp0, true
	default:
		panic(fmt.Sprintf("syndrome not mapped (%x)", syndrome))
	}
}

// Design based on Henry S. Warren's book "Hacker's delight" (2013), ECC chapter.
func Correct(packedCheckable int64) (nBitErrors int, corrected int64) {
	parity := computeParityInt64(packedCheckable)

	oldCheckbits := getHammingCheckBits(packedCheckable)
	checkbits := computeHammingCheckBits(packedCheckable)
	syndrome := (checkbits ^ oldCheckbits) >> 2

	if parity == 0 {
		if syndrome == 0 { // No errors:
			// return the original (correct) value for convenience:
			return 0, packedCheckable
		}
		// Two errors, uncorrectable:
		return 2, 0
	}

	// One error, correctable:

	if syndrome == 0 { // The global parity bit is in error:
		correctedVal := packedCheckable ^ (1 << globalParityBitp)
		return 1, correctedVal
	}

	errBitPos, _ := mapSyndromeToBitPos(syndrome)
	correctedVal := packedCheckable ^ (1 << errBitPos)
	return 1, correctedVal
}

func computeParityInt64(val int64) int {
	return onesCount64(uint64(val)) & 1
}

func computeParity(val uint64) int {
	return onesCount64(val) & 1
}

// TODO: use the newly added 'math/bits.OnesCount64' when moving to Go 1.9
func onesCount64(x uint64) int {
	var n int
	for x != 0 {
		n++
		x &= x - 1
	}
	return n
}
