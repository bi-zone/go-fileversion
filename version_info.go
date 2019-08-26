package fileversion

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/xerrors"
)

// FileVersion consists info about version.
type FileVersion struct {
	Major uint16
	Minor uint16
	Patch uint16
	Build uint16
}

// String returns a string representation of the version.
func (f FileVersion) String() string {
	return fmt.Sprintf("%d.%d.%d.%d", f.Major, f.Minor, f.Patch, f.Build)
}

// FixedFileInfo contains info from VS_FIXEDFILEINFO windows structure.
type FixedFileInfo struct {
	FileVersion
	ProductVersion FileVersion
	FileFlagsMask  uint32
	FileFlags      uint32
	FileOs         uint32
	FileType       uint32
	FileSubType    uint32
	FileDateMS     uint32
	FileDateLS     uint32
}

// LangID is a language id, should set considering
// https://docs.microsoft.com/en-us/windows/win32/menurc/versioninfo-resource
type LangID uint16

// CharsetId is character-set identifier, should set considering
// https://docs.microsoft.com/en-us/windows/win32/menurc/versioninfo-resource
type CharsetID uint16

// Locale defines a pair of a language ID and a charsetID.
// It can be either one of default locales or any of locales
// get from the file information using `.Locales()` method
type Locale struct {
	LangID    LangID
	CharsetID CharsetID
}

// LangID and CharsetID constants.
// More combinations you can find in windows docs or
// https://godoc.org/github.com/josephspurrier/goversioninfo#pkg-constants
const (
	LangEnglish = LangID(0x049)

	CSAscii   = CharsetID(0x04e4)
	CSUnicode = CharsetID(0x04B0)
	CSUnknown = CharsetID(0x0000)
)

// Info contains windows object, which is being used
// for getting file version properties.
type Info struct {
	data    []byte
	Locales []Locale
}

var (
	version                    = syscall.NewLazyDLL("version.dll")
	getFileVersionInfoSizeProc = version.NewProc("GetFileVersionInfoSizeW")
	getFileVersionInfoProc     = version.NewProc("GetFileVersionInfoW")
	verQueryValueProc          = version.NewProc("VerQueryValueW")
)

// CompanyName returns CompanyName property.
func (f Info) CompanyName() string {
	p, _ := f.GetProperty("CompanyName")
	return p
}

// FileDescription returns FileDescription property.
func (f Info) FileDescription() string {
	p, _ := f.GetProperty("FileDescription")
	return p
}

// FileVersion returns FileVersion property.
func (f Info) FileVersion() string {
	p, _ := f.GetProperty("FileVersion")
	return p
}

// InternalName returns InternalName property.
func (f Info) InternalName() string {
	p, _ := f.GetProperty("InternalName")
	return p
}

// LegalCopyright returns LegalCopyright property.
func (f Info) LegalCopyright() string {
	p, _ := f.GetProperty("LegalCopyright")
	return p
}

// OriginalFilename returns OriginalFilename property.
func (f Info) OriginalFilename() string {
	p, _ := f.GetProperty("OriginalFilename")
	return p
}

// ProductName returns ProductName property.
func (f Info) ProductName() string {
	p, _ := f.GetProperty("ProductName")
	return p
}

// ProductVersion returns ProductVersion property.
func (f Info) ProductVersion() string {
	p, _ := f.GetProperty("ProductVersion")
	return p
}

// Comments returns Comments property.
func (f Info) Comments() string {
	p, _ := f.GetProperty("Comments")
	return p
}

// FileVeLegalTrademarksrsion returns FileVeLegalTrademarksrsion property.
func (f Info) FileVeLegalTrademarksrsion() string {
	p, _ := f.GetProperty("LegalTrademarks")
	return p
}

// PrivateBuild returns PrivateBuild property.
func (f Info) PrivateBuild() string {
	p, _ := f.GetProperty("PrivateBuild")
	return p
}

// SpecialBuild returns SpecialBuild property.
func (f Info) SpecialBuild() string {
	p, _ := f.GetProperty("SpecialBuild")
	return p
}

// New creates Info instance with default locale.
func New(path string) (Info, error) {
	info, err := newWithoutLocale(path)
	if err != nil {
		return Info{}, xerrors.Errorf("failed to get VersionInfo; %s", err)
	}

	locales, err := info.getLocales()
	if err != nil {
		return Info{}, xerrors.Errorf("failed to get locales; %s", err)
	}
	info.Locales = locales

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
	return vi, nil
}

