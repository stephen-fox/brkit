package glibckit

// IO_FILE is used by glibc to store state for files.
// This struct is useful for implementing FSOP (file
// stream oriented programming).
//
// This struct was generated using glibc 2.31.
type IO_FILE struct {
	Flags          uint64
	IO_read_ptr    uint64 // *byte
	IO_read_end    uint64 // *byte
	IO_read_base   uint64 // *byte
	IO_write_base  uint64 // *byte
	IO_write_ptr   uint64 // *byte
	IO_write_end   uint64 // *byte
	IO_buf_base    uint64 // *byte
	IO_buf_end     uint64 // *byte
	IO_save_base   uint64 // *byte
	IO_backup_base uint64 // *byte
	IO_save_end    uint64 // *byte
	Markers        uint64 // *_IO_marker
	Chain          uint64 // *_IO_FILE
	Fileno         int32
	Flags2         int32
	Old_offset     uint64 // __off_t
	Cur_column     uint16
	Vtable_offset  int8
	Shortbuf       [1]byte
	Unknown1       int32
	Lock           uint64 // unsafe.Pointer
	Offset         uint64 // __off64_t
	Codecvt        uint64 // *_IO_codecvt
	Wide_data      uint64 // *_IO_wide_data
	// Unclear if these fields are needed.
	//Freeres_list uint64 // *_IO_FILE
	//Freeres_buf  uint64 // unsafe.Pointer
	//Pad5         uint64 // size_t
	//Mode         int32
	Unused2 [80]byte
}
