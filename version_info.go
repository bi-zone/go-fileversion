package fileversion

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/xerrors"
)

// FixedFileInfo contains info from VS_FIXEDFILEINFO windows structure.
type FixedFileInfo struct {
	FileMajor      uint16
	FileMinor      uint16
	FileBuild      uint16
	FilePrivate    uint16
	ProductMajor   uint16
	ProductMinor   uint16
	ProductBuild   uint16
	ProductPrivate uint16
	FileFlagsMask  uint32
	FileFlags      uint32
	FileOs         uint32
	FileType       uint32
	FileSubType    uint32
	FileDateMS     uint32
	FileDateLS     uint32
}

type Locale string

var EnglishUSAscii = Locale("040904E4") // US English + CP_USASCII
var EnglishUnicode = Locale("040904B0") // US English + CP_UNICODE
var EnglishUnknown = Locale("04090000") // US English + unknown codepage

// Info contains windows object, which is being used
// for getting file version properties.
type Info struct {
	data          []byte
	FixedFileInfo FixedFileInfo
	pageID        Locale
}

var (
	version                    = syscall.NewLazyDLL("version.dll")
	getFileVersionInfoSizeProc = version.NewProc("GetFileVersionInfoSizeW")
	getFileVersionInfoProc     = version.NewProc("GetFileVersionInfoW")
	verQueryValueProc          = version.NewProc("VerQueryValueW")
)

// CompanyName returns CompanyName property.
func (f Info) CompanyName() string {
	return f.GetProperty("CompanyName")
}

// FileDescription returns FileDescription property.
func (f Info) FileDescription() string {
	return f.GetProperty("FileDescription")
}

// FileVersion returns FileVersion property.
func (f Info) FileVersion() string {
	return f.GetProperty("FileVersion")
}

// InternalName returns InternalName property.
func (f Info) InternalName() string {
	return f.GetProperty("InternalName")
}

// LegalCopyright returns LegalCopyright property.
func (f Info) LegalCopyright() string {
	return f.GetProperty("LegalCopyright")
}

// OriginalFilename returns OriginalFilename property.
func (f Info) OriginalFilename() string {
	return f.GetProperty("OriginalFilename")
}

// ProductName returns ProductName property.
func (f Info) ProductName() string {
	return f.GetProperty("ProductName")
}

// ProductVersion returns ProductVersion property.
func (f Info) ProductVersion() string {
	return f.GetProperty("ProductVersion")
}

// Comments returns Comments property.
func (f Info) Comments() string {
	return f.GetProperty("Comments")
}

// FileVeLegalTrademarksrsion returns FileVeLegalTrademarksrsion property.
func (f Info) FileVeLegalTrademarksrsion() string {
	return f.GetProperty("LegalTrademarks")
}

// PrivateBuild returns PrivateBuild property.
func (f Info) PrivateBuild() string {
	return f.GetProperty("PrivateBuild")
}

// SpecialBuild returns SpecialBuild property.
func (f Info) SpecialBuild() string {
	return f.GetProperty("SpecialBuild")
}

// New creates Info instance with default locale.
func New(path string) (Info, error) {
	info, err := newWithoutLocale(path)
	if err != nil {
		return Info{}, xerrors.Errorf("failed to get VersionInfo; %s", err)
	}

	pageID := info.GetPageID()
	info.pageID = pageID

	return info, nil
}

// NewWithLocale creates Info instance with user-defined locale.
func NewWithLocale(path string, pageID Locale) (Info, error) {
	info, err := newWithoutLocale(path)
	if err != nil {
		return Info{}, xerrors.Errorf("failed to get VersionInfo; %s", err)
	}

	info.pageID = pageID

	return info, nil
}

func newWithoutLocale(path string) (Info, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return Info{}, xerrors.Errorf("failed to convert image path to utf16; %s", err)
	}
	size, _, err := getFileVersionInfoSizeProc.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0,
	)
	if size == 0 {
		return Info{}, xerrors.Errorf("failed to get memory size for VersionInfo slice; %s", err)
	}
	info := make([]byte, size)
	ret, _, err := getFileVersionInfoProc.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0,
		uintptr(len(info)),
		uintptr(unsafe.Pointer(&info[0])),
	)
	if ret == 0 {
		return Info{}, xerrors.Errorf("failed to get VersionInfo from windows; %s", err)
	}

	vi := Info{data: info}

	fixedInfo := vi.GetFixedInfo()
	vi.FixedFileInfo = fixedInfo

	return vi, nil
}

// GetPageID gets pageID filed in structure.
// It tries to get pageID from VersionInfo data.
// If getting ends with fail, it returns default pageID
// 040904E4 // US English + CP_USASCII.
func (f Info) GetPageID() Locale {
	data, err := f.verQueryValue(`\VarFileInfo\Translation`, false)
	if err != nil || len(data) == 0 {
		return EnglishUSAscii
	}

	// each pageID consists of a 16-bit language ID and a 16-bit code page
	type langAndCodePage struct {
		Language uint16
		CodePage uint16
	}
	if len(data)%int(unsafe.Sizeof(langAndCodePage{})) != 0 {
		return EnglishUSAscii
	}
	n := len(data) / int(unsafe.Sizeof(langAndCodePage{}))
	if n == 0 {
		return EnglishUSAscii
	}

	ids := (*[1 << 28]langAndCodePage)(unsafe.Pointer(&data[0]))[:n:n]
	return Locale(fmt.Sprintf("%04x%04x", ids[0].Language, ids[0].CodePage))
}

