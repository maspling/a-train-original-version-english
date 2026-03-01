package pe

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"unicode/utf16"
)

const (
	ImageBase    = 0x00400000
	ReturnVA     = 0x00412554
	PatchFileOff = 0x0001254E
	SDIBIatVA    = 0x0042F028

	GDI32StrRVA = 0x0003516C
	USER32StrVA = 0x00434EE4

	FrameW = 640
	FrameH = 400

	WinNumX = 0x95
	WinNumY = 0x09

	WindowTitle = "A-Train Original Version 15th Anniversary Edition"
)

const (
	OScale       = 0x000
	OSDIBIat     = 0x004
	ONullIat     = 0x008
	OSDIBInt     = 0x00C
	ONullInt     = 0x010
	OHwnd        = 0x014
	OOrigWP      = 0x018
	OChromeW     = 0x01C
	OChromeH     = 0x020
	OHintName    = 0x024
	OSubmenu     = 0x034
	OChildHwnd   = 0x038
	OChildOrigWP = 0x03C
	OImports     = 0x040
	OGetDC       = 0x118
	OReleaseDC   = 0x11C
	OGDCName     = 0x120
	ORDCName     = 0x126
	OBlit        = 0x130
	OWndProc     = 0x2A0
	OChildWP     = 0x450
	OMenu        = 0x4C0
	ODialog      = 0x670
	OStrTbl      = 0x860
)

var IAT = map[string]uint32{
	"GetActiveWindow":  0x0042F44C,
	"GetClientRect":    0x0042F398,
	"GetWindowRect":    0x0042F3B0,
	"SetWindowLongA":   0x0042F3C4,
	"SetWindowPos":     0x0042F3C0,
	"GetMenu":          0x0042F360,
	"GetSubMenu":       0x0042F368,
	"CheckMenuItem":    0x0042F3F8,
	"EnableMenuItem":   0x0042F3FC,
	"CallWindowProcA":  0x0042F384,
	"GetWindow":        0x0042F39C,
	"GetModuleHandleA": 0x0042F258,
	"GetProcAddress":   0x0042F25C,
}

const (
	ResMenuDEFoff    = 0x39650
	ResDlgDEFoff     = 0x39660
	ResStr9DEFoff    = 0x39690
	ResStr3585DEFoff = 0x396a0
)

const (
	WinPatchOff = 0x69cc
	WinPatchLen = 58
)

const (
	KBTableFoff = 0x4744
)

func Patch(srcPath, outDir string) error {
	log.SetPrefix("[Patching EXE] ")
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	patched, err := patchEXE(data)
	if err != nil {
		return err
	}

	exeOut := filepath.Join(outDir, "A1Win.exe")
	if err := os.WriteFile(exeOut, patched, 0o755); err != nil {
		return err
	}
	return nil
}

func p32(val uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, val)
	return b
}

func s32(val int32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(val))
	return b
}

func alignUp(v, a uint32) uint32 {
	return (v + a - 1) & ^(a - 1)
}

func menuStr(s string) []byte {
	runes := []rune(s)
	u16 := utf16.Encode(runes)
	b := make([]byte, (len(u16)+1)*2)
	for i, v := range u16 {
		binary.LittleEndian.PutUint16(b[i*2:], v)
	}

	return b
}

func buildStrTbl(strings [16]string) []byte {
	var buf []byte
	for _, s := range strings {
		if s != "" {
			runes := []rune(s)
			u16 := utf16.Encode(runes)
			b := make([]byte, 2)
			binary.LittleEndian.PutUint16(b, uint16(len(runes)))
			buf = append(buf, b...)
			for _, v := range u16 {
				b := make([]byte, 2)
				binary.LittleEndian.PutUint16(b, v)
				buf = append(buf, b...)
			}
		} else {
			buf = append(buf, 0, 0)
		}
	}
	return buf
}

func u16(v uint16) []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, v)
	return b
}

func u32(v uint32) []byte {
	return p32(v)
}

func i16(v int16) []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, uint16(v))
	return b
}

type dialogItem struct {
	style  uint32
	x, y   int16
	cx, cy int16
	id     uint16
	cls    interface{}
	text   interface{}
}

