package main

import (
	"encoding/binary"
	"os"
)

var (
	HEADER_BYTES = []byte{'F', 'L', 'V', 0x01, 0x05, 0x00, 0x00, 0x00, 0x09, 0x00, 0x00, 0x00, 0x00,
		0x12, 0x00, 0x00, 0x28, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 11
		0x02, 0x00, 0x0a, 0x6f, 0x6e, 0x4d, 0x65, 0x74, 0x61, 0x44, 0x61, 0x74, 0x61, // 13
		0x08, 0x00, 0x00, 0x00, 0x01, // 5
		0x00, 0x08, 0x64, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6F, 0x6E, // 10
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 9
		0x00, 0x00, 0x09, // 3
		0x00, 0x00, 0x00, 0x33}
)

const (
	DURATION_OFFSET = 53
	HEADER_LEN      = 13
)

type File struct {
	file              *os.File
	name              string
	readOnly          bool
	size              int64
	headerBuf         []byte
	duration          float64
	lastTimestamp     uint32
	firstTimestampSet bool
	firstTimestamp    uint32
}

type TagHeader struct {
	TagType   byte
	DataSize  uint32
	Timestamp uint32
}

func CreateFile(name string) (flvFile *File, err error) {
	var file *os.File
	// Create file
	if file, err = os.Create(name); err != nil {
		return
	}
	// Write flv header
	if _, err = file.Write(HEADER_BYTES); err != nil {
		file.Close()
		return
	}

	// Sync to disk
	if err = file.Sync(); err != nil {
		file.Close()
		return
	}

	flvFile = &File{
		file:      file,
		name:      name,
		readOnly:  false,
		headerBuf: make([]byte, 11),
		duration:  0.0,
	}

	return
}

func (flvFile *File) Close() {
	flvFile.file.Close()
}

//Write tag bytes
func (flvFile *File) WriteTagDirect(tag []byte) (err error) {
	if _, err = flvFile.file.Write(tag); err != nil {
		return
	}

	// Write previous tag size
	if err = binary.Write(flvFile.file, binary.BigEndian, uint32(len(tag))); err != nil {
		return
	}
	return nil
}

func (flvFile *File) SetDuration(duration float64) {
	flvFile.duration = duration
}

func (flvFile *File) Sync() (err error) {
	// Update duration on MetaData
	if _, err = flvFile.file.Seek(DURATION_OFFSET, 0); err != nil {
		return
	}
	if err = binary.Write(flvFile.file, binary.BigEndian, flvFile.duration); err != nil {
		return
	}
	if _, err = flvFile.file.Seek(0, 2); err != nil {
		return
	}

	err = flvFile.file.Sync()
	return
}
func (flvFile *File) Size() (size int64) {
	size = flvFile.size
	return
}

func (flvFile *File) IsFinished() bool {
	pos, err := flvFile.file.Seek(0, 1)
	return (err != nil) || (pos >= flvFile.size)
}
func (flvFile *File) LoopBack() {
	flvFile.file.Seek(HEADER_LEN, 0)
}
func (flvFile *File) FilePath() string {
	return flvFile.name
}
