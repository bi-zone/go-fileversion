// Package fileversion provides wrapper for querying properties from windows
// version-information resource.
//
// fileversion API is aimed to the easiest way of getting file properties so
// it ignore most of errors querying properties. We suppose most of the time
// it will be used as "create with New and just access properties". If you
// need some guaranties - access the properties manually using GetProperty and
// GetPropertyWithLocale.
//
// For more info about version-information resource look at
// https://docs.microsoft.com/en-us/windows/win32/menurc/versioninfo-resource
package fileversion

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/xerrors"
)

// FileVersion is a multi-component version.
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

// FixedFileInfo contains a "fixed" part of a file information (without any strings).
//
// Ref VS_FIXEDFILEINFO:
// https://docs.microsoft.com/en-us/windows/win32/api/verrsrc/ns-verrsrc-vs_fixedfileinfo
type FixedFileInfo struct {
	FileVersion    FileVersion
	ProductVersion FileVersion
	FileFlagsMask  uint32
	FileFlags      uint32
	FileOs         uint32
	FileType       uint32
	FileSubType    uint32
	FileDateMS     uint32
	FileDateLS     uint32
}

// LangID is a Windows language identifier. Could be one of the codes listed in
// `langID` section of
// https://docs.microsoft.com/en-us/windows/win32/menurc/versioninfo-resource
type LangID uint16

// CharsetID is character-set identifier. Could be one of the codes listed in
// `charsetID` section of
// https://docs.microsoft.com/en-us/windows/win32/menurc/versioninfo-resource
type CharsetID uint16

// Locale defines a pair of a language ID and a charsetID. It can be either any
// combination of predefined LangID and CharsetID or crafted manually suing
// values from https://docs.microsoft.com/en-us/windows/win32/menurc/versioninfo-resource
type Locale struct {
	LangID    LangID
	CharsetID CharsetID
}

// The package defines a list of most commonly used LangID and CharsetID
// constant. More combinations you can find in windows docs or at
// https://godoc.org/github.com/josephspurrier/goversioninfo#pkg-constants
const (
	LangEnglish = LangID(0x049)

	CSAscii   = CharsetID(0x04e4)
	CSUnicode = CharsetID(0x04B0)
	CSUnknown = CharsetID(0x0000)
)

