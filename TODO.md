# TODO

## Scripting behavior

- Remove "OrExit" functions and replace with a global variable that makes
  things call DefaultExitFn (we will have public wrapper functions that
  enforce the behavior so code can still call methods like "Write"
  without causing the underlying call to exit early)
- Add a global library that sets the DefaultExitFn and exit on error
  behavior for all brkit libraries

## iokit

- Add TrimEnd (or similarly named) method to remove n bytes from end

## pattern

- Maybe use more capital letters in de bruijn string?
  (like this: "Aa0Aa1Aa2Aa3Aa"
  https://zerosum0x0.blogspot.com/2016/11/overflow-exploit-pattern-generator.html)
- Make de bruijn generation more efficient (less duplicate copies)

## process

- Add "Ctx" functions (i.e., FromIOCtx) where the first argument takes
  a `context.Context`. Use the Context.Done channel to make the process
  "exit" by closing IO
- Consider adding read/write timeouts / deadlines

## cmd/frag

- Maybe try searching for fragment again if it is not found by reversing
  endianness?
- Maybe look for multiple matches in pattern string instead of one?
