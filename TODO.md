# TODO

## Scripting behavior

- Remove "OrExit" functions and replace with a global variable that makes
  things call DefaultExitFn (we will have public wrapper functions that
  enforce the behavior so code can still call methods like "Write"
  without causing the underlying call to exit early)
- Add a global library that sets the DefaultExitFn and exit on error
  behavior for all brkit libraries

## conv

- Have HexArrayReaderFrom return an object that provides a method
  that can return that last C comment it read
- Rewrite CArrayToBlobs to support findComment function

## memory

- PointerMaker (maybe?): Optionally log pointers as they are created

## pattern

- Maybe use more capital letters in de bruijn string?
  (like this: "Aa0Aa1Aa2Aa3Aa"
  https://zerosum0x0.blogspot.com/2016/11/overflow-exploit-pattern-generator.html)
- Make de bruijn generation more efficient (less duplicate copies)

## process

- Consider adding read/write timeouts / deadlines
- Add sendLineAfter

## cmd/frag

- Maybe try searching for fragment again if it is not found by reversing
  endianness?
- Maybe look for multiple matches in pattern string instead of one?

## cmd/dasm

- Include C comments with output if they are available (refer to conv
  package TODOs in this file)

## cmd/stringer

- Add support for conv.HexArrayReaderFrom