func buildDialog(style uint32, cx, cy int16, title string, fontSize uint16, fontName string, items []dialogItem) []byte {
	var buf []byte
	buf = append(buf, u32(style)...)
	buf = append(buf, u32(0)...)
	buf = append(buf, u16(uint16(len(items)))...)
	buf = append(buf, i16(0)...)
	buf = append(buf, i16(0)...)
	buf = append(buf, i16(cx)...)
	buf = append(buf, i16(cy)...)
	buf = append(buf, u16(0)...)
	buf = append(buf, u16(0)...)
	buf = append(buf, menuStr(title)...)
	if style&0x40 != 0 {
		buf = append(buf, u16(fontSize)...)
		buf = append(buf, menuStr(fontName)...)
	}
	for _, item := range items {

		for len(buf)%4 != 0 {
			buf = append(buf, 0)
		}
		buf = append(buf, u32(item.style)...)
		buf = append(buf, u32(0)...)
		buf = append(buf, i16(item.x)...)
		buf = append(buf, i16(item.y)...)
		buf = append(buf, i16(item.cx)...)
		buf = append(buf, i16(item.cy)...)
		buf = append(buf, u16(item.id)...)
		switch cls := item.cls.(type) {
		case int:
			buf = append(buf, u16(0xFFFF)...)
			buf = append(buf, u16(uint16(cls))...)
		case string:
			buf = append(buf, menuStr(cls)...)
		}
		switch text := item.text.(type) {
		case int:
			buf = append(buf, u16(0xFFFF)...)
			buf = append(buf, u16(uint16(text))...)
		case string:
			buf = append(buf, menuStr(text)...)
		}
		buf = append(buf, u16(0)...)
	}
	return buf
}

