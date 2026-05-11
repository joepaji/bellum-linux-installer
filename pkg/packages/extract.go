package packages

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xi2/xz"
)

const (
	defaultBufSize = 1 << 20 // 1MB buffer
)

// ExtractPackage extracts an archive to a temp directory under the packages/.tmp folder.
// archivePath: path to the archive file
// destName: base name for the temp directory (e.g., "dxvk", "winetricks")
// Returns the extracted directory path, or error
func ExtractPackage(archivePath string, destName string) (string, error) {
	// Get the packages directory (parent of archive)
	pkgDir := filepath.Dir(archivePath)
	tmpRoot := filepath.Join(pkgDir, ".tmp")
	if err := os.MkdirAll(tmpRoot, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Extract basename from destName since MkdirTemp pattern cannot contain path separators
	destNameBase := filepath.Base(destName)

	// Create temp subdirectory
	tmpDir, err := os.MkdirTemp(tmpRoot, destNameBase+".")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	return tmpDir, extractArchive(archivePath, tmpDir, 0)
}

// ExtractPackageTo extracts an archive directly to a specified destination directory.
// archivePath: path to the archive file
// destDir: the destination directory to extract to
// stripComponents: number of path components to strip from the archive entries (0 = no stripping)
// Returns the extracted directory path (same as destDir), or error
func ExtractPackageTo(archivePath, destDir string, stripComponents int) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	return extractArchive(archivePath, destDir, stripComponents)
}

// extractArchive extracts an archive to a specified destination directory.
// archivePath: path to the archive file
// destDir: the destination directory to extract to
// stripComponents: number of path components to strip from the archive entries
// Returns error if any
func extractArchive(archivePath, destDir string, stripComponents int) error {
	ext := strings.ToLower(archivePath)
	if strings.HasSuffix(ext, ".tar.xz") {
		return extractXZ(archivePath, destDir, stripComponents > 0)
	} else if strings.HasSuffix(ext, ".tar.gz") || strings.HasSuffix(ext, ".tgz") {
		return extractGZ(archivePath, destDir, stripComponents > 0)
	} else if strings.HasSuffix(ext, ".tar") {
		return extractTar(archivePath, destDir, stripComponents > 0)
	}

	return fmt.Errorf("unsupported archive format: %s", archivePath)
}

// extractGZ extracts a tar.gz archive
func extractGZ(archivePath, destDir string, stripComponents bool) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	// Collect all entries first - directories and file metadata
	type fileEntry struct {
		header *tar.Header
		target string
		data   []byte
	}

	var entries []fileEntry
	var dirs []string

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		// Apply strip components if requested
		entryName := header.Name
		if stripComponents && strings.Contains(entryName, string(filepath.Separator)) {
			parts := strings.Split(entryName, string(filepath.Separator))
			if len(parts) > 1 {
				// Strip the first component (archive root directory)
				entryName = strings.Join(parts[1:], string(filepath.Separator))
			}
		}

		target := filepath.Join(destDir, entryName)

		// Read the entire entry data into memory
		entryData, err := io.ReadAll(tr)
		if err != nil {
			return fmt.Errorf("failed to read entry data: %w", err)
		}

		if header.Typeflag == tar.TypeDir {
			dirs = append(dirs, target)
		} else {
			entries = append(entries, fileEntry{
				header: header,
				target: target,
				data:   entryData,
			})
		}
	}

	// Create all directories in sorted order (parents first)
	sortPaths(dirs)

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Separate symlink entries from regular file entries
	var symlinkEntries []fileEntry
	var regularEntries []fileEntry

	for _, e := range entries {
		if e.header.Typeflag == tar.TypeSymlink {
			symlinkEntries = append(symlinkEntries, e)
		} else {
			regularEntries = append(regularEntries, e)
		}
	}

	// Process regular files first (so symlink targets exist)
	for _, e := range regularEntries {
		if err := extractTarEntryFromData(e.data, e.header, e.target, destDir); err != nil {
			return err
		}
	}

	// Then process symlinks (targets now exist)
	for _, e := range symlinkEntries {
		if err := extractTarEntryFromData(e.data, e.header, e.target, destDir); err != nil {
			return err
		}
	}

	return nil
}

