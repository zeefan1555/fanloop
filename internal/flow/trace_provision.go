package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	cardruntime "github.com/zeefan1555/fanloop/internal/card"
	"github.com/zeefan1555/fanloop/internal/idl/traceidl"
	"github.com/zeefan1555/fanloop/internal/larkexec"
	"github.com/zeefan1555/fanloop/internal/trace"
	"github.com/zeefan1555/fanloop/internal/traceconfig"
)

const traceDocumentTimeout = 35 * time.Second

// provisionTraceDocument creates a dedicated Trace document and binds it during
// flow init. A captured Botmux binding identifies an interactive run; without
// one, offline and test runs remain hermetic.
func provisionTraceDocument(ctx context.Context, root, title string) string {
	_, configured, err := cardruntime.LoadOrCaptureBinding(root)
	if err != nil {
		return provisionWarning("load Card binding", err)
	}
	if !configured {
		return ""
	}
	_, current, _, failure := load(root)
	if failure != nil {
		return provisionWarning("load initialized Flow", failure)
	}
	documentURL, err := createTraceDocument(ctx, title)
	if err != nil {
		return provisionWarning("create Trace document", err)
	}
	request := &traceidl.TraceBindRequest{DocumentUrl: documentURL}
	registry, ok := traceconfig.Resolve(traceconfig.RegistryProduction, current.Release.Workflow.ID)
	if ok && registry.RequireCLILogDocument {
		cliLogDocumentURL, err := createCLILogDocument(ctx, title)
		if err != nil {
			return provisionWarning("create CLI log document", err)
		}
		request.CliLogDocumentUrl = &cliLogDocumentURL
	}
	if _, failure := trace.DefaultRuntime().Bind(ctx, root, request, false); failure != nil {
		return provisionWarning("bind Trace document", failure)
	}
	return ""
}

func createTraceDocument(ctx context.Context, title string) (string, error) {
	return createDocument(ctx, "Trace · "+title, "Fanloop 审计投影文档，将由 trace sync 自动同步。")
}

func createCLILogDocument(ctx context.Context, title string) (string, error) {
	return createDocument(ctx, "CLI 日志 · "+title, "Fanloop CLI 完整输入输出，将由 trace sync 自动同步。")
}

func createDocument(ctx context.Context, title, description string) (string, error) {
	content := "<title>" + escapeXML(title) + "</title><p>" + escapeXML(description) + "</p>"
	result, err := larkexec.Execute(ctx, []string{
		"docs", "+create", "--as", "user", "--content", content,
	}, nil, traceDocumentTimeout)
	if err != nil {
		return "", err
	}
	if result.ExitCode != 0 {
		return "", fmt.Errorf("lark-cli exited with %d: %s", result.ExitCode, firstNonEmpty(strings.TrimSpace(result.Stderr), strings.TrimSpace(result.Stdout)))
	}
	var response struct {
		OK   bool `json:"ok"`
		Data struct {
			Document struct {
				URL string `json:"url"`
			} `json:"document"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		return "", fmt.Errorf("decode lark-cli output: %w", err)
	}
	if !response.OK || response.Data.Document.URL == "" {
		return "", fmt.Errorf("lark-cli returned no document URL")
	}
	return response.Data.Document.URL, nil
}

func escapeXML(value string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(value)
}

func provisionWarning(action string, err error) string {
	return fmt.Sprintf("Trace document provisioning during flow init could not %s: %v", action, err)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
