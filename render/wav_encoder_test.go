package render

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/tetrackt/tetrackt/audio"
	"github.com/youpy/go-wav"
)

func TestEncodeWAV(t *testing.T) {
	// Create test data - simple stereo sine wave-like data
	const sampleRate = audio.SampleRate(44100)
	frames := [][2]float64{
		{0.0, 0.0},      // silence
		{0.5, -0.5},     // moderate levels
		{1.0, -1.0},     // full scale
		{-1.0, 1.0},     // full scale inverted
		{0.25, -0.25},   // quarter scale
	}

	var buf bytes.Buffer
	err := EncodeWAV(&buf, sampleRate, frames)
	if err != nil {
		t.Fatalf("EncodeWAV failed: %v", err)
	}

	// Verify we got some data
	if buf.Len() == 0 {
		t.Fatal("No data written to buffer")
	}

	// Write to a temp file and read back (youpy/go-wav needs file-like reader)
	tmpFile, err := os.CreateTemp("", "test_wav_*.wav")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write the buffer to file
	_, err = io.Copy(tmpFile, &buf)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Reopen for reading
	file, err := os.Open(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to reopen temp file: %v", err)
	}
	defer file.Close()

	// Try to read back the WAV file to verify it's valid
	reader := wav.NewReader(file)
	
	format, err := reader.Format()
	if err != nil {
		t.Fatalf("Failed to read WAV format: %v", err)
	}

	// Verify format parameters
	if format.AudioFormat != wav.AudioFormatPCM {
		t.Errorf("Expected PCM format (%d), got %d", wav.AudioFormatPCM, format.AudioFormat)
	}
	if format.NumChannels != 2 {
		t.Errorf("Expected 2 channels, got %d", format.NumChannels)
	}
	if format.SampleRate != uint32(sampleRate) {
		t.Errorf("Expected sample rate %d, got %d", sampleRate, format.SampleRate)
	}
	if format.BitsPerSample != 16 {
		t.Errorf("Expected 16 bits per sample, got %d", format.BitsPerSample)
	}

	// Read back samples and verify they match our input (within quantization tolerance)
	samples, err := reader.ReadSamples()
	if err != nil {
		t.Fatalf("Failed to read samples: %v", err)
	}

	if len(samples) != len(frames) {
		t.Errorf("Expected %d samples, got %d", len(frames), len(samples))
	}

	// Verify the samples are approximately correct (accounting for int16 quantization)
	for i, sample := range samples {
		if i >= len(frames) {
			break
		}
		expectedL := floatToInt16(frames[i][0])
		expectedR := floatToInt16(frames[i][1])
		
		if int16(sample.Values[0]) != expectedL {
			t.Errorf("Sample %d left channel: expected %d, got %d", i, expectedL, sample.Values[0])
		}
		if int16(sample.Values[1]) != expectedR {
			t.Errorf("Sample %d right channel: expected %d, got %d", i, expectedR, sample.Values[1])
		}
	}
}

func TestEncodeWAVEmpty(t *testing.T) {
	// Test with empty frames
	frames := [][2]float64{}
	
	var buf bytes.Buffer
	err := EncodeWAV(&buf, 44100, frames)
	if err != nil {
		t.Fatalf("EncodeWAV with empty frames failed: %v", err)
	}

	// Should still produce a valid WAV file with just header
	if buf.Len() == 0 {
		t.Fatal("No data written for empty WAV")
	}

	// Write to a temp file and read back
	tmpFile, err := os.CreateTemp("", "test_empty_wav_*.wav")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, &buf)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	file, err := os.Open(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to reopen temp file: %v", err)
	}
	defer file.Close()

	// Verify it's readable
	reader := wav.NewReader(file)
	format, err := reader.Format()
	if err != nil {
		t.Fatalf("Failed to read empty WAV format: %v", err)
	}

	if format.NumChannels != 2 {
		t.Errorf("Expected 2 channels, got %d", format.NumChannels)
	}
}