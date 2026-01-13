// Copyright 2024 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

// syz-concurrentgen generates programs with multiple concurrent sequences for testing race conditions.
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/google/syzkaller/prog"
	_ "github.com/google/syzkaller/sys"
)

var (
	flagOS         = flag.String("os", "linux", "target OS")
	flagArch       = flag.String("arch", "amd64", "target architecture")
	flagCalls      = flag.Int("calls", 20, "number of calls to generate")
	flagSequences  = flag.Int("sequences", 3, "number of concurrent sequences")
	flagCount      = flag.Int("count", 1, "number of programs to generate")
)

func main() {
	flag.Parse()

	target, err := prog.GetTarget(*flagOS, *flagArch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get target: %v\n", err)
		os.Exit(1)
	}

	rs := rand.NewSource(time.Now().UnixNano())
	ct := target.DefaultChoiceTable()

	for i := 0; i < *flagCount; i++ {
		p := target.GenerateConcurrent(rs, *flagCalls, *flagSequences, ct)
		
		fmt.Printf("# Program %d\n", i+1)
		fmt.Printf("# Sequences: %d, Total calls: %d\n", p.NumSequences(), len(p.AllCalls()))
		
		if p.HasSequences() {
			for seqIdx, seq := range p.Sequences {
				fmt.Printf("\n# Sequence %d (%d calls):\n", seqIdx+1, len(seq))
				for _, c := range seq {
					fmt.Printf("%s\n", c.Meta.Name)
				}
			}
		} else {
			fmt.Printf("\n# Single sequence (%d calls):\n", len(p.Calls))
			for _, c := range p.Calls {
				fmt.Printf("%s\n", c.Meta.Name)
			}
		}
		
		// Serialize to show it can be executed
		data, err := p.SerializeForExec()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to serialize: %v\n", err)
		} else {
			fmt.Printf("\n# Serialized size: %d bytes\n", len(data))
		}
		
		fmt.Println("\n" + string(p.Serialize()))
		fmt.Println("---")
	}
}