// extractXZ extracts a tar.xz archive
func extractXZ(archivePath, destDir string, stripComponents bool) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer f.Close()

	xzReader, err := xz.NewReader(f, 0)
	if err != nil {
		return fmt.Errorf("failed to create xz reader: %w", err)
	}

	tr := tar.NewReader(xzReader)
	buf := make([]byte, defaultBufSize)

	var deferred []*tar.Header

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		entryName := header.Name
		if stripComponents {
			parts := strings.Split(entryName, "/")
			if len(parts) <= 1 {
				continue
			}
			entryName = strings.Join(parts[1:], "/")
		}

		target, err := sanitizePath(destDir, entryName)
		if err != nil {
			return fmt.Errorf("invalid path in archive: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, header.FileInfo().Mode()); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			if _, err := io.CopyBuffer(outFile, tr, buf); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}

			if err := outFile.Close(); err != nil {
				return fmt.Errorf("failed to close file: %w", err)
			}

			if err := os.Chtimes(target, header.ModTime, header.ModTime); err != nil {
				return fmt.Errorf("failed to set file times: %w", err)
			}

		case tar.TypeSymlink, tar.TypeLink:
			// Defer links until files exist.
			h := *header
			h.Name = entryName
			deferred = append(deferred, &h)

		default:
			// Skip unsupported entry types.
			continue
		}
	}

	for _, header := range deferred {
		target, err := sanitizePath(destDir, header.Name)
		if err != nil {
			return fmt.Errorf("invalid deferred path in archive: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create symlink parent directory: %w", err)
			}
			if err := os.Symlink(header.Linkname, target); err != nil && !os.IsExist(err) {
				return fmt.Errorf("failed to create symlink: %w", err)
			}

		case tar.TypeLink:
			linkTarget := header.Linkname
			if stripComponents {
				parts := strings.Split(linkTarget, "/")
				if len(parts) > 1 {
					linkTarget = strings.Join(parts[1:], "/")
				}
			}

			linkTarget, err := sanitizePath(destDir, linkTarget)
			if err != nil {
				return fmt.Errorf("invalid hardlink target: %w", err)
			}

			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create hardlink parent directory: %w", err)
			}
			if err := os.Link(linkTarget, target); err != nil {
				return fmt.Errorf("failed to create hard link: %w", err)
			}
		}
	}

	return nil
}

// sortPaths sorts paths so parents come before children
func sortPaths(paths []string) {
	sort.Slice(paths, func(i, j int) bool {
		return paths[i] < paths[j]
	})
}

// sanitizePath validates and sanitizes a tar entry path to prevent path traversal
func sanitizePath(destDir, entryName string) (string, error) {
	// Clean the path to remove any ../ or . components
	cleanName := filepath.Clean(entryName)

	// Reject paths that try to escape the destination directory
	if strings.HasPrefix(cleanName, "..") || strings.Contains(cleanName, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("entry name contains path traversal: %s", entryName)
	}

	// Reject absolute paths
	if filepath.IsAbs(cleanName) {
		return "", fmt.Errorf("entry name is an absolute path: %s", entryName)
	}

	return filepath.Join(destDir, cleanName), nil
}

// extractTarEntry extracts a single tar entry with proper permissions
func extractTarEntry(tr io.Reader, header *tar.Header, target, destDir string) error {
	switch header.Typeflag {
	case tar.TypeDir:
		// Preserve original directory permissions
		if err := os.MkdirAll(target, header.FileInfo().Mode()); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

	case tar.TypeReg:
		// Create the file with original permissions
		outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, header.FileInfo().Mode())
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		if _, err := io.Copy(outFile, tr); err != nil {
			outFile.Close()
			return fmt.Errorf("failed to write file: %w", err)
		}
		outFile.Close()

	case tar.TypeSymlink:
		// Handle symbolic links
		if err := os.Symlink(header.Linkname, target); err != nil {
			return fmt.Errorf("failed to create symlink: %w", err)
		}

	case tar.TypeLink:
		// Handle hard links - resolve the link target to an absolute path in destDir
		// The linkname in tar may be absolute (within archive) or relative
		linkTarget := header.Linkname
		if filepath.IsAbs(linkTarget) {
			// Link target is absolute within the archive - strip the archive root and join with destDir
			// Clean the path to remove any leading slashes
			cleanLink := filepath.Clean(linkTarget)
			if strings.HasPrefix(cleanLink, string(filepath.Separator)) {
				cleanLink = cleanLink[1:]
			}
			linkTarget = filepath.Join(destDir, cleanLink)
		} else {
			// Link target is relative - resolve relative to the directory containing the link
			linkTarget = filepath.Join(destDir, filepath.Dir(target), linkTarget)
		}
		linkTarget = filepath.Clean(linkTarget)
		if err := os.Link(linkTarget, target); err != nil {
			return fmt.Errorf("failed to create hard link: %w", err)
		}

	default:
		// Skip unsupported entry types (e.g., device files, sockets)
		return nil
	}

	// Restore file permissions and modification time
	if err := os.Chtimes(target, header.ModTime, header.ModTime); err != nil {
		return fmt.Errorf("failed to set file times: %w", err)
	}

	return nil
}

