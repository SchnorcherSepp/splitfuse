package fuse

import (
	"testing"
	"github.com/SchnorcherSepp/splitfuse/core"
)

func TestCalcChunkSize(t *testing.T) {
	var test uint64

	test = 0
	if x := calcChunkSize(0, test); x != 0 {
		t.Errorf("TestCalcChunkSize Test #1: (%d)", x)
	}
	if x := calcChunkSize(1, test); x != 0 {
		t.Errorf("TestCalcChunkSize Test #2: (%d)", x)
	}

	test = 17
	if x := calcChunkSize(0, test); x != test {
		t.Errorf("TestCalcChunkSize Test #3: (%d)", x)
	}
	if x := calcChunkSize(1, test); x != 0 {
		t.Errorf("TestCalcChunkSize Test #4: (%d)", x)
	}
	if x := calcChunkSize(2, test); x != 0 {
		t.Errorf("TestCalcChunkSize Test #5: (%d)", x)
	}

	test = core.CHUNKSIZE*3 + 99
	if x := calcChunkSize(0, test); x != core.CHUNKSIZE {
		t.Errorf("TestCalcChunkSize Test #6: (%d)", x)
	}
	if x := calcChunkSize(1, test); x != core.CHUNKSIZE {
		t.Errorf("TestCalcChunkSize Test #7: (%d)", x)
	}
	if x := calcChunkSize(2, test); x != core.CHUNKSIZE {
		t.Errorf("TestCalcChunkSize Test #8: (%d)", x)
	}
	if x := calcChunkSize(3, test); x != 99 {
		t.Errorf("TestCalcChunkSize Test #9: (%d)", x)
	}
	if x := calcChunkSize(4, test); x != 0 {
		t.Errorf("TestCalcChunkSize Test #10: (%d)", x)
	}
	if x := calcChunkSize(5, test); x != 0 {
		t.Errorf("TestCalcChunkSize Test #11: (%d)", x)
	}

	test = core.CHUNKSIZE
	if x := calcChunkSize(0, test); x != test {
		t.Errorf("TestCalcChunkSize Test #12: (%d)", x)
	}
	if x := calcChunkSize(1, test); x != 0 {
		t.Errorf("TestCalcChunkSize Test #13: (%d)", x)
	}
	if x := calcChunkSize(3, test); x != 0 {
		t.Errorf("TestCalcChunkSize Test #14: (%d)", x)
	}

	test = core.CHUNKSIZE - 1
	if x := calcChunkSize(0, test); x != test {
		t.Errorf("TestCalcChunkSize Test #15: (%d)", x)
	}
	if x := calcChunkSize(1, test); x != 0 {
		t.Errorf("TestCalcChunkSize Test #16: (%d)", x)
	}
	if x := calcChunkSize(3, test); x != 0 {
		t.Errorf("TestCalcChunkSize Test #17: (%d)", x)
	}

	test = core.CHUNKSIZE + 1
	if x := calcChunkSize(0, test); x != core.CHUNKSIZE {
		t.Errorf("TestCalcChunkSize Test #18: (%d)", x)
	}
	if x := calcChunkSize(1, test); x != 1 {
		t.Errorf("TestCalcChunkSize Test #19: (%d)", x)
	}
	if x := calcChunkSize(3, test); x != 0 {
		t.Errorf("TestCalcChunkSize Test #20: (%d)", x)
	}
	if x := calcChunkSize(4, test); x != 0 {
		t.Errorf("TestCalcChunkSize Test #21: (%d)", x)
	}
}
