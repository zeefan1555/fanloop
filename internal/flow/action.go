package flow

import (
	"os"
	"path/filepath"
	"strings"

	cardruntime "github.com/zeefan1555/fanloop/internal/card"
	"github.com/zeefan1555/fanloop/internal/cardsnapshot"
	"github.com/zeefan1555/fanloop/internal/idl/erroridl"
	"github.com/zeefan1555/fanloop/internal/state"
)

func validatePanoramaSnapshot(root, relative string, pending *state.PendingAction, current state.State) *erroridl.PublicError {
	const prefix = ".fanloop/card/"
	native := filepath.FromSlash(relative)
	name := strings.TrimPrefix(relative, prefix)
	if !strings.HasPrefix(relative, prefix) || name == "" || strings.Contains(name, "/") ||
		!strings.HasSuffix(name, ".json") || filepath.Clean(native) != native {
		return newFlowError(erroridl.ErrorCode_INVALID_ARGUMENT, "panorama_snapshot_path must reference one .fanloop/card/*.json snapshot", nil)
	}
	path := filepath.Join(root, native)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return newFlowError(erroridl.ErrorCode_INVALID_ARGUMENT, "panorama_snapshot_path must reference an existing regular snapshot", nil)
	}
	if !cardsnapshot.MatchesSourceEvent(relative, pending.TriggerEventID) {
		return newFlowError(erroridl.ErrorCode_INVALID_ARGUMENT, "panorama_snapshot_path must be rendered for the pending Flow action", nil)
	}
	matches, err := cardruntime.IsCurrentPanoramaSnapshot(root, relative, current, pending.TriggerEventID)
	if err != nil || !matches {
		return newFlowError(erroridl.ErrorCode_INVALID_ARGUMENT, "panorama_snapshot_path must match the current canonical Panorama", nil)
	}
	return nil
}
