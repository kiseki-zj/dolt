package dolt

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const maxMapSize int = 0xFFFFFFFF

type DB struct {
	path    string
	file    *os.File
	dataref []byte
	data    *[maxMapSize]byte
	datasz  int
}

func Open(path string, mode os.FileMode) (*DB, error) {
	var db = &DB{}
	flag := os.O_RDWR
	db.path = path
	var err error
	if db.file, err = os.OpenFile(path, flag|os.O_CREATE, mode); err != nil {
		_ = db.close()
		return nil, err
	}
	if err := db.mmap(0x10000); err != nil {
		_ = db.close()
		return nil, err
	}
	return db, nil
}

func (db *DB) mmap(minsz int) error {

	info, _ := db.file.Stat()
	var size = int(info.Size())
	if size < minsz {
		size = minsz
	}
	if err := db.munmap(); err != nil {
		return err
	}

	b, err := syscall.Mmap(int(db.file.Fd()), 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return err
	}
	if err := madvise(b, syscall.MADV_RANDOM); err != nil {
		return fmt.Errorf("madvise: %s", err)
	}
	db.dataref = b
	db.data = (*[maxMapSize]byte)(unsafe.Pointer(&b[0]))
	db.datasz = size
	return nil
}

func madvise(b []byte, advice int) (err error) {
	_, _, e1 := syscall.Syscall(syscall.SYS_MADVISE, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)), uintptr(advice))
	if e1 != 0 {
		err = e1
	}
	return
}
func (db *DB) munmap() error {
	if db.dataref == nil {
		return nil
	}
	err := syscall.Munmap(db.dataref)
	db.dataref = nil
	db.data = nil
	db.datasz = 0
	return err
}
func (db *DB) close() error {
	if err := db.munmap(); err != nil {
		return err
	}
	if err := db.file.Close(); err != nil {
		return fmt.Errorf("db file close: %s", err)
	}
	db.file = nil
	db.path = ""
	return nil
}

func _assert(condition bool, msg string, v ...interface{}) {
	if !condition {
		panic(fmt.Sprintf("assertion failed: "+msg, v...))
	}
}
