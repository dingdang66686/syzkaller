# Concurrent Execution Mechanism

This document describes the concurrent execution mechanism in syzkaller for testing race conditions and concurrent bugs.

## Overview

Syzkaller now supports programs with multiple sequences that execute concurrently. This feature allows testing of race conditions and bugs that only manifest when multiple system calls are executed simultaneously.

## Program Structure

A program can now contain multiple sequences of system calls:
- **Single sequence (legacy)**: A traditional program with calls executed sequentially
- **Multi-sequence**: A program with multiple sequences, each executed in a separate thread concurrently

### Data Structure

```go
type Prog struct {
    Target    *Target
    Calls     []*Call     // Legacy single sequence
    Sequences [][]*Call   // Multi-sequence format
    // ... other fields
}
```

When `Sequences` is non-empty, the executor will spawn threads to execute each sequence concurrently. The `Calls` field is ignored when `Sequences` is populated.

## Generating Concurrent Programs

Use the `GenerateConcurrent` function to generate programs with multiple sequences:

```go
// Generate a program with 20 calls distributed across 3 sequences
p := target.GenerateConcurrent(rs, 20, 3, ct)
```

Parameters:
- `rs`: Random source
- `ncalls`: Total number of calls to generate
- `numSequences`: Number of concurrent sequences
- `ct`: Choice table for syscall selection

## Serialization Format

Multi-sequence programs use a special serialization format:

```
<MULTISEQ_MARKER> <num_sequences> 
<num_calls_seq1> <calls...> <SEQSEP>
<num_calls_seq2> <calls...> <SEQSEP>
...
<num_calls_seqN> <calls...>
<EOF>
```

The format includes:
- `MULTISEQ_MARKER`: Special marker (`execInstrMultiSeq`) identifying multi-sequence programs
- `SEQSEP`: Separator between sequences (`execInstrSeqSep`)
- `EOF`: End of program marker

Legacy single-sequence programs maintain backward compatibility with the original format.

## Executor Behavior

When the executor receives a multi-sequence program:

1. Detects the `MULTISEQ_MARKER` at the start of the program
2. Reads the number of sequences
3. For each sequence:
   - Parses the sequence calls
   - Schedules calls using the existing thread pool
   - Executes sequences concurrently

The executor uses the existing threading infrastructure (`flag_threaded` mode) to achieve concurrency. Each sequence's calls are scheduled through the same thread pool used for async calls.

## Helper Methods

The `Prog` struct provides several helper methods for working with sequences:

```go
// Check if program uses multi-sequence format
if p.HasSequences() {
    // Work with p.Sequences
}

// Get all calls regardless of format
allCalls := p.AllCalls()

// Get number of sequences (1 for legacy programs)
numSeq := p.NumSequences()
```

## Example Tool

The `syz-concurrentgen` tool generates example concurrent programs:

```bash
# Generate a program with 15 calls across 3 sequences
./bin/syz-concurrentgen -calls 15 -sequences 3 -count 1

# Generate 5 programs with 20 calls across 4 sequences each
./bin/syz-concurrentgen -calls 20 -sequences 4 -count 5
```

## Use Cases

The concurrent execution mechanism is particularly useful for:

1. **Race condition testing**: Execute related syscalls simultaneously to expose race conditions
2. **Concurrency bugs**: Test synchronization issues in kernel subsystems
3. **Lock ordering**: Discover deadlocks and lock ordering problems
4. **Resource contention**: Test behavior under concurrent resource access

## Implementation Notes

- The implementation maintains backward compatibility with existing single-sequence programs
- Multi-sequence programs only execute concurrently when `flag_threaded` is enabled
- Each sequence is independent and doesn't share dependencies with other sequences during generation
- The executor's existing thread pool limits the maximum concurrency (kMaxThreads = 32)

## Future Enhancements

Potential improvements include:
- Resource sharing between sequences for more realistic race condition testing
- Synchronization points to control execution order
- Sequence priorities to control relative execution timing
- Integration with coverage-guided fuzzing to find interesting concurrent schedules
