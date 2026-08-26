package card

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/zeefan1555/fanloop/internal/idl/cardidl"
)

const botmuxSendTimeout = 30 * time.Second

func AttemptPanoramaDelivery(ctx context.Context, root, sourceEventID string) string {
	binding, configured, err := LoadOrCaptureBinding(root)
	if err != nil {
		return deliveryWarning(sourceEventID, "load Card binding", err)
	}
	if !configured {
		return ""
	}
	response, failure := DefaultRuntime().Render(ctx, root, &cardidl.CardRenderRequest{View: cardidl.CardView_panorama, Format: cardidl.CardFormat_lark_json}, false)
	if failure != nil {
		return deliveryWarning(sourceEventID, "render Panorama Card", failure)
	}
	if response.GetSnapshotPath() == "" {
		return deliveryWarning(sourceEventID, "render Panorama Card", fmt.Errorf("renderer returned no snapshot_path"))
	}
	if err := sendBotmux(ctx, binding, filepath.Join(root, filepath.FromSlash(response.GetSnapshotPath()))); err != nil {
		return deliveryWarning(sourceEventID, "send Panorama Card", err)
	}
	return ""
}

func sendBotmux(ctx context.Context, binding BotmuxBinding, snapshotPath string) error {
	ctx, cancel := context.WithTimeout(ctx, botmuxSendTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, "botmux",
		"send", "--card-file", snapshotPath,
		"--no-mention", "--session-id", binding.SessionID,
	)
	command.Stdout, command.Stderr = io.Discard, io.Discard
	if err := command.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("Botmux send timed out; outcome unknown because the message may already have been sent; not retrying")
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return fmt.Errorf("Botmux send was canceled; outcome unknown because the message may already have been sent; not retrying")
		}
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return fmt.Errorf("Botmux failed with exit code %d", exitError.ExitCode())
		}
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("Botmux executable was not found")
		}
		return fmt.Errorf("Botmux process could not start")
	}
	return nil
}

func deliveryWarning(sourceEventID, action string, err error) string {
	return fmt.Sprintf("Panorama Card delivery for Flow Event %s could not %s: %v", sourceEventID, action, err)
}
