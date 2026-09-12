package tools

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Source: tools/BriefTool/attachments.ts (upload.ts is out of scope — it is
// BRIDGE_MODE-gated and depends on unrelated auth/upload plumbing; FileUUID
// is therefore always left unset here).

// ResolvedAttachment is a stat'd attachment ready for display.
// Source: attachments.ts:22-27 (ResolvedAttachment)
type ResolvedAttachment struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	IsImage  bool   `json:"isImage"`
	FileUUID string `json:"file_uuid,omitempty"`
}

// briefImageExtensions matches IMAGE_EXTENSION_REGEX (/\.(png|jpe?g|gif|webp)$/i)
// used by attachments.ts — narrower than pkg/tools/fileread.go's imageExtensions
// (which also covers bmp/tiff/tif/ico/svg for FileRead's own rendering), so it
// is deliberately not reused here.
// Source: utils/imagePaste.ts:270 — IMAGE_EXTENSION_REGEX
var briefImageExtensions = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
}

// expandAttachmentPath resolves ~ and makes a relative path absolute against cwd.
// Source: attachments.ts:29 (expandPath), utils/path.ts
func expandAttachmentPath(cwd, p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			p = filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	if !filepath.IsAbs(p) && cwd != "" {
		p = filepath.Join(cwd, p)
	}
	return p
}

// validateAttachmentPaths checks that every path exists, is a regular file,
// and is accessible. Source: attachments.ts:29-56 (validateAttachmentPaths)
func validateAttachmentPaths(cwd string, raw []string) error {
	for _, rawPath := range raw {
		fullPath := expandAttachmentPath(cwd, rawPath)
		stat, err := os.Stat(fullPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("Attachment %q does not exist. Current working directory: %s.", rawPath, cwd)
			}
			if errors.Is(err, fs.ErrPermission) {
				return fmt.Errorf("Attachment %q is not accessible (permission denied).", rawPath)
			}
			return err
		}
		if !stat.Mode().IsRegular() {
			return fmt.Errorf("Attachment %q is not a regular file.", rawPath)
		}
	}
	return nil
}

// resolveAttachments stats each path (serially, deterministic order) and
// returns size/isImage metadata. Source: attachments.ts:64-88 (resolveAttachments,
// minus the BRIDGE_MODE upload branch at attachments.ts:90-108)
func resolveAttachments(cwd string, raw []string) ([]ResolvedAttachment, error) {
	stated := make([]ResolvedAttachment, 0, len(raw))
	for _, rawPath := range raw {
		fullPath := expandAttachmentPath(cwd, rawPath)
		// TOCTOU: validateAttachmentPaths ran before us, but the file could
		// have moved since — let the error propagate so the model sees it.
		stat, err := os.Stat(fullPath)
		if err != nil {
			return nil, err
		}
		ext := strings.ToLower(filepath.Ext(fullPath))
		stated = append(stated, ResolvedAttachment{
			Path:    fullPath,
			Size:    stat.Size(),
			IsImage: briefImageExtensions[ext],
		})
	}
	return stated, nil
}
