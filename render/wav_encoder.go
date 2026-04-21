package render

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/tetrackt/tetrackt/audio"
)

// EncodeWAV writes a standard RIFF/WAV file with 16-bit signed PCM stereo data to w.
func EncodeWAV(w io.Writer, sampleRate audio.SampleRate, frames [][2]float64) error {
	const numChannels = 2
	const bitsPerSample = 16
	const bytesPerSample = 2

	numSamples := len(frames)
	dataSize := uint32(numSamples * numChannels * bytesPerSample)
	fileSize := 36 + dataSize

	var hdr bytes.Buffer
	hdr.WriteString("RIFF")
	binary.Write(&hdr, binary.LittleEndian, fileSize)
	hdr.WriteString("WAVE")
	hdr.WriteString("fmt ")
	binary.Write(&hdr, binary.LittleEndian, uint32(16))
	binary.Write(&hdr, binary.LittleEndian, uint16(1))
	binary.Write(&hdr, binary.LittleEndian, uint16(numChannels))
	binary.Write(&hdr, binary.LittleEndian, uint32(sampleRate))
	binary.Write(&hdr, binary.LittleEndian, uint32(int(sampleRate)*numChannels*bytesPerSample))
	binary.Write(&hdr, binary.LittleEndian, uint16(numChannels*bytesPerSample))
	binary.Write(&hdr, binary.LittleEndian, uint16(bitsPerSample))
	hdr.WriteString("data")
	binary.Write(&hdr, binary.LittleEndian, dataSize)

	if _, err := w.Write(hdr.Bytes()); err != nil {
		return err
	}

	buf := make([]byte, numSamples*numChannels*bytesPerSample)
	for i, frame := range frames {
		l := floatToInt16(frame[0])
		r := floatToInt16(frame[1])
		off := i * 4
		buf[off+0] = byte(l)
		buf[off+1] = byte(uint16(l) >> 8)
		buf[off+2] = byte(r)
		buf[off+3] = byte(uint16(r) >> 8)
	}
	_, err := w.Write(buf)
	return err
}
