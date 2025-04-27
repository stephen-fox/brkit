package linuxkit

// SigcontextX86_64 is the struct that is read by sigreturn(2) on
// x86 64-bit systems. It is useful for manipulating CPU registers
// using SROP (Sigreturn-oriented programming).
type SigcontextX86_64 struct {
	IDK1    uint64
	IDK2    uint64
	IDK3    uint64
	IDK4    uint64
	IDK5    uint64
	R8      uint64
	R9      uint64
	R10     uint64
	R11     uint64
	R12     uint64
	R13     uint64
	R14     uint64
	R15     uint64
	RDI     uint64
	RSI     uint64
	RBP     uint64
	RBX     uint64
	RDX     uint64
	RAX     uint64
	RCX     uint64
	RSP     uint64
	RIP     uint64
	EFLAGS  uint64
	CS      uint16
	GS      uint16
	FS      uint16
	PAD0    uint16
	ERR     uint64
	TRAPNO  uint64
	OLDMASK uint64
	CR2     uint64
	Fpstate uint64
	RES0    uint64
	RES1    uint64
	RES2    uint64
	RES3    uint64
	RES4    uint64
	RES5    uint64
	RES6    uint64
	RES7    uint64
}
