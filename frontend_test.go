package gore

import (
	"bytes"
	"image"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type recordingFrontend struct {
	channel int
	volume  int
	sep     int
	playing bool
	song    uintptr
	loop    bool
	paused  bool
}

func (*recordingFrontend) DrawFrame(*image.RGBA)     {}
func (*recordingFrontend) SetTitle(string)           {}
func (*recordingFrontend) GetEvent(*DoomEvent) bool  { return false }
func (*recordingFrontend) CacheSound(string, []byte) {}
func (f *recordingFrontend) PlaySound(_ string, channel, volume, sep int) {
	f.channel = channel
	f.volume = volume
	f.sep = sep
	f.playing = true
}
func (f *recordingFrontend) StopSound(channel int) {
	if channel == f.channel {
		f.playing = false
	}
}
func (f *recordingFrontend) SoundIsPlaying(channel int) bool {
	return channel == f.channel && f.playing
}
func (f *recordingFrontend) UpdateSoundParams(channel, volume, sep int) {
	f.channel = channel
	f.volume = volume
	f.sep = sep
}
func (*recordingFrontend) RegisterSong([]byte) uintptr          { return 7 }
func (f *recordingFrontend) UnregisterSong(uintptr)             { f.song = 0 }
func (f *recordingFrontend) PlaySong(handle uintptr, loop bool) { f.song = handle; f.loop = loop }
func (f *recordingFrontend) StopSong()                          { f.song = 0 }
func (f *recordingFrontend) PauseSong()                         { f.paused = true }
func (f *recordingFrontend) ResumeSong()                        { f.paused = false }
func (f *recordingFrontend) SetMusicVolume(volume int)          { f.volume = volume }

func TestOptionalAudioFrontend(t *testing.T) {
	old := dg_frontend
	defer func() { dg_frontend = old }()
	f := &recordingFrontend{}
	dg_frontend = f
	handle := i_StartSound(&sfxinfo_t{Fname: "pistol"}, 3, 100, 128)
	if handle != 3 || i_SoundIsPlaying(handle) == 0 {
		t.Fatal("lost sound channel")
	}
	i_UpdateSoundParams(handle, 80, 32)
	if f.volume != 80 || f.sep != 32 {
		t.Fatal("lost sound parameters")
	}
	i_StopSound(handle)
	if i_SoundIsPlaying(handle) != 0 {
		t.Fatal("sound still playing")
	}
	song := i_RegisterSong([]byte("MThd"))
	i_PlaySong(song, 1)
	if f.song != 7 || !f.loop {
		t.Fatal("lost song handle")
	}
	i_PauseSong()
	if !f.paused {
		t.Fatal("song not paused")
	}
	i_ResumeSong()
	if f.paused {
		t.Fatal("song still paused")
	}
	i_SetMusicVolume(8)
	if f.volume != 8 {
		t.Fatal("lost music volume")
	}
	i_StopSong()
	i_UnRegisterSong(song)
}

func TestSaveBuffer(t *testing.T) {
	b := &saveBuffer{}
	b.Write([]byte{1, 2, 3})
	b.Seek(1, io.SeekStart)
	b.Write([]byte{4})
	b.Seek(0, io.SeekStart)
	data, err := io.ReadAll(b)
	if err != nil || !bytes.Equal(data, []byte{1, 4, 3}) {
		t.Fatalf("read %v: %v", data, err)
	}
	if _, err = b.Seek(-1, io.SeekStart); err == nil {
		t.Fatal("accepted negative offset")
	}
}

func TestDiskStateStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dgsave0.dsg")
	store := diskStateStore{}
	for _, data := range [][]byte{{1, 2, 3}, {4, 5}} {
		if err := store.WriteFile(path, data); err != nil {
			t.Fatal(err)
		}
		got, err := store.ReadFile(path)
		if err != nil || !bytes.Equal(got, data) {
			t.Fatalf("read %v: %v", got, err)
		}
	}
	files, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(files) != 1 {
		t.Fatalf("temporary files remain: %v, %v", files, err)
	}
}
