//go:build windows

package windows

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"github.com/benworks/menuworks/discover"
)

// StartMenuSource discovers applications from Windows Start Menu shortcuts.
type StartMenuSource struct{}

func (s *StartMenuSource) Name() string     { return "startmenu" }
func (s *StartMenuSource) Category() string { return "Applications" }

func (s *StartMenuSource) Available() bool {
	for _, dir := range startMenuDirs() {
		if _, err := os.Stat(dir); err == nil {
			return true
		}
	}
	return false
}

func (s *StartMenuSource) Discover() ([]discover.DiscoveredApp, error) {
	var apps []discover.DiscoveredApp
	seen := make(map[string]bool)

	for _, dir := range startMenuDirs() {
		if _, err := os.Stat(dir); err != nil {
			continue
		}

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // skip inaccessible entries
			}
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(path), ".lnk") {
				return nil
			}

			name := strings.TrimSuffix(info.Name(), ".lnk")
			if isFilteredShortcut(name) {
				return nil
			}

			target := resolveShortcut(path)
			if target == "" {
				return nil
			}

			// Normalize for dedup
			key := strings.ToLower(target)
			if seen[key] {
				return nil
			}
			seen[key] = true

			apps = append(apps, discover.DiscoveredApp{
				Name:     name,
				Exec:     target,
				Source:   "startmenu",
				Category: "Applications",
			})
			return nil
		})
		if err != nil {
			// Continue scanning other directories
			continue
		}
	}

	return apps, nil
}

// startMenuDirs returns the Start Menu Programs directories to scan.
func startMenuDirs() []string {
	var dirs []string

	// Common (all users) start menu
	if pd := os.Getenv("ProgramData"); pd != "" {
		dirs = append(dirs, filepath.Join(pd, "Microsoft", "Windows", "Start Menu", "Programs"))
	}

	// Per-user start menu
	if appdata := os.Getenv("APPDATA"); appdata != "" {
		dirs = append(dirs, filepath.Join(appdata, "Microsoft", "Windows", "Start Menu", "Programs"))
	}

	return dirs
}

// isFilteredShortcut returns true if the shortcut name suggests it should be excluded.
func isFilteredShortcut(name string) bool {
	lower := strings.ToLower(name)
	filterWords := []string{
		"uninstall", "remove", "update", "updater",
		"readme", "help", "documentation", "manual",
		"license", "release notes", "changelog",
		"repair", "troubleshoot", "diagnostic",
		"website", "web site", "home page",
	}
	for _, w := range filterWords {
		if strings.Contains(lower, w) {
			return true
		}
	}
	return false
}

// resolveShortcut reads a .lnk file and returns the target path.
// Uses the Windows IShellLink COM interface via raw syscall.
func resolveShortcut(lnkPath string) string {
	// Initialize COM
	ole32 := syscall.NewLazyDLL("ole32.dll")
	coInitialize := ole32.NewProc("CoInitializeEx")
	coUninitialize := ole32.NewProc("CoUninitialize")
	coCreateInstance := ole32.NewProc("CoCreateInstance")

	ret, _, _ := coInitialize.Call(0, 0) // COINIT_MULTITHREADED
	if ret != 0 && ret != 1 {            // S_OK or S_FALSE (already initialized)
		return ""
	}
	defer coUninitialize.Call()

	// CLSID_ShellLink = {00021401-0000-0000-C000-000000000046}
	clsidShellLink := &syscall.GUID{
		Data1: 0x00021401,
		Data2: 0x0000,
		Data3: 0x0000,
		Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
	}

	// IID_IShellLinkW = {000214F9-0000-0000-C000-000000000046}
	iidShellLink := &syscall.GUID{
		Data1: 0x000214F9,
		Data2: 0x0000,
		Data3: 0x0000,
		Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
	}

	// IID_IPersistFile = {0000010B-0000-0000-C000-000000000046}
	iidPersistFile := &syscall.GUID{
		Data1: 0x0000010B,
		Data2: 0x0000,
		Data3: 0x0000,
		Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
	}

	var psl uintptr
	ret, _, _ = coCreateInstance.Call(
		uintptr(unsafe.Pointer(clsidShellLink)),
		0,
		1, // CLSCTX_INPROC_SERVER
		uintptr(unsafe.Pointer(iidShellLink)),
		uintptr(unsafe.Pointer(&psl)),
	)
	if ret != 0 {
		return ""
	}
	defer callRelease(psl)

	// QueryInterface for IPersistFile
	var ppf uintptr
	ret = queryInterface(psl, iidPersistFile, &ppf)
	if ret != 0 {
		return ""
	}
	defer callRelease(ppf)

	// IPersistFile::Load
	lnkPathUTF16, err := syscall.UTF16PtrFromString(lnkPath)
	if err != nil {
		return ""
	}
	ret = persistFileLoad(ppf, lnkPathUTF16, 0) // STGM_READ
	if ret != 0 {
		return ""
	}

	// IShellLinkW::GetPath
	buf := make([]uint16, syscall.MAX_PATH)
	ret = shellLinkGetPath(psl, &buf[0], int32(len(buf)), 0, 0) // SLGP_RAWPATH
	if ret != 0 {
		return ""
	}

	return strings.TrimRight(syscall.UTF16ToString(buf), "\x00")
}

// COM vtable helpers

func queryInterface(obj uintptr, iid *syscall.GUID, out *uintptr) uintptr {
	// vtable[0] = QueryInterface
	vtable := *(*[3]uintptr)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(obj))))
	ret, _, _ := syscall.SyscallN(vtable[0], obj, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(out)))
	return ret
}

func callRelease(obj uintptr) {
	// vtable[2] = Release
	vtable := *(*[3]uintptr)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(obj))))
	syscall.SyscallN(vtable[2], obj)
}

func persistFileLoad(ppf uintptr, path *uint16, mode uint32) uintptr {
	// IPersistFile vtable: QI, AddRef, Release, GetClassID, IsDirty, Load(5), Save, SaveCompleted, GetCurFile
	vtable := *(*[9]uintptr)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(ppf))))
	ret, _, _ := syscall.SyscallN(vtable[5], ppf, uintptr(unsafe.Pointer(path)), uintptr(mode))
	return ret
}

func shellLinkGetPath(psl uintptr, buf *uint16, bufLen int32, findData uintptr, flags uint32) uintptr {
	// IShellLinkW vtable: QI, AddRef, Release, GetPath(3), ...
	vtable := *(*[20]uintptr)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(psl))))
	ret, _, _ := syscall.SyscallN(vtable[3], psl, uintptr(unsafe.Pointer(buf)), uintptr(bufLen), findData, uintptr(flags))
	return ret
}

// utf16ToString converts a UTF-16 byte slice to string. Used for internal processing.
func utf16ToString(s []uint16) string {
	for i, v := range s {
		if v == 0 {
			s = s[:i]
			break
		}
	}
	return string(utf16.Decode(s))
}