func patchEXE(data []byte) ([]byte, error) {
	log.Println("Patching resolution scaling")
	d := make([]byte, len(data))
	copy(d, data)

	eLfanew := binary.LittleEndian.Uint32(d[0x3C:])
	ntOff := eLfanew
	optOff := ntOff + 24

	numSections := binary.LittleEndian.Uint16(d[ntOff+6:])
	sizeOfOpt := binary.LittleEndian.Uint16(d[ntOff+20:])
	sectAlign := binary.LittleEndian.Uint32(d[optOff+32:])
	fileAlign := binary.LittleEndian.Uint32(d[optOff+36:])

	sectTableOff := ntOff + 24 + uint32(sizeOfOpt)

	lastOff := sectTableOff + uint32(numSections-1)*40
	lastVA := binary.LittleEndian.Uint32(d[lastOff+12:])
	lastVSize := binary.LittleEndian.Uint32(d[lastOff+8:])
	lastRawOff := binary.LittleEndian.Uint32(d[lastOff+20:])
	lastRawSize := binary.LittleEndian.Uint32(d[lastOff+16:])

	newRVA := alignUp(lastVA+lastVSize, sectAlign)
	newRaw := alignUp(lastRawOff+lastRawSize, fileAlign)
	newRawSize := fileAlign
	newVSize := newRawSize
	caveVA := ImageBase + newRVA

	cave := make([]byte, newRawSize)

	cave[OScale] = 1

	hintNameRVA := newRVA + OHintName
	binary.LittleEndian.PutUint16(cave[OHintName:], 0)
	sname := []byte("StretchDIBits\x00")
	copy(cave[OHintName+2:], sname)

	binary.LittleEndian.PutUint32(cave[OSDIBIat:], hintNameRVA)
	binary.LittleEndian.PutUint32(cave[ONullIat:], 0)
	binary.LittleEndian.PutUint32(cave[OSDIBInt:], hintNameRVA)
	binary.LittleEndian.PutUint32(cave[ONullInt:], 0)

	copy(cave[OGDCName:], []byte("GetDC\x00"))
	copy(cave[ORDCName:], []byte("ReleaseDC\x00"))

	impRVA := binary.LittleEndian.Uint32(d[optOff+104:])
	impSize := binary.LittleEndian.Uint32(d[optOff+108:])
	impFoff := impRVA
	nOrig := impSize/20 - 1
	origDescs := make([]byte, nOrig*20)
	copy(origDescs, d[impFoff:impFoff+nOrig*20])

	var ourDesc [20]byte
	binary.LittleEndian.PutUint32(ourDesc[0:], newRVA+OSDIBInt)
	binary.LittleEndian.PutUint32(ourDesc[4:], 0)
	binary.LittleEndian.PutUint32(ourDesc[8:], 0xFFFFFFFF)
	binary.LittleEndian.PutUint32(ourDesc[12:], GDI32StrRVA)
	binary.LittleEndian.PutUint32(ourDesc[16:], newRVA+OSDIBIat)

	p := OImports
	copy(cave[p:], ourDesc[:])
	p += 20
	copy(cave[p:], origDescs)
	p += len(origDescs)

	p += 20
	impTotal := uint32(p - OImports)

	b := make([]byte, 0, 512)

	b = append(b, 0x83, 0x3D)
	b = append(b, p32(caveVA+OHwnd)...)
	b = append(b, 0x00)
	jneInit := len(b)
	b = append(b, 0x0F, 0x85, 0x00, 0x00, 0x00, 0x00)

	b = append(b, 0xFF, 0x15)
	b = append(b, p32(IAT["GetActiveWindow"])...)
	b = append(b, 0x85, 0xC0)
	jzInit := len(b)
	b = append(b, 0x0F, 0x84, 0x00, 0x00, 0x00, 0x00)

	b = append(b, 0xA3)
	b = append(b, p32(caveVA+OHwnd)...)

	b = append(b, 0x83, 0xEC, 0x20)
	b = append(b, 0x8D, 0x0C, 0x24)
	b = append(b, 0x51)
	b = append(b, 0xFF, 0x35)
	b = append(b, p32(caveVA+OHwnd)...)
	b = append(b, 0xFF, 0x15)
	b = append(b, p32(IAT["GetClientRect"])...)

	b = append(b, 0x8D, 0x4C, 0x24, 0x10)
	b = append(b, 0x51)
	b = append(b, 0xFF, 0x35)
	b = append(b, p32(caveVA+OHwnd)...)
	b = append(b, 0xFF, 0x15)
	b = append(b, p32(IAT["GetWindowRect"])...)

	b = append(b, 0x8B, 0x44, 0x24, 0x18)
	b = append(b, 0x2B, 0x44, 0x24, 0x10)
	b = append(b, 0x2B, 0x44, 0x24, 0x08)
	b = append(b, 0xA3)
	b = append(b, p32(caveVA+OChromeW)...)

	b = append(b, 0x8B, 0x44, 0x24, 0x1C)
	b = append(b, 0x2B, 0x44, 0x24, 0x14)
	b = append(b, 0x2B, 0x44, 0x24, 0x0C)
	b = append(b, 0xA3)
	b = append(b, p32(caveVA+OChromeH)...)

	b = append(b, 0x83, 0xC4, 0x20)

	b = append(b, 0x68)
	b = append(b, p32(caveVA+OWndProc)...)
	b = append(b, 0x6A, 0xFC)
	b = append(b, 0xFF, 0x35)
	b = append(b, p32(caveVA+OHwnd)...)
	b = append(b, 0xFF, 0x15)
	b = append(b, p32(IAT["SetWindowLongA"])...)
	b = append(b, 0xA3)
	b = append(b, p32(caveVA+OOrigWP)...)

	b = append(b, 0x6A, 0x05)
	b = append(b, 0xFF, 0x35)
	b = append(b, p32(caveVA+OHwnd)...)
	b = append(b, 0xFF, 0x15)
	b = append(b, p32(IAT["GetWindow"])...)
	b = append(b, 0x85, 0xC0)
	jzNochild := len(b)
	b = append(b, 0x74, 0x00)

	b = append(b, 0xA3)
	b = append(b, p32(caveVA+OChildHwnd)...)
	b = append(b, 0x68)
	b = append(b, p32(caveVA+OChildWP)...)
	b = append(b, 0x6A, 0xFC)
	b = append(b, 0x50)
	b = append(b, 0xFF, 0x15)
	b = append(b, p32(IAT["SetWindowLongA"])...)
	b = append(b, 0xA3)
	b = append(b, p32(caveVA+OChildOrigWP)...)

	lblNochild := len(b)
	b[jzNochild+1] = byte(lblNochild - (jzNochild + 2))

	b = append(b, 0x68)
	b = append(b, p32(USER32StrVA)...)
	b = append(b, 0xFF, 0x15)
	b = append(b, p32(IAT["GetModuleHandleA"])...)
	b = append(b, 0x50)

	b = append(b, 0x68)
	b = append(b, p32(caveVA+OGDCName)...)
	b = append(b, 0x50)
	b = append(b, 0xFF, 0x15)
	b = append(b, p32(IAT["GetProcAddress"])...)
	b = append(b, 0xA3)
	b = append(b, p32(caveVA+OGetDC)...)

	b = append(b, 0x58)
	b = append(b, 0x68)
	b = append(b, p32(caveVA+ORDCName)...)
	b = append(b, 0x50)
	b = append(b, 0xFF, 0x15)
	b = append(b, p32(IAT["GetProcAddress"])...)
	b = append(b, 0xA3)
	b = append(b, p32(caveVA+OReleaseDC)...)

	lblBlit := len(b)
	binary.LittleEndian.PutUint32(b[jneInit+2:], uint32(int32(lblBlit-(jneInit+6))))
	binary.LittleEndian.PutUint32(b[jzInit+2:], uint32(int32(lblBlit-(jzInit+6))))

	b = append(b, 0x80, 0x3D)
	b = append(b, p32(caveVA+OScale)...)
	b = append(b, 0x01)
	jneScaled := len(b)
	b = append(b, 0x0F, 0x85, 0x00, 0x00, 0x00, 0x00)

	b = append(b, 0x68)
	b = append(b, p32(ReturnVA)...)
	b = append(b, 0xFF, 0x25)
	b = append(b, p32(SDIBIatVA)...)

	lblScaled := len(b)
	binary.LittleEndian.PutUint32(b[jneScaled+2:], uint32(int32(lblScaled-(jneScaled+6))))

	b = append(b, 0x53, 0x56, 0x57)

	b = append(b, 0x0F, 0xB6, 0x1D)
	b = append(b, p32(caveVA+OScale)...)
	b = append(b, 0x8B, 0x74, 0x24, 0x30)
	b = append(b, 0x8B, 0x7C, 0x24, 0x34)

	b = append(b, 0xFF, 0x35)
	b = append(b, p32(caveVA+OChildHwnd)...)
	b = append(b, 0xFF, 0x15)
	b = append(b, p32(caveVA+OGetDC)...)
	b = append(b, 0x50)

	b = append(b, 0x68)
	b = append(b, p32(0x00CC0020)...)
	b = append(b, 0x6A, 0x00)
	b = append(b, 0x57)
	b = append(b, 0x56)
	b = append(b, 0x68)
	b = append(b, p32(FrameH)...)
	b = append(b, 0x68)
	b = append(b, p32(FrameW)...)
	b = append(b, 0x6A, 0x00)
	b = append(b, 0x6A, 0x00)

	b = append(b, 0x69, 0xC3)
	b = append(b, p32(FrameH)...)
	b = append(b, 0x50)

	b = append(b, 0x69, 0xC3)
	b = append(b, p32(FrameW)...)
	b = append(b, 0x50)

	b = append(b, 0x6A, 0x00)
	b = append(b, 0x6A, 0x00)
	b = append(b, 0xFF, 0x74, 0x24, 0x30)
	b = append(b, 0xFF, 0x15)
	b = append(b, p32(caveVA+OSDIBIat)...)

	b = append(b, 0x58)
	b = append(b, 0x50)
	b = append(b, 0xFF, 0x35)
	b = append(b, p32(caveVA+OChildHwnd)...)
	b = append(b, 0xFF, 0x15)
	b = append(b, p32(caveVA+OReleaseDC)...)

	b = append(b, 0x5F, 0x5E, 0x5B)
	b = append(b, 0x83, 0xC4, 0x30)

	jmpVA := caveVA + uint32(OBlit+len(b))
	jmpRel := int32(ReturnVA) - int32(jmpVA+5)
	b = append(b, 0xE9)
	b = append(b, s32(jmpRel)...)

	blitSize := len(b)
	if blitSize > OWndProc-OBlit {
		return nil, fmt.Errorf("blit_hook overflows into wndproc")
	}
	copy(cave[OBlit:], b)

	w := make([]byte, 0, 512)

	w = append(w, 0x81, 0x7C, 0x24, 0x08)
	w = append(w, p32(0x111)...)
	jCmd := len(w)
	w = append(w, 0x0F, 0x84, 0x00, 0x00, 0x00, 0x00)

	w = append(w, 0x81, 0x7C, 0x24, 0x08)
	w = append(w, p32(0x117)...)
	jImp := len(w)
	w = append(w, 0x0F, 0x84, 0x00, 0x00, 0x00, 0x00)

	lblPt := len(w)

	w = append(w, 0xFF, 0x74, 0x24, 0x10)
	w = append(w, 0xFF, 0x74, 0x24, 0x10)
	w = append(w, 0xFF, 0x74, 0x24, 0x10)
	w = append(w, 0xFF, 0x74, 0x24, 0x10)
	w = append(w, 0xFF, 0x35)
	w = append(w, p32(caveVA+OOrigWP)...)
	w = append(w, 0xFF, 0x15)
	w = append(w, p32(IAT["CallWindowProcA"])...)
	w = append(w, 0xC2, 0x10, 0x00)

	lblCmd := len(w)
	binary.LittleEndian.PutUint32(w[jCmd+2:], uint32(int32(lblCmd-(jCmd+6))))

	w = append(w, 0x8B, 0x44, 0x24, 0x0C)
	w = append(w, 0x25)
	w = append(w, p32(0xFFFF)...)

	w = append(w, 0x3D)
	w = append(w, p32(0x7D00)...)
	jCe1 := len(w)
	w = append(w, 0x0F, 0x82, 0x00, 0x00, 0x00, 0x00)

	w = append(w, 0x3D)
	w = append(w, p32(0x7D03)...)
	jCe2 := len(w)
	w = append(w, 0x0F, 0x87, 0x00, 0x00, 0x00, 0x00)

	w = append(w, 0x2D)
	w = append(w, p32(0x7D00)...)
	w = append(w, 0x40)
	w = append(w, 0xA2)
	w = append(w, p32(caveVA+OScale)...)

	w = append(w, 0x53, 0x56, 0x57)

	w = append(w, 0x0F, 0xB6, 0x3D)
	w = append(w, p32(caveVA+OScale)...)

	w = append(w, 0x83, 0x3D)
	w = append(w, p32(caveVA+OSubmenu)...)
	w = append(w, 0x00)
	jHs := len(w)
	w = append(w, 0x0F, 0x85, 0x00, 0x00, 0x00, 0x00)

	w = append(w, 0xFF, 0x35)
	w = append(w, p32(caveVA+OHwnd)...)
	w = append(w, 0xFF, 0x15)
	w = append(w, p32(IAT["GetMenu"])...)
	w = append(w, 0x6A, 0x01)
	w = append(w, 0x50)
	w = append(w, 0xFF, 0x15)
	w = append(w, p32(IAT["GetSubMenu"])...)
	w = append(w, 0xA3)
	w = append(w, p32(caveVA+OSubmenu)...)

	lblHs := len(w)
	binary.LittleEndian.PutUint32(w[jHs+2:], uint32(int32(lblHs-(jHs+6))))

	w = append(w, 0x8B, 0x35)
	w = append(w, p32(caveVA+OSubmenu)...)
	w = append(w, 0xBB)
	w = append(w, p32(0x7D00)...)

	lblLoop := len(w)

	w = append(w, 0x89, 0xD8)
	w = append(w, 0x2D)
	w = append(w, p32(0x7CFF)...)
	w = append(w, 0x31, 0xC9)
	w = append(w, 0x39, 0xF8)
	w = append(w, 0x75, 0x05)
	w = append(w, 0xB9)
	w = append(w, p32(8)...)

	w = append(w, 0x51)
	w = append(w, 0x53)
	w = append(w, 0x56)
	w = append(w, 0xFF, 0x15)
	w = append(w, p32(IAT["CheckMenuItem"])...)
	w = append(w, 0x43)
	w = append(w, 0x81, 0xFB)
	w = append(w, p32(0x7D04)...)
	relBack := byte(lblLoop - (len(w) + 2))
	w = append(w, 0x72, relBack)

	w = append(w, 0x69, 0xCF)
	w = append(w, p32(FrameW)...)
	w = append(w, 0x03, 0x0D)
	w = append(w, p32(caveVA+OChromeW)...)
	w = append(w, 0x69, 0xD7)
	w = append(w, p32(FrameH)...)
	w = append(w, 0x03, 0x15)
	w = append(w, p32(caveVA+OChromeH)...)

	w = append(w, 0x6A, 0x06)
	w = append(w, 0x52)
	w = append(w, 0x51)
	w = append(w, 0x6A, 0x00)
	w = append(w, 0x6A, 0x00)
	w = append(w, 0x6A, 0x00)
	w = append(w, 0xFF, 0x35)
	w = append(w, p32(caveVA+OHwnd)...)
	w = append(w, 0xFF, 0x15)
	w = append(w, p32(IAT["SetWindowPos"])...)

	w = append(w, 0x5F, 0x5E, 0x5B)
	w = append(w, 0x31, 0xC0)
	w = append(w, 0xC2, 0x10, 0x00)

	lblEdit := len(w)
	binary.LittleEndian.PutUint32(w[jCe1+2:], uint32(int32(lblEdit-(jCe1+6))))
	binary.LittleEndian.PutUint32(w[jCe2+2:], uint32(int32(lblEdit-(jCe2+6))))

	w = append(w, 0x3D)
	w = append(w, p32(0x7D10)...)
	jPtEdit := len(w)
	w = append(w, 0x0F, 0x85, 0x00, 0x00, 0x00, 0x00)
	binary.LittleEndian.PutUint32(w[jPtEdit+2:], uint32(int32(lblPt-(jPtEdit+6))))

	w = append(w, 0x51, 0x52)
	callWPos := len(w)
	callVA := caveVA + uint32(OWndProc+callWPos)
	callRel := int32(0x0042c67b) - int32(callVA+5)
	w = append(w, 0xE8)
	w = append(w, s32(callRel)...)
	w = append(w, 0x8B, 0x40, 0x04)
	w = append(w, 0x8B, 0x80)
	w = append(w, p32(0xC4)...)

	w = append(w, 0x80, 0xB8)
	w = append(w, p32(0xC66)...)
	w = append(w, 0x00)
	jneExit := len(w)
	w = append(w, 0x75, 0x00)

	w = append(w, 0x66, 0xC7, 0x80)
	w = append(w, p32(0xCB2)...)
	w = append(w, u16(0x5B)...)
	w = append(w, 0x66, 0xC7, 0x80)
	w = append(w, p32(0xCB4)...)
	w = append(w, u16(0x5C)...)
	w = append(w, 0x66, 0xC7, 0x80)
	w = append(w, p32(0xB98)...)
	w = append(w, u16(0x5D)...)
	jmpDone := len(w)
	w = append(w, 0xEB, 0x00)

	lblExit := len(w)
	w[jneExit+1] = byte(lblExit - (jneExit + 2))
	w = append(w, 0x66, 0xC7, 0x80)
	w = append(w, p32(0xB98)...)
	w = append(w, u16(0x1B)...)

	lblDone := len(w)
	w[jmpDone+1] = byte(lblDone - (jmpDone + 2))

	w = append(w, 0x5A, 0x59)
	w = append(w, 0x31, 0xC0)
	w = append(w, 0xC2, 0x10, 0x00)

	lblImp := len(w)
	binary.LittleEndian.PutUint32(w[jImp+2:], uint32(int32(lblImp-(jImp+6))))

	w = append(w, 0xFF, 0x74, 0x24, 0x10)
	w = append(w, 0xFF, 0x74, 0x24, 0x10)
	w = append(w, 0xFF, 0x74, 0x24, 0x10)
	w = append(w, 0xFF, 0x74, 0x24, 0x10)
	w = append(w, 0xFF, 0x35)
	w = append(w, p32(caveVA+OOrigWP)...)
	w = append(w, 0xFF, 0x15)
	w = append(w, p32(IAT["CallWindowProcA"])...)

	w = append(w, 0x50, 0x53, 0x56)

	w = append(w, 0xFF, 0x74, 0x24, 0x10)
	w = append(w, 0xFF, 0x15)
	w = append(w, p32(IAT["GetMenu"])...)
	w = append(w, 0x89, 0xC6)

	w = append(w, 0xBB)
	w = append(w, p32(0x7D00)...)

	lblEn := len(w)
	w = append(w, 0x6A, 0x00)
	w = append(w, 0x53)
	w = append(w, 0x56)
	w = append(w, 0xFF, 0x15)
	w = append(w, p32(IAT["EnableMenuItem"])...)
	w = append(w, 0x43)
	w = append(w, 0x81, 0xFB)
	w = append(w, p32(0x7D04)...)
	relBack2 := byte(lblEn - (len(w) + 2))
	w = append(w, 0x72, relBack2)

	w = append(w, 0x6A, 0x00)
	w = append(w, 0x68)
	w = append(w, p32(0x7D10)...)
	w = append(w, 0x56)
	w = append(w, 0xFF, 0x15)
	w = append(w, p32(IAT["EnableMenuItem"])...)

	w = append(w, 0x5E, 0x5B, 0x58)
	w = append(w, 0xC2, 0x10, 0x00)

	wpSize := len(w)
	if wpSize > OChildWP-OWndProc {
		return nil, fmt.Errorf("frame wndproc overflows into child wndproc")
	}
	copy(cave[OWndProc:], w)

	c := make([]byte, 0, 128)

	c = append(c, 0x80, 0x3D)
	c = append(c, p32(caveVA+OScale)...)
	c = append(c, 0x01)
	jCpt := len(c)
	c = append(c, 0x74, 0x00)

	c = append(c, 0x8B, 0x44, 0x24, 0x08)
	c = append(c, 0x2D)
	c = append(c, p32(0x200)...)
	c = append(c, 0x3D)
	c = append(c, p32(0x09)...)
	jCpt2 := len(c)
	c = append(c, 0x77, 0x00)

	c = append(c, 0x51, 0x52)
	c = append(c, 0x0F, 0xB6, 0x0D)
	c = append(c, p32(caveVA+OScale)...)

	c = append(c, 0x8B, 0x44, 0x24, 0x18)
	c = append(c, 0x0F, 0xB7, 0xC0)
	c = append(c, 0x31, 0xD2)
	c = append(c, 0xF7, 0xF1)
	c = append(c, 0x50)

	c = append(c, 0x8B, 0x44, 0x24, 0x1C)
	c = append(c, 0xC1, 0xE8, 0x10)
	c = append(c, 0x31, 0xD2)
	c = append(c, 0xF7, 0xF1)

	c = append(c, 0xC1, 0xE0, 0x10)
	c = append(c, 0x5A)
	c = append(c, 0x09, 0xD0)
	c = append(c, 0x89, 0x44, 0x24, 0x18)
	c = append(c, 0x5A, 0x59)

	lblCpt := len(c)
	c[jCpt+1] = byte(lblCpt - (jCpt + 2))
	c[jCpt2+1] = byte(lblCpt - (jCpt2 + 2))

	c = append(c, 0xFF, 0x74, 0x24, 0x10)
	c = append(c, 0xFF, 0x74, 0x24, 0x10)
	c = append(c, 0xFF, 0x74, 0x24, 0x10)
	c = append(c, 0xFF, 0x74, 0x24, 0x10)
	c = append(c, 0xFF, 0x35)
	c = append(c, p32(caveVA+OChildOrigWP)...)
	c = append(c, 0xFF, 0x15)
	c = append(c, p32(IAT["CallWindowProcA"])...)
	c = append(c, 0xC2, 0x10, 0x00)

	cwpSize := len(c)
	if cwpSize > OMenu-OChildWP {
		return nil, fmt.Errorf("child wndproc overflows into menu")
	}
	copy(cave[OChildWP:], c)

	log.Println("Patching new menu bar")
	var newMenu []byte

	newMenu = append(newMenu, u16(0)...)
	newMenu = append(newMenu, u16(0)...)

	newMenu = append(newMenu, u16(0x0010)...)
	newMenu = append(newMenu, menuStr("&File")...)

	newMenu = append(newMenu, u16(0x0000)...)
	newMenu = append(newMenu, u16(0x7D10)...)
	newMenu = append(newMenu, menuStr("&Edit Mode")...)

	newMenu = append(newMenu, u16(0x0080)...)
	newMenu = append(newMenu, u16(0xE141)...)
	newMenu = append(newMenu, menuStr("&Quit to Desktop ")...)

	newMenu = append(newMenu, u16(0x0010)...)
	newMenu = append(newMenu, menuStr("&Resolution")...)

	newMenu = append(newMenu, u16(0x0008)...)
	newMenu = append(newMenu, u16(0x7D00)...)
	newMenu = append(newMenu, menuStr("640x400 (1x)")...)

	newMenu = append(newMenu, u16(0x0000)...)
	newMenu = append(newMenu, u16(0x7D01)...)
	newMenu = append(newMenu, menuStr("1280x800 (2x)")...)

	newMenu = append(newMenu, u16(0x0000)...)
	newMenu = append(newMenu, u16(0x7D02)...)
	newMenu = append(newMenu, menuStr("1920x1200 (3x)")...)

	newMenu = append(newMenu, u16(0x0080)...)
	newMenu = append(newMenu, u16(0x7D03)...)
	newMenu = append(newMenu, menuStr("2560x1600 (4x)")...)

	newMenu = append(newMenu, u16(0x0090)...)
	newMenu = append(newMenu, menuStr("&Help")...)

	newMenu = append(newMenu, u16(0x0000)...)
	newMenu = append(newMenu, u16(57670)...)
	newMenu = append(newMenu, menuStr("&Help")...)

	newMenu = append(newMenu, u16(0x0080)...)
	newMenu = append(newMenu, u16(57664)...)
	newMenu = append(newMenu, menuStr("&About")...)

	if OMenu+len(newMenu) > ODialog {
		return nil, fmt.Errorf("menu overflows into dialog")
	}
	copy(cave[OMenu:], newMenu)

	log.Println("Patching new about dialog and window title")
	aboutDlg := buildDialog(
		0x80C800C0,
		310, 55,
		"A-Train Original Version",
		9,
		"\uFF2D\uFF33 \uFF30\u30B4\u30B7\u30C3\u30AF",
		[]dialogItem{
			{style: 0x50000003, x: 11, y: 17, cx: 20, cy: 20,
				id: 0xFFFF, cls: 0x82, text: 0x80},
			{style: 0x50020080, x: 40, y: 10, cx: 210, cy: 8,
				id: 0xFFFF, cls: 0x82,
				text: "A-Train Original Version 15th Anniversary Edition"},
			{style: 0x50020000, x: 40, y: 25, cx: 210, cy: 8,
				id: 0xFFFF, cls: 0x82,
				text: "(C) 1994,2000 ARTDINK. All Rights Reserved."},
			{style: 0x50020000, x: 40, y: 37, cx: 210, cy: 8,
				id: 0xFFFF, cls: 0x82,
				text: "(C) 2026 English Translation by Yugge"},
			{style: 0x50030001, x: 253, y: 7, cx: 50, cy: 14,
				id: 1, cls: 0x80, text: "OK"},
		},
	)

	if ODialog+len(aboutDlg) > OStrTbl {
		return nil, fmt.Errorf("dialog overflows into string table")
	}
	copy(cave[ODialog:], aboutDlg)

	var strs128 [16]string
	strs128[0] = WindowTitle
	strtbl128 := buildStrTbl(strs128)

	var strs57344 [16]string
	strs57344[0] = WindowTitle
	strs57344[1] = "Ready"
	strtbl57344 := buildStrTbl(strs57344)

	pStr := OStrTbl
	copy(cave[pStr:], strtbl128)
	oStrtbl128 := pStr
	pStr += len(strtbl128)

	copy(cave[pStr:], strtbl57344)
	oStrtbl57344 := pStr
	pStr += len(strtbl57344)

	if pStr > int(newRawSize) {
		return nil, fmt.Errorf("string tables overflow section")
	}

	newSectOff := sectTableOff + uint32(numSections)*40
	headersSize := binary.LittleEndian.Uint32(d[optOff+60:])
	if newSectOff+40 > headersSize {
		return nil, fmt.Errorf("no room for new section header in PE headers")
	}

	copy(d[newSectOff:], []byte(".scale\x00\x00"))
	binary.LittleEndian.PutUint32(d[newSectOff+8:], newVSize)
	binary.LittleEndian.PutUint32(d[newSectOff+12:], newRVA)
	binary.LittleEndian.PutUint32(d[newSectOff+16:], newRawSize)
	binary.LittleEndian.PutUint32(d[newSectOff+20:], newRaw)
	binary.LittleEndian.PutUint32(d[newSectOff+24:], 0)
	binary.LittleEndian.PutUint32(d[newSectOff+28:], 0)
	binary.LittleEndian.PutUint16(d[newSectOff+32:], 0)
	binary.LittleEndian.PutUint16(d[newSectOff+34:], 0)
	binary.LittleEndian.PutUint32(d[newSectOff+36:], 0xE0000060)

	binary.LittleEndian.PutUint16(d[ntOff+6:], numSections+1)

	newSOI := alignUp(newRVA+newVSize, sectAlign)
	binary.LittleEndian.PutUint32(d[optOff+56:], newSOI)

	binary.LittleEndian.PutUint32(d[optOff+104:], newRVA+OImports)
	binary.LittleEndian.PutUint32(d[optOff+108:], impTotal)

	jmpTarget := caveVA + OBlit
	rel32 := int32(jmpTarget) - int32(0x0041254E+5)
	d[PatchFileOff] = 0xE9
	binary.LittleEndian.PutUint32(d[PatchFileOff+1:], uint32(rel32))
	d[PatchFileOff+5] = 0x90

	log.Println("Patch Scenario number texture location")
	winCode := make([]byte, 0, WinPatchLen)
	winCode = append(winCode, 0x6A, 0x0F)
	winCode = append(winCode, 0x8B, 0x8E, 0x94, 0x0C, 0x00, 0x00)
	winCode = append(winCode, 0x51)
	winCode = append(winCode, 0x6A, 0x0F)
	winCode = append(winCode, 0x8D, 0x57, byte(WinNumY&0xFF))
	winCode = append(winCode, 0x52)
	winCode = append(winCode, 0x8D, 0x8D)
	winCode = append(winCode, s32(int32(WinNumX))...)
	winCode = append(winCode, 0x0F, 0xBF, 0xD0)
	winCode = append(winCode, 0x83, 0xEA, 0x20)
	winCode = append(winCode, 0x51)
	winCode = append(winCode, 0x52)
	for len(winCode) < WinPatchLen {
		winCode = append(winCode, 0x90)
	}
	copy(d[WinPatchOff:], winCode)

	d[KBTableFoff] = 18
	d[KBTableFoff+1] = 19
	d[KBTableFoff+2] = 20
	log.Println("Changed edit mode cheat code to 1,2,3")

	binary.LittleEndian.PutUint32(d[ResMenuDEFoff:], newRVA+uint32(OMenu))
	binary.LittleEndian.PutUint32(d[ResMenuDEFoff+4:], uint32(len(newMenu)))

	binary.LittleEndian.PutUint32(d[ResDlgDEFoff:], newRVA+uint32(ODialog))
	binary.LittleEndian.PutUint32(d[ResDlgDEFoff+4:], uint32(len(aboutDlg)))

	binary.LittleEndian.PutUint32(d[ResStr9DEFoff:], newRVA+uint32(oStrtbl128))
	binary.LittleEndian.PutUint32(d[ResStr9DEFoff+4:], uint32(len(strtbl128)))

	binary.LittleEndian.PutUint32(d[ResStr3585DEFoff:], newRVA+uint32(oStrtbl57344))
	binary.LittleEndian.PutUint32(d[ResStr3585DEFoff+4:], uint32(len(strtbl57344)))

	needed := int(newRaw + newRawSize)
	if len(d) < needed {
		d = append(d, make([]byte, needed-len(d))...)
	}
	copy(d[newRaw:], cave)

	d = d[:needed]

	return d, nil
}