// extractTarEntryFromData extracts a single tar entry from pre-read data
func extractTarEntryFromData(data []byte, header *tar.Header, target, destDir string) error {
	switch header.Typeflag {
	case tar.TypeDir:
		// Preserve original directory permissions
		if err := os.MkdirAll(target, header.FileInfo().Mode()); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

	case tar.TypeReg:
		// Create the file with original permissions
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}
		if err := os.WriteFile(target, data, header.FileInfo().Mode()); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

	case tar.TypeSymlink:
		// Handle symbolic links - use Lchown/LChmod equivalent by not following symlinks
		if err := os.Symlink(header.Linkname, target); err != nil {
			return fmt.Errorf("failed to create symlink: %w", err)
		}
		// For symlinks, only set times if the symlink itself should have times
		// Note: os.Chtimes on a symlink follows the symlink, so we skip it for symlinks
		// to avoid errors when the target doesn't exist yet

	case tar.TypeLink:
		// Handle hard links - resolve the link target to an absolute path in destDir
		// The linkname in tar may be absolute (within archive) or relative
		linkTarget := header.Linkname

		// If the link target starts with the archive root prefix, strip it
		// The archive root is the first path component of the current entry's name
		archiveRoot := filepath.Base(filepath.Dir(target))
		if strings.HasPrefix(linkTarget, archiveRoot+string(filepath.Separator)) {
			// Strip the archive root prefix
			linkTarget = linkTarget[len(archiveRoot)+1:]
		}

		if strings.HasPrefix(linkTarget, string(filepath.Separator)) {
			// Remove leading slash if present
			linkTarget = linkTarget[1:]
		}

		if filepath.IsAbs(linkTarget) {
			// Link target is absolute - join with destDir
			linkTarget = filepath.Join(destDir, linkTarget)
		} else {
			// Link target is relative - resolve relative to the directory containing the link
			linkTarget = filepath.Join(destDir, filepath.Dir(target), linkTarget)
		}
		linkTarget = filepath.Clean(linkTarget)
		if err := os.Link(linkTarget, target); err != nil {
			return fmt.Errorf("failed to create hard link: %w", err)
		}

	default:
		// Skip unsupported entry types (e.g., device files, sockets)
		return nil
	}

	// Restore file permissions and modification time
	// Skip for symlinks to avoid following them and failing on non-existent targets
	if header.Typeflag != tar.TypeSymlink {
		if err := os.Chtimes(target, header.ModTime, header.ModTime); err != nil {
			return fmt.Errorf("failed to set file times: %w", err)
		}
	}

	return nil
}

// extractTar extracts a plain tar archive
func extractTar(archivePath, destDir string, stripComponents bool) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer f.Close()

	tr := tar.NewReader(f)

	// Collect all entries first - directories and file metadata
	type fileEntry struct {
		header *tar.Header
		target string
		data   []byte
	}

	var entries []fileEntry
	var dirs []string

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		// Apply strip components if requested
		entryName := header.Name
		if stripComponents && strings.Contains(entryName, string(filepath.Separator)) {
			parts := strings.Split(entryName, string(filepath.Separator))
			if len(parts) > 1 {
				// Strip the first component (archive root directory)
				entryName = strings.Join(parts[1:], string(filepath.Separator))
			}
		}

		target := filepath.Join(destDir, entryName)

		// Read the entire entry data into memory
		entryData, err := io.ReadAll(tr)
		if err != nil {
			return fmt.Errorf("failed to read entry data: %w", err)
		}

		if header.Typeflag == tar.TypeDir {
			dirs = append(dirs, target)
		} else {
			entries = append(entries, fileEntry{
				header: header,
				target: target,
				data:   entryData,
			})
		}
	}

	// Create all directories in sorted order (parents first)
	sortPaths(dirs)

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Separate symlink entries from regular file entries
	var symlinkEntries []fileEntry
	var regularEntries []fileEntry

	for _, e := range entries {
		if e.header.Typeflag == tar.TypeSymlink {
			symlinkEntries = append(symlinkEntries, e)
		} else {
			regularEntries = append(regularEntries, e)
		}
	}

	// Process regular files first (so symlink targets exist)
	for _, e := range regularEntries {
		if err := extractTarEntryFromData(e.data, e.header, e.target, destDir); err != nil {
			return err
		}
	}

	// Then process symlinks (targets now exist)
	for _, e := range symlinkEntries {
		if err := extractTarEntryFromData(e.data, e.header, e.target, destDir); err != nil {
			return err
		}
	}

	return nil
}

// CleanupTempDir removes the packages/.tmp directory
func CleanupTempDir(workdir string) {
	tmpRoot := filepath.Join(workdir, "packages", ".tmp")
	if _, err := os.Stat(tmpRoot); err == nil {
		if err := os.RemoveAll(tmpRoot); err != nil {
			// Log warning but don't fail - cleanup is best-effort
		}
	}
}

// CleanupSpecificTempDir removes a specific temp directory
func CleanupSpecificTempDir(tempDir string) {
	if tempDir != "" {
		os.RemoveAll(tempDir)
	}
}

// GetPackagePath returns the path to a bundled package file
// relative to the binary's directory
func GetPackagePath(name string) string {
	exePath, err := os.Executable()
	if err != nil {
		return ""
	}
	exeDir := filepath.Dir(exePath)
	return filepath.Join(exeDir, "packages", name)
}