// GetFixedInfo returns FixedFileInfo structure, which constructs from windows
// VS_FIXEDFILEINFO structure.
// source:
// https://helloacm.com/c-function-to-get-file-version-using-win32-api-ansi-and-unicode-version/
func (f Info) GetFixedInfo() FixedFileInfo {
	data, err := f.verQueryValue(`\`, false)
	if err != nil {
		return FixedFileInfo{}
	}
	// source:
	// https://docs.microsoft.com/en-us/windows/win32/api/verrsrc/ns-verrsrc-vs_fixedfileinfo
	type rawFixedFileInfo struct {
		Signature        uint32
		StrucVersion     uint32
		FileVersionMS    uint32
		FileVersionLS    uint32
		ProductVersionMS uint32
		ProductVersionLS uint32
		FileFlagsMask    uint32
		FileFlags        uint32
		FileOS           uint32
		FileType         uint32
		FileSubtype      uint32
		FileDateMS       uint32
		FileDateLS       uint32
	}
	vsFixedInfo := *((*rawFixedFileInfo)(unsafe.Pointer(&data[0])))
	return FixedFileInfo{
		FileMajor:      uint16(vsFixedInfo.FileVersionMS >> 16),
		FileMinor:      uint16(vsFixedInfo.FileVersionMS & 0xffff),
		FileBuild:      uint16(vsFixedInfo.FileVersionLS >> 16),
		FilePrivate:    uint16(vsFixedInfo.FileVersionLS & 0xffff),
		ProductMajor:   uint16(vsFixedInfo.ProductVersionMS >> 16),
		ProductMinor:   uint16(vsFixedInfo.ProductVersionMS & 0xffff),
		ProductBuild:   uint16(vsFixedInfo.ProductVersionLS >> 16),
		ProductPrivate: uint16(vsFixedInfo.ProductVersionLS & 0xffff),
		FileFlagsMask:  vsFixedInfo.FileFlagsMask,
		FileFlags:      vsFixedInfo.FileFlags,
		FileOs:         vsFixedInfo.FileOS,
		FileType:       vsFixedInfo.FileType,
		FileSubType:    vsFixedInfo.FileSubtype,
		FileDateMS:     vsFixedInfo.FileDateMS,
		FileDateLS:     vsFixedInfo.FileDateLS,
	}
}

// GetProperty returns string-property.
func (f Info) GetProperty(propertyName string) (property string) {
	property, err := f.verQueryValueString(f.pageID, propertyName)
	if err == nil {
		return
	}
	// Some dlls might not contain correct codepage information. In this case we will fail during lookup.
	// Explorer will take a few shots in dark by trying `defaultPageIDs`.
	// Explorer also randomly guess 041D04B0=Swedish+CP_UNICODE and 040704B0=German+CP_UNICODE) sometimes.
	// We will try to simulate similiar behavior here.
	for _, id := range []Locale{EnglishUSAscii, EnglishUnicode, EnglishUnknown} {
		if id == f.pageID {
			continue
		}
		property, err = f.verQueryValueString(id, propertyName)
		if err == nil {
			return
		}
	}
	return
}

// verQueryValueString returns property with type UTF16.
func (f Info) verQueryValueString(pageID Locale, property string) (string, error) {
	data, err := f.verQueryValue(`\StringFileInfo\`+string(pageID)+`\`+property, true)
	if err != nil || len(data) == 0 {
		return "", err
	}
	n := len(data) / 2
	u16 := (*[1 << 28]uint16)(unsafe.Pointer(&data[0]))[:n:n]
	return syscall.UTF16ToString(u16), err
}

// verQueryValue returns property data.
func (f Info) verQueryValue(property string, isUTF16String bool) ([]byte, error) {
	var offset uintptr
	var length uint
	blockStart := uintptr(unsafe.Pointer(&f.data[0]))
	propertyUTF16Ptr, err := syscall.UTF16PtrFromString(property)
	if err != nil {
		return nil, err
	}
	ret, _, err := verQueryValueProc.Call(
		blockStart,
		uintptr(unsafe.Pointer(propertyUTF16Ptr)),
		uintptr(unsafe.Pointer(&offset)),
		uintptr(unsafe.Pointer(&length)),
	)
	if ret == 0 {
		return nil, err
	}
	// We need calculate indexes of needed data in `f.data` memory.
	// `end` depends on length, which can be represent in characters or in bytes
	// source: `puLen` parameter in
	// https://docs.microsoft.com/en-us/windows/win32/api/winver/nf-winver-verqueryvaluew
	start := int(offset) - int(blockStart)
	var end int
	if isUTF16String {
		end = start + int(2*length) // length represents in characters count in string
	} else {
		end = start + int(length)
	}
	if start < 0 || end > len(f.data) {
		return nil, xerrors.New("Index out of array")
	}
	return f.data[start:end], nil
}
