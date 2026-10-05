package gore

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type freedoomFrame struct {
	Frame   int
	Tick    int32
	Hash    string
	X, Y, Z int32
	Angle   uint32
	Health  int32
}

type freedoomReplay struct {
	frames []freedoomFrame
	limit  int
	save   bool
	load   bool
	move   bool
	event  int
}

func (r *freedoomReplay) DrawFrame(img *image.RGBA) {
	if len(r.frames) > 0 && r.frames[len(r.frames)-1].Tick == gametic {
		return
	}
	h := sha256.Sum256(img.Pix)
	f := freedoomFrame{Frame: len(r.frames), Tick: gametic, Hash: hex.EncodeToString(h[:])}
	p := players[consoleplayer].Fmo
	if p != nil {
		f.X = p.Fx
		f.Y = p.Fy
		f.Z = p.Fz
		f.Angle = p.Fangle
		f.Health = p.Fhealth
	}
	r.frames = append(r.frames, f)
	if r.save && len(r.frames) == 20 {
		g_SaveGame(0, "Freedoom replay")
	}
	if r.load && len(r.frames) == 60 {
		g_LoadGame(p_SaveGameFile(0))
	}
	if len(r.frames) >= r.limit {
		if path := os.Getenv("GORE_REPLAY_PNG"); path != "" {
			file, err := os.Create(path)
			if err != nil {
				panic(err)
			}
			if err = png.Encode(file, img); err != nil {
				panic(err)
			}
			file.Close()
		}
		Stop()
	}
}
func (*freedoomReplay) SetTitle(string) {}
func (r *freedoomReplay) GetEvent(e *DoomEvent) bool {
	if !r.move {
		return false
	}
	events := []struct {
		frame int
		kind  Evtype_t
		key   uint8
	}{
		{5, Ev_keydown, KEY_UPARROW1}, {25, Ev_keyup, KEY_UPARROW1},
		{30, Ev_keydown, KEY_RIGHTARROW1}, {40, Ev_keyup, KEY_RIGHTARROW1},
		{45, Ev_keydown, KEY_FIRE1}, {55, Ev_keyup, KEY_FIRE1},
		{65, Ev_keydown, KEY_USE1}, {66, Ev_keyup, KEY_USE1},
	}
	if r.event >= len(events) || len(r.frames) < events[r.event].frame {
		return false
	}
	next := events[r.event]
	r.event++
	e.Type = next.kind
	e.Key = next.key
	return true
}
func (*freedoomReplay) CacheSound(string, []byte)       {}
func (*freedoomReplay) PlaySound(string, int, int, int) {}

func TestFreedoomReplay(t *testing.T) {
	wad := os.Getenv("GORE_REPLAY_WAD")
	if wad == "" {
		t.Skip("set GORE_REPLAY_WAD to run the Freedoom replay")
	}
	limit := 60
	if n, e := strconv.Atoi(os.Getenv("GORE_REPLAY_FRAMES")); e == nil && n > 0 {
		limit = n
	}
	dg_run_full_speed = true
	SetVirtualFileSystem(os.DirFS(filepath.Dir(wad)))
	args := []string{"-iwad", filepath.Base(wad), "-warp"}
	args = append(args, strings.Fields(os.Getenv("GORE_REPLAY_MAP"))...)
	r := &freedoomReplay{limit: limit, save: os.Getenv("GORE_REPLAY_SAVE") == "1", load: os.Getenv("GORE_REPLAY_LOAD") == "1", move: os.Getenv("GORE_REPLAY_MOVE") == "1"}
	if load := os.Getenv("GORE_REPLAY_LOAD_SLOT"); load != "" {
		args = append(args, "-loadgame", load)
	}
	Run(r, args)
	if len(r.frames) == 0 {
		t.Fatal("no frames")
	}
	b, e := json.MarshalIndent(r.frames, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	if out := os.Getenv("GORE_REPLAY_OUTPUT"); out != "" {
		if e = os.WriteFile(out, b, 0644); e != nil {
			t.Fatal(e)
		}
	}
	t.Logf("%d frames, last: %+v", len(r.frames), r.frames[len(r.frames)-1])
}
