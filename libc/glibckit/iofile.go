package glibckit

// _IO_FILE is used by glibc to store state for files.
// This struct is useful for implementing FSOP (file
// stream oriented programming).
//
// This struct was generated using glibc 2.31.
type _IO_FILE struct {
	_flags          uint64
	_IO_read_ptr    uint64 // *byte
	_IO_read_end    uint64 // *byte
	_IO_read_base   uint64 // *byte
	_IO_write_base  uint64 // *byte
	_IO_write_ptr   uint64 // *byte
	_IO_write_end   uint64 // *byte
	_IO_buf_base    uint64 // *byte
	_IO_buf_end     uint64 // *byte
	_IO_save_base   uint64 // *byte
	_IO_backup_base uint64 // *byte
	_IO_save_end    uint64 // *byte
	_markers        uint64 // *_IO_marker
	_chain          uint64 // *_IO_FILE
	_fileno         int32
	_flags2         int32
	_old_offset     uint64 // __off_t
	_cur_column     uint16
	_vtable_offset  int8
	_shortbuf       [1]byte
	unknown1        int32
	_lock           uint64 // unsafe.Pointer
	_offset         uint64 // __off64_t
	_codecvt        uint64 // *_IO_codecvt
	_wide_data      uint64 // *_IO_wide_data
	// Unclear if these fields are needed.
	//_freeres_list   uint64 // *_IO_FILE
	//_freeres_buf    uint64 // unsafe.Pointer
	//__pad5          uint64 // size_t
	//_mode           int32
	_unused2 [80]byte
}
