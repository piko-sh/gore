package gore

import "encoding/binary"

type byteView struct {
	data   []byte
	offset int
}

func (v byteView) At(i int) byte {
	index := v.offset + i
	if index >= len(v.data) && index < len(v.data)+128 {
		return 0
	}
	return v.data[index]
}

func dataRange(data []byte, offset, size int) []byte {
	if offset < 0 || size < 0 || offset > len(data) || size > len(data)-offset {
		i_Error("invalid lump range: offset %d, size %d, length %d", offset, size, len(data))
	}
	return data[offset : offset+size]
}

func readInt32(data []byte, offset int) int32 {
	return int32(binary.LittleEndian.Uint32(dataRange(data, offset, 4)))
}

func decodeRecord(data []byte, value any) {
	if _, err := binary.Decode(data, binary.LittleEndian, value); err != nil {
		i_Error("invalid lump record: %v", err)
	}
}

func decodeRecords[T any](data []byte) []T {
	var value T
	size := binary.Size(value)
	if size <= 0 || len(data)%size != 0 {
		i_Error("invalid lump record size %d: length %d", size, len(data))
	}
	values := make([]T, len(data)/size)
	decodeRecord(data, values)
	return values
}

var patchCache map[int32]*patch_t

func readPatch(data []byte) *patch_t {
	header := dataRange(data, 0, 8)
	p := &patch_t{
		Fwidth:      int16(binary.LittleEndian.Uint16(header)),
		Fheight:     int16(binary.LittleEndian.Uint16(header[2:])),
		Fleftoffset: int16(binary.LittleEndian.Uint16(header[4:])),
		Ftopoffset:  int16(binary.LittleEndian.Uint16(header[6:])),
		data:        data,
	}
	if p.Fwidth < 0 || p.Fheight < 0 {
		i_Error("invalid patch dimensions")
	}
	p.Fcolumnofs = decodeRecords[int32](dataRange(data, 8, int(p.Fwidth)*4))
	return p
}

func readColumn(data []byte, offset int) *column_t {
	c := &column_t{data: data, offset: offset, Ftopdelta: dataRange(data, offset, 1)[0]}
	if c.Ftopdelta != 255 {
		c.Flength = dataRange(data, offset+1, 1)[0]
		dataRange(data, offset, int(c.Flength)+4)
	}
	return c
}

func textureCount(data []byte) int32 {
	count := readInt32(data, 0)
	if count < 0 || int64(count)*4 > int64(len(data)-4) {
		i_Error("invalid texture directory")
	}
	return count
}

func readMapTexture(data []byte, offset int) *maptexture_t {
	header := dataRange(data, offset, 22)
	m := &maptexture_t{
		Fmasked:     readInt32(header, 8),
		Fwidth:      int16(binary.LittleEndian.Uint16(header[12:])),
		Fheight:     int16(binary.LittleEndian.Uint16(header[14:])),
		Fobsolete:   readInt32(header, 16),
		Fpatchcount: int16(binary.LittleEndian.Uint16(header[20:])),
	}
	copy(m.Fname[:], header[:8])
	if m.Fwidth <= 0 || m.Fheight <= 0 || m.Fpatchcount < 0 {
		i_Error("invalid texture dimensions")
	}
	m.Fpatches = decodeRecords[mappatch_t](dataRange(data, offset+22, int(m.Fpatchcount)*10))
	return m
}

func screenClip(clip []int16, base int32) []int16 {
	if base == 0 {
		return clip
	}
	result := make([]int16, SCREENWIDTH)
	copy(result[base:], clip)
	return result
}

func objectIndex[T any](p *T, values []T, table *map[*T]int32) int32 {
	if len(values) == 0 {
		i_Error("empty object table")
	}
	_, current := (*table)[&values[0]]
	if !current || len(*table) != len(values) {
		*table = make(map[*T]int32, len(values))
		for i := range values {
			(*table)[&values[i]] = int32(i)
		}
	}
	index, ok := (*table)[p]
	if !ok {
		i_Error("object outside table")
	}
	return index
}

type savedThinker struct{}

func (savedThinker) ThinkerFunc() {}

func savedPointer(present bool) uintptr {
	if present {
		return 1
	}
	return 0
}
