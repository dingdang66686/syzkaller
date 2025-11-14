// Copyright 2024 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package prog

import (
	"encoding/binary"
	"math/rand"
	"testing"
)

func TestGenerateConcurrent(t *testing.T) {
	target, rs, _ := initRandomTargetTest(t, "test", "64")
	ct := target.DefaultChoiceTable()
	
	// Test with different numbers of sequences
	testCases := []struct {
		name         string
		ncalls       int
		numSequences int
	}{
		{"single sequence", 10, 1},
		{"two sequences", 10, 2},
		{"three sequences", 15, 3},
		{"many sequences", 20, 5},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := target.GenerateConcurrent(rs, tc.ncalls, tc.numSequences, ct)
			
			if tc.numSequences <= 1 {
				// Should use legacy format
				if p.HasSequences() {
					t.Errorf("expected legacy format for numSequences=%d", tc.numSequences)
				}
				if len(p.Calls) == 0 {
					t.Errorf("expected calls in legacy format")
				}
			} else {
				// Should use multi-sequence format
				if !p.HasSequences() {
					t.Errorf("expected multi-sequence format")
				}
				if len(p.Sequences) != tc.numSequences {
					t.Errorf("expected %d sequences, got %d", tc.numSequences, len(p.Sequences))
				}
				
				totalCalls := 0
				for i, seq := range p.Sequences {
					if len(seq) == 0 {
						t.Errorf("sequence %d is empty", i)
					}
					totalCalls += len(seq)
				}
				
				// Check that total calls is approximately what we requested
				if totalCalls < tc.ncalls-tc.numSequences || totalCalls > tc.ncalls+tc.numSequences {
					t.Errorf("expected approximately %d calls, got %d", tc.ncalls, totalCalls)
				}
			}
			
			// Validate the program
			if err := p.sanitize(false); err != nil {
				t.Errorf("sanitize failed: %v", err)
			}
		})
	}
}

func TestSerializeMultiSequence(t *testing.T) {
	target, rs, _ := initRandomTargetTest(t, "test", "64")
	ct := target.DefaultChoiceTable()
	
	// Generate a multi-sequence program
	p := target.GenerateConcurrent(rs, 12, 3, ct)
	
	if !p.HasSequences() {
		t.Skip("program doesn't have sequences")
	}
	
	// Serialize the program
	data, err := p.SerializeForExec()
	if err != nil {
		t.Fatalf("SerializeForExec failed: %v", err)
	}
	
	if len(data) == 0 {
		t.Error("serialized data is empty")
	}
	
	// Verify structure by reading the varint-encoded values
	buf := data
	firstVal, n := binary.Varint(buf)
	if n <= 0 {
		t.Fatal("failed to read first value")
	}
	buf = buf[n:]
	
	// Check if this is the multi-sequence marker
	if uint64(firstVal) == execInstrMultiSeq {
		// Read number of sequences
		numSeqs, n := binary.Varint(buf)
		if n <= 0 {
			t.Fatal("failed to read number of sequences")
		}
		if numSeqs != int64(len(p.Sequences)) {
			t.Errorf("expected %d sequences in serialization, got %d", len(p.Sequences), numSeqs)
		}
	} else {
		t.Errorf("expected multi-sequence marker %x, got %x", execInstrMultiSeq, uint64(firstVal))
	}
}

func TestAllCallsHelper(t *testing.T) {
	target := initTargetTest(t, "test", "64")
	
	// Test with legacy single sequence
	p1 := &Prog{
		Target: target,
		Calls:  []*Call{MakeCall(target.Syscalls[0], nil), MakeCall(target.Syscalls[1], nil)},
	}
	
	allCalls := p1.AllCalls()
	if len(allCalls) != 2 {
		t.Errorf("expected 2 calls, got %d", len(allCalls))
	}
	
	// Test with multi-sequence
	p2 := &Prog{
		Target: target,
		Sequences: [][]*Call{
			{MakeCall(target.Syscalls[0], nil)},
			{MakeCall(target.Syscalls[1], nil), MakeCall(target.Syscalls[2], nil)},
		},
	}
	
	allCalls = p2.AllCalls()
	if len(allCalls) != 3 {
		t.Errorf("expected 3 calls, got %d", len(allCalls))
	}
	
	if p2.NumSequences() != 2 {
		t.Errorf("expected 2 sequences, got %d", p2.NumSequences())
	}
}

func TestConcurrentMutation(t *testing.T) {
	target, rs, _ := initRandomTargetTest(t, "test", "64")
	ct := target.DefaultChoiceTable()
	
	// Generate a multi-sequence program
	p := target.GenerateConcurrent(rs, 15, 3, ct)
	
	if !p.HasSequences() {
		t.Skip("program doesn't have sequences")
	}
	
	// Try to mutate it
	r := newRand(target, rand.NewSource(rs.Int63()))
	p1 := p.Clone()
	p1.Mutate(r, 10, ct, nil, nil)
	
	// Check that it's still valid
	if err := p1.sanitize(false); err != nil {
		t.Errorf("mutated program is invalid: %v", err)
	}
}