// DefaultLocales is a list of default Locale values. It's used as a fallback
// in a calls with automatic locales detection.
//
//nolint:gochecknoglobals
var DefaultLocales = []Locale{
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

// Info contains a transparent windows object, which is being used for getting
// file version resource properties.
//
// Locales is a list of locales defined for the object. For the Info created
// using New it's queried from `\VarFileInfo\Translation`, for ones created
// using NewWithLocale it's just the given locale.
//
// A translation for the any property value is automatically chosen from Locales
// and then from fileversion.DefaultLocales prior to to the list order. Use
// GetPropertyWithLocale for deterministic selection of the property translation.
type Info struct {
	Locales []Locale
	data    []byte
}

// New creates an Info instance.
//
// It queries a list of translations from the version-information resource and
// uses them as preferred translations for string properties.
func New(path string) (Info, error) {
	info, err := newWithoutLocale(path)
	if err != nil {
		return Info{}, xerrors.Errorf("failed to get VersionInfo: %w", err)
	}

	if locales, err := info.getLocales(); err == nil {
		info.Locales = locales
	} else {
		info.Locales = DefaultLocales
	}

	return info, nil
}

// NewWithLocale creates an Info instance with a given locale. All the string
// properties translations will be firstly queried with the given locale.
//
// See GetPropertyWithLocale for exact properties querying.
func NewWithLocale(path string, locale Locale) (Info, error) {
	info, err := newWithoutLocale(path)
	if err != nil {
		return Info{}, xerrors.Errorf("failed to get VersionInfo: %w", err)
	}
	info.Locales = []Locale{locale}
	return info, nil
}

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

// LegalTrademarks returns LegalTrademarks property.
func (f Info) LegalTrademarks() string {
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

// FixedInfo returns a fixed (non-string) part of the file version-information
// resource. Contains file and product versions.
//
// Ref: https://helloacm.com/c-function-to-get-file-version-using-win32-api-ansi-and-unicode-version/
func (f Info) FixedInfo() FixedFileInfo {
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

// GetProperty queries a string-property from version-information resource.
//
// Single property in a version-information resource can have multiple
// translations. GetProperty does its best trying to find an existing
// translation: it returns a first existing translation for any of .Locales
// and if failed tries to query it for locales from fileversion.DefaultLocales.
func (f Info) GetProperty(propertyName string) (string, error) {
	for _, id := range f.Locales {
		property, err := f.GetPropertyWithLocale(propertyName, id)
		if err == nil {
			return property, nil
		}
	}
	// Some dlls might not contain correct codepage information. In this case we will fail during lookup.
	// Explorer will take a few shots in dark by trying `defaultPageIDs`.
	// Explorer also randomly guess 041D04B0=Swedish+CP_UNICODE and 040704B0=German+CP_UNICODE) sometimes.
	// We will try to simulate similar behavior here.
	for _, id := range DefaultLocales {
		property, err := f.GetPropertyWithLocale(propertyName, id)
		if err == nil {
			return property, nil
		}
	}
	return "", xerrors.Errorf("failed to get property %q", propertyName)
}

// GetPropertyWithLocale returns string-property with user-defined locale. It's
// the only way to get the property with the selected translation, all other
// methods do heuristics in translation choosing.
//
// See Locale, LangID and CharsetID docs for more info about locales.
func (f Info) GetPropertyWithLocale(propertyName string, locale Locale) (string, error) {
	property, err := f.verQueryValueString(locale, propertyName)
	if err != nil {
		return "", xerrors.Errorf("failed to get property %q with locale %+v", propertyName, locale)
	}
	return property, nil
}

//nolint:gochecknoglobals
var uint16Size = int(unsafe.Sizeof(uint16(0)))

//nolint:gochecknoglobals
var (
	version                    = syscall.NewLazyDLL("version.dll")
	getFileVersionInfoSizeProc = version.NewProc("GetFileVersionInfoSizeW")
	getFileVersionInfoProc     = version.NewProc("GetFileVersionInfoW")
	verQueryValueProc          = version.NewProc("VerQueryValueW")
)

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
		return nil, xerrors.New("index out of range")
	}
	return f.data[start:end], nil
}

func newWithoutLocale(path string) (Info, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return Info{}, xerrors.Errorf("failed to convert image path to utf16: %w", err)
	}
	size, _, err := getFileVersionInfoSizeProc.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0,
	)
	if size == 0 {
		return Info{}, xerrors.Errorf("failed to get memory size for VersionInfo slice: %w", err)
	}
	info := make([]byte, size)
	ret, _, err := getFileVersionInfoProc.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0,
		uintptr(len(info)),
		uintptr(unsafe.Pointer(&info[0])),
	)
	if ret == 0 {
		return Info{}, xerrors.Errorf("failed to get VersionInfo from windows: %w", err)
	}

	vi := Info{data: info}
	return vi, nil
}

// getLocales tries to get `Translation` property from VersionInfo data.
func (f Info) getLocales() ([]Locale, error) {
	data, err := f.verQueryValue(`\VarFileInfo\Translation`, false)
	if err != nil || len(data) == 0 {
		return nil, xerrors.Errorf("failed to get Translation property from a windows object: %w", err)
	}

	if len(data)%int(unsafe.Sizeof(Locale{})) != 0 {
		return nil, xerrors.New("get wrong locales len in a windows object")
	}
	n := len(data) / int(unsafe.Sizeof(Locale{}))
	if n == 0 {
		return nil, xerrors.New("get empty locales array in a windows object")
	}
	locales := (*[1 << 28]Locale)(unsafe.Pointer(&data[0]))[:n:n]
	return locales, nil
}
