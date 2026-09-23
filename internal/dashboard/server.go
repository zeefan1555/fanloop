package dashboard

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/zeefan1555/fanloop/internal/workflow"
)

//go:embed static/*
var assets embed.FS

type fileItem struct {
	Path string `json:"path"`
}

type fileResponse struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	SHA256  string `json:"sha256"`
}

type apiError struct {
	Error struct {
		Code          string `json:"code"`
		Message       string `json:"message"`
		CurrentSHA256 string `json:"current_sha256,omitempty"`
	} `json:"error"`
}

type handler struct {
	root      string
	ui        http.Handler
	saveMu    sync.Mutex
	allowedMu sync.RWMutex
	allowed   []string
}

func NewHandler(root string) (http.Handler, error) {
	if !filepath.IsAbs(root) {
		return nil, fmt.Errorf("Workflow root must be absolute")
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("Workflow root must be a real directory")
	}
	definitions, err := workflow.ListFrom(root)
	if err != nil {
		return nil, err
	}
	static, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, err
	}
	return &handler{root: filepath.Clean(root), ui: http.FileServer(http.FS(static)), allowed: allowedPathsFrom(definitions)}, nil
}

func (h *handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if strings.HasPrefix(request.URL.Path, "/api/") {
		unlock, err := workflow.LockConfiguration(filepath.Dir(filepath.Dir(h.root)), false)
		if err != nil {
			h.writeError(writer, http.StatusInternalServerError, "configuration_locked", err.Error(), "")
			return
		}
		defer unlock()
	}

	switch request.URL.Path {
	case "/api/files":
		if request.Method != http.MethodGet {
			h.writeError(writer, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.", "")
			return
		}
		h.list(writer)
	case "/api/file":
		switch request.Method {
		case http.MethodGet:
			h.read(writer, request.URL.Query().Get("path"))
		default:
			h.writeError(writer, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.", "")
		}
	case "/api/topology":
		if request.Method != http.MethodGet {
			h.writeError(writer, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.", "")
			return
		}
		h.topology(writer, request.URL.Query().Get("workflow"))
	default:
		h.ui.ServeHTTP(writer, request)
	}
}

func (h *handler) topology(writer http.ResponseWriter, workflowID string) {
	h.saveMu.Lock()
	defer h.saveMu.Unlock()

	definitions, err := workflow.ListFrom(h.root)
	if err != nil {
		h.writeError(writer, http.StatusUnprocessableEntity, "invalid_workflow", err.Error(), "")
		return
	}
	found := false
	for _, definition := range definitions {
		if definition.Ref.ID == workflowID {
			found = true
			break
		}
	}
	if !found {
		h.writeError(writer, http.StatusBadRequest, "invalid_workflow", errUnknownWorkflow.Error(), "")
		return
	}
	allowed := allowedPathsFrom(definitions)
	h.rememberAllowed(allowed)
	value, err := buildTopology(workflowID, func(relative string) ([]byte, error) {
		path, err := h.resolveAllowed(relative, allowed)
		if err != nil {
			return nil, err
		}
		return os.ReadFile(path)
	})
	if err != nil {
		h.writeError(writer, http.StatusUnprocessableEntity, "invalid_workflow", err.Error(), "")
		return
	}
	h.writeJSON(writer, http.StatusOK, value)
}

func (h *handler) list(writer http.ResponseWriter) {
	paths, err := h.allowedPaths()
	if err != nil {
		h.writeError(writer, http.StatusUnprocessableEntity, "invalid_workflows", err.Error(), "")
		return
	}
	items := make([]fileItem, len(paths))
	for index, path := range paths {
		items[index] = fileItem{Path: path}
	}
	h.writeJSON(writer, http.StatusOK, map[string]any{"files": items})
}

func (h *handler) read(writer http.ResponseWriter, relative string) {
	path, err := h.resolve(relative)
	if err != nil {
		h.writeError(writer, http.StatusBadRequest, "invalid_path", err.Error(), "")
		return
	}
	content, err := os.ReadFile(path)
	if err != nil {
		h.writeError(writer, http.StatusBadRequest, "invalid_path", err.Error(), "")
		return
	}
	h.writeJSON(writer, http.StatusOK, fileResponse{Path: relative, Content: string(content), SHA256: digest(content)})
}

func (h *handler) allowedPaths() ([]string, error) {
	definitions, err := workflow.ListFrom(h.root)
	if err != nil {
		h.allowedMu.RLock()
		defer h.allowedMu.RUnlock()
		if len(h.allowed) == 0 {
			return nil, err
		}
		return append([]string(nil), h.allowed...), nil
	}
	allowed := allowedPathsFrom(definitions)
	h.rememberAllowed(allowed)
	return allowed, nil
}

func (h *handler) rememberAllowed(allowed []string) {
	h.allowedMu.Lock()
	defer h.allowedMu.Unlock()
	h.allowed = append(h.allowed[:0], allowed...)
}

func allowedPathsFrom(definitions []workflow.Loaded) []string {
	paths := make([]string, 0)
	for _, definition := range definitions {
		for _, name := range workflow.BundleFileNames() {
			paths = append(paths, definition.Ref.ID+"/"+name)
		}
		for name := range definition.SkillPaths {
			paths = append(paths, definition.Ref.ID+"/skills/"+name+"/SKILL.md")
		}
	}
	sort.Strings(paths)
	return paths
}

func (h *handler) resolve(relative string) (string, error) {
	allowed, err := h.allowedPaths()
	if err != nil {
		return "", err
	}
	return h.resolveAllowed(relative, allowed)
}

func (h *handler) resolveAllowed(relative string, allowed []string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) || strings.Contains(relative, `\`) || filepath.ToSlash(filepath.Clean(relative)) != relative || strings.HasPrefix(relative, "../") {
		return "", fmt.Errorf("path is not allowed")
	}
	index := sort.SearchStrings(allowed, relative)
	if index == len(allowed) || allowed[index] != relative {
		return "", fmt.Errorf("path is not allowed")
	}
	path := h.root
	for _, part := range strings.Split(relative, "/") {
		path = filepath.Join(path, part)
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("path is not a regular allowlisted file")
		}
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("path is not a regular allowlisted file")
	}
	return path, nil
}

func digest(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func (h *handler) writeError(writer http.ResponseWriter, status int, code, message, current string) {
	value := apiError{}
	value.Error.Code = code
	value.Error.Message = message
	value.Error.CurrentSHA256 = current
	h.writeJSON(writer, status, value)
}

func (h *handler) writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
