// Copyright 2015 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package prog

import (
	"math/rand"
)

// Generate generates a random program with ncalls calls.
// ct contains a set of allowed syscalls, if nil all syscalls are used.
func (target *Target) Generate(rs rand.Source, ncalls int, ct *ChoiceTable) *Prog {
	p := &Prog{
		Target: target,
	}
	r := newRand(target, rs)
	s := newState(target, ct, nil)
	for len(p.Calls) < ncalls {
		calls := r.generateCall(s, p, len(p.Calls))
		for _, c := range calls {
			s.analyze(c)
			p.Calls = append(p.Calls, c)
		}
	}
	// For the last generated call we could get additional calls that create
	// resources and overflow ncalls. Remove some of these calls.
	// The resources in the last call will be replaced with the default values,
	// which is exactly what we want.
	for len(p.Calls) > ncalls {
		p.RemoveCall(ncalls - 1)
	}
	p.sanitizeFix()
	p.debugValidate()
	return p
}

// GenerateConcurrent generates a random program with multiple sequences for concurrent execution.
// Each sequence will have approximately ncalls/numSequences calls.
// ct contains a set of allowed syscalls, if nil all syscalls are used.
func (target *Target) GenerateConcurrent(rs rand.Source, ncalls int, numSequences int, ct *ChoiceTable) *Prog {
	if numSequences <= 1 {
		return target.Generate(rs, ncalls, ct)
	}
	
	p := &Prog{
		Target:    target,
		Sequences: make([][]*Call, numSequences),
	}
	r := newRand(target, rs)
	
	// Distribute calls across sequences
	callsPerSeq := ncalls / numSequences
	remainder := ncalls % numSequences
	
	for seqIdx := 0; seqIdx < numSequences; seqIdx++ {
		s := newState(target, ct, nil)
		seqLen := callsPerSeq
		if seqIdx < remainder {
			seqLen++
		}
		
		// Create a temporary prog for generating this sequence
		tempProg := &Prog{
			Target: target,
			Calls:  []*Call{},
		}
		
		for len(tempProg.Calls) < seqLen {
			generatedCalls := r.generateCall(s, tempProg, len(tempProg.Calls))
			for _, c := range generatedCalls {
				s.analyze(c)
				tempProg.Calls = append(tempProg.Calls, c)
			}
		}
		
		// Trim excess calls
		if len(tempProg.Calls) > seqLen {
			tempProg.Calls = tempProg.Calls[:seqLen]
		}
		
		p.Sequences[seqIdx] = tempProg.Calls
	}
	
	p.sanitizeFix()
	p.debugValidate()
	return p
}
