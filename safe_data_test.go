package gore

import (
	"encoding/binary"
	"testing"
)

func TestPatchColumns(t *testing.T) {
	data := []byte{1, 0, 3, 0, 0, 0, 0, 0, 12, 0, 0, 0, 0, 3, 0, 17, 18, 19, 0, 255}
	patch := readPatch(data)
	column := patch.GetColumn(0)
	if patch.Fwidth != 1 || patch.Fheight != 3 || string(column.Data()) != string([]byte{17, 18, 19}) || column.Next().Ftopdelta != 255 {
		t.Fatalf("invalid decoded patch: %+v, %+v", patch, column)
	}
}

func TestInvalidLumpData(t *testing.T) {
	for name, run := range map[string]func(){
		"header":    func() { readPatch([]byte{1}) },
		"column":    func() { readColumn([]byte{0, 10, 0}, 0) },
		"offset":    func() { dataRange([]byte{1}, -1, 1) },
		"size":      func() { dataRange([]byte{1}, 0, 2) },
		"records":   func() { decodeRecords[mapvertex_t]([]byte{1}) },
		"directory": func() { textureCount([]byte{255, 255, 255, 127}) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if _, ok := recover().(engineError); !ok {
					t.Fatal("expected engine error")
				}
			}()
			run()
		})
	}
}

func TestMapRecordSizes(t *testing.T) {
	for _, item := range []struct {
		value any
		size  int
	}{
		{mapvertex_t{}, 4}, {mapsidedef_t{}, 30}, {maplinedef_t{}, 14},
		{mapsector_t{}, 26}, {mapsubsector_t{}, 4}, {mapseg_t{}, 12},
		{mapnode_t{}, 28}, {mapthing_t{}, 10}, {filelump_t{}, 16},
	} {
		if got := binary.Size(item.value); got != item.size {
			t.Errorf("%T: got %d, want %d", item.value, got, item.size)
		}
	}
}

func TestObjectIndexBackingChanges(t *testing.T) {
	var table map[*mapvertex_t]int32
	first := make([]mapvertex_t, 2)
	second := make([]mapvertex_t, 2)
	if objectIndex(&first[1], first, &table) != 1 || objectIndex(&second[0], second, &table) != 0 {
		t.Fatal("incorrect object index")
	}
	if _, ok := table[&first[0]]; ok {
		t.Fatal("retained stale object")
	}
}

func TestSavedThinkerPresence(t *testing.T) {
	for _, present := range []bool{false, true} {
		save_stream = &saveBuffer{}
		var before thinker_func_t
		if present {
			before = savedThinker{}
		}
		saveg_write_actionf_t(&before)
		if len(save_stream.data) != 4 {
			t.Fatal("changed legacy pointer width")
		}
		save_stream.pos = 0
		var after thinker_func_t
		saveg_read_actionf_t(&after)
		if (after != nil) != present {
			t.Fatal("lost thinker presence")
		}
	}
}

func TestInterceptOverflow(t *testing.T) {
	interceptsOverrun(MAXINTERCEPTS_ORIGINAL, &intercept_t{})
	defer func() {
		failure, ok := recover().(engineError)
		if !ok || failure.Error() != "unsupported vanilla intercept pointer overflow" {
			t.Fatalf("unexpected error: %v", failure)
		}
	}()
	interceptsOverrun(MAXINTERCEPTS_ORIGINAL+1, &intercept_t{})
}