// getLocales tries to get `Translation` property from VersionInfo data.
func (f Info) getLocales() ([]Locale, error) {
	data, err := f.verQueryValue(`\VarFileInfo\Translation`, false)
	if err != nil || len(data) == 0 {
		return nil, xerrors.Errorf("failed to get Translation property from windows object; %s", err)
	}

	if len(data)%int(unsafe.Sizeof(Locale{})) != 0 {
		return nil, xerrors.New("get wrong locales len from windows object")
	}
	n := len(data) / int(unsafe.Sizeof(Locale{}))
	if n == 0 {
		return nil, xerrors.New("get empty locales array from windows object")
	}
	locales := (*[1 << 28]Locale)(unsafe.Pointer(&data[0]))[:n:n]
	return locales, nil
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
		FileVersion: FileVersion{
			Major: uint16(vsFixedInfo.FileVersionMS >> 16),
			Minor: uint16(vsFixedInfo.FileVersionMS & 0xffff),
			Patch: uint16(vsFixedInfo.FileVersionLS & 0xffff),
			Build: uint16(vsFixedInfo.FileVersionLS >> 16),
		},
		ProductVersion: FileVersion{
			Major: uint16(vsFixedInfo.ProductVersionMS >> 16),
			Minor: uint16(vsFixedInfo.ProductVersionMS & 0xffff),
			Patch: uint16(vsFixedInfo.ProductVersionLS & 0xffff),
			Build: uint16(vsFixedInfo.ProductVersionLS >> 16),
		},
		FileFlagsMask: vsFixedInfo.FileFlagsMask,
		FileFlags:     vsFixedInfo.FileFlags,
		FileOs:        vsFixedInfo.FileOS,
		FileType:      vsFixedInfo.FileType,
		FileSubType:   vsFixedInfo.FileSubtype,
		FileDateMS:    vsFixedInfo.FileDateMS,
		FileDateLS:    vsFixedInfo.FileDateLS,
	}
}

var defaultLocales = []Locale{
	{
		LangID:    LangEnglish,
		CharsetID: CSAscii,
	},
	{
		LangID:    LangEnglish,
		CharsetID: CSUnicode,
	},
	{
		LangID:    LangEnglish,
		CharsetID: CSUnknown,
	},
}

// GetProperty returns string-property.
func (f Info) GetProperty(propertyName string) (string, error) {
	if len(f.Locales) != 0 {
		property, err := f.GetPropertyWithLocale(propertyName, f.Locales[0])
		if err == nil {
			return property, nil
		}
	}
	// Some dlls might not contain correct codepage information. In this case we will fail during lookup.
	// Explorer will take a few shots in dark by trying `defaultPageIDs`.
	// Explorer also randomly guess 041D04B0=Swedish+CP_UNICODE and 040704B0=German+CP_UNICODE) sometimes.
	// We will try to simulate similiar behavior here.
	for _, id := range defaultLocales {
		if len(f.Locales) != 0 && id == f.Locales[0] {
			continue
		}
		property, err := f.GetPropertyWithLocale(propertyName, id)
		if err == nil {
			return property, nil
		}
	}
	return "", xerrors.Errorf("failed to get property %q", propertyName)
}

// GetProperty returns string-property with user-defined locale.
func (f Info) GetPropertyWithLocale(propertyName string, locale Locale) (string, error) {
	property, err := f.verQueryValueString(locale, propertyName)
	if err != nil {
		return "", xerrors.Errorf("failed to get property %q with locale %#+v", propertyName, locale)
	}
	return property, nil
}

var uint16Size = int(unsafe.Sizeof(uint16(0)))

// verQueryValueString returns property with type UTF16.
func (f Info) verQueryValueString(locale Locale, property string) (string, error) {
	localeStr := fmt.Sprintf("%04x%04x", locale.LangID, locale.CharsetID)
	data, err := f.verQueryValue(`\StringFileInfo\`+localeStr+`\`+property, true)
	if err != nil || len(data) == 0 {
		return "", err
	}
	n := len(data) / uint16Size
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
		end = start + uint16Size*int(length) // length represents in characters count in string
	} else {
		end = start + int(length)
	}
	if start < 0 || end > len(f.data) {
		return nil, xerrors.New("Index out of array")
	}
	return f.data[start:end], nil
}
