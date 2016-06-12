package aiff

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	MAGIC_FORM = "FORM"
	MAGIC_AIFF = "AIFF"
	MAGIC_COMT = "COMT"
	MAGIC_COMM = "COMM"
	MAGIC_CHAN = "CHAN"
	MAGIC_SSND = "SSND"
	MAGIC_LGWV = "LGWV"
	MAGIC_MARK = "MARK"
)

type AIFFReader interface {
	io.Reader
	io.ReaderAt
	io.Seeker
}

//
type Chunk struct {
	chunkID   [4]byte // ref: Magic
	chunkSize uint32  // big-endian
	data      *io.SectionReader
}

// FormChunk is Chunk with sub-chunks
type FormChunk struct {
	Chunk
	formType       [4]byte // Should be "AIFF"
	chunks         map[[4]byte]*Chunk
	commonChunk    *CommonChunk
	soundDataChunk *SoundDataChunk
}

type CommonChunk struct {
	Chunk
	numChannels     int16
	numSampleFrames uint32
	sampleSize      int16
	sampleRate      float80
}
type SoundDataChunk struct {
	Chunk
	offset    uint32 // should = 0
	blockSize uint32 // should = 0
	soundData []byte

	commonChunk *CommonChunk
}

type Reader struct {
	// r *riff.Reader
	AIFFReader

	// riffChunk *riff.RIFFChunk
	// format *WavFormat
	formChunk *FormChunk
	// *WavData
}

func NewReader(r AIFFReader) *Reader {
	return &Reader{r, nil}
}

func (fc *FormChunk) FindChunk(id string) (chunk *Chunk, err error) {
	var key [4]byte
	copy(key[:], []byte(id)[:4])
	chunk, ok := fc.chunks[key]
	if !ok {
		err = errors.New("chunk " + id + " not found")
		return
	}
	return
}

func (c *Chunk) ChunkID() string {
	return string(c.chunkID[:])
}

func (fc *FormChunk) CommonChunk() (chunk *CommonChunk) {
	if fc.commonChunk != nil {
		chunk = fc.commonChunk
	}
	chunk = &CommonChunk{}
	tmpChunk, _ := fc.FindChunk(MAGIC_COMM)
	chunk.Chunk = *tmpChunk

	binary.Read(chunk.data, binary.BigEndian, &chunk.numChannels)
	binary.Read(chunk.data, binary.BigEndian, &chunk.numSampleFrames)
	binary.Read(chunk.data, binary.BigEndian, &chunk.sampleSize)
	binary.Read(chunk.data, binary.BigEndian, &chunk.sampleRate)
	fmt.Println("Debug: Common Chunk numChannels", chunk.numChannels)
	fmt.Println("Debug: Common Chunk numSampleFrames", chunk.numSampleFrames)
	fmt.Println("Debug: Common Chunk sampleSize", chunk.sampleSize)
	fmt.Println("Debug: Common Chunk sampleRate", chunk.sampleRate.Float64())

	return
}

func (fc *FormChunk) SoundDataChunk() (chunk *SoundDataChunk) {
	if fc.soundDataChunk != nil {
		chunk = fc.soundDataChunk
		return
	}
	chunk = &SoundDataChunk{}
	chunk.commonChunk = fc.CommonChunk()
	tmpChunk, _ := fc.FindChunk(MAGIC_SSND)
	chunk.Chunk = *tmpChunk
	binary.Read(chunk.data, binary.BigEndian, &chunk.offset)
	binary.Read(chunk.data, binary.BigEndian, &chunk.blockSize)
	fmt.Println("Debug: Sound Data Chunk offset", chunk.offset)
	fmt.Println("Debug: Sound Data Chunk blockSize", chunk.blockSize)
	chunk.soundData = make([]byte, uint32(chunk.commonChunk.numChannels)*chunk.commonChunk.numSampleFrames*uint32(chunk.commonChunk.sampleSize)/8)
	chunk.data.Read(chunk.soundData)
	return
}

func (sc *SoundDataChunk) Sample(channel int, nFrame int) []byte {
	if nFrame >= int(sc.commonChunk.numSampleFrames) {
		return nil
	}
	byteNum := int(sc.commonChunk.sampleSize / 8)
	offset := (int(sc.commonChunk.numChannels)*int(byteNum))*nFrame + channel*byteNum
	return sc.soundData[offset : offset+byteNum]
}

func (r *Reader) FormChunk() (chunk *FormChunk) {
	if r.formChunk != nil {
		return r.formChunk
	}

	chunk = &FormChunk{}
	// magic := make([]byte, 4)
	r.Read(chunk.chunkID[:])
	if string(chunk.chunkID[:]) != MAGIC_FORM {
		panic("magic not equal \"FORM\"")
	}

	err := binary.Read(r, binary.BigEndian, &chunk.chunkSize)
	if err != nil {
		panic("read form chunk size error: " + err.Error())
	}
	fmt.Println("Debug: chunkSize ", chunk.chunkSize)

	r.Read(chunk.formType[:])
	if string(chunk.formType[:]) != MAGIC_AIFF {
		panic("magic not equal \"AIFF\"")
	}

	// var offset uint64
	offset, _ := r.Seek(0, os.SEEK_CUR)
	fmt.Println("Debug: Form Chunk offset", offset)

	chunk.chunks = make(map[[4]byte]*Chunk)

	for chunkOffset := uint32(0); chunkOffset < chunk.chunkSize-4; {
		subChunk := &Chunk{}
		fmt.Println("Debug: chunkOffset", chunkOffset)
		r.Seek(int64(chunkOffset)+(offset), os.SEEK_SET)
		r.Read(subChunk.chunkID[:])
		binary.Read(r, binary.BigEndian, &subChunk.chunkSize)

		fmt.Println("Debug: Get Sub Chunk", string(subChunk.chunkID[:]), "Chunk Size", subChunk.chunkSize)
		subChunk.data = io.NewSectionReader(r, int64(chunkOffset+8)+offset, int64(subChunk.chunkSize))
		chunkOffset += subChunk.chunkSize + 8
		chunk.chunks[subChunk.chunkID] = subChunk

	}
	r.formChunk = chunk
	return

}
