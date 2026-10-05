package gore

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type SoundFrontend interface {
	StopSound(channel int)
	SoundIsPlaying(channel int) bool
	UpdateSoundParams(channel, volume, sep int)
}

type MusicFrontend interface {
	RegisterSong(data []byte) uintptr
	UnregisterSong(handle uintptr)
	PlaySong(handle uintptr, looping bool)
	StopSong()
	PauseSong()
	ResumeSong()
	SetMusicVolume(volume int)
}

type StateStore interface {
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte) error
}

var stateStore StateStore = diskStateStore{}

func SetStateStore(store StateStore) {
	if store == nil {
		stateStore = diskStateStore{}
	} else {
		stateStore = store
	}
}

type diskStateStore struct{}

func (diskStateStore) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (diskStateStore) WriteFile(name string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(name), ".gore-state-*")
	if err != nil {
		return err
	}
	temp := file.Name()
	defer os.Remove(temp)
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(temp, name)
}

type saveBuffer struct {
	data []byte
	pos  int64
}

func (b *saveBuffer) Read(p []byte) (int, error) {
	if b.pos >= int64(len(b.data)) {
		return 0, io.EOF
	}
	n := copy(p, b.data[b.pos:])
	b.pos += int64(n)
	return n, nil
}

func (b *saveBuffer) Write(p []byte) (int, error) {
	end := b.pos + int64(len(p))
	if end > int64(len(b.data)) {
		b.data = append(b.data, make([]byte, end-int64(len(b.data)))...)
	}
	copy(b.data[b.pos:end], p)
	b.pos = end
	return len(p), nil
}

func (b *saveBuffer) Seek(offset int64, whence int) (int64, error) {
	pos := offset
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		pos += b.pos
	case io.SeekEnd:
		pos += int64(len(b.data))
	default:
		return b.pos, fmt.Errorf("invalid seek origin %d", whence)
	}
	if pos < 0 {
		return b.pos, fmt.Errorf("negative save offset")
	}
	b.pos = pos
	return pos, nil
}
