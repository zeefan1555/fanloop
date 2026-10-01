package dashboard

import (
	cryptorand "crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/zeefan1555/fanloop/internal/workflow"
	"golang.org/x/sys/unix"
)

//go:embed static/*
var assets embed.FS

type fileItem struct {
	Path string `json:"path"`
}

type fileListResponse struct {
	Files      []fileItem `json:"files"`
	Editable   bool       `json:"editable"`
	SourceKind string     `json:"source_kind"`
}

type fileResponse struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	SHA256  string `json:"sha256"`
}

type saveRequestBody struct {
	Path           string `json:"path"`
	Content        string `json:"content"`
	ObservedSHA256 string `json:"observed_sha256"`
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
	rootInfo  os.FileInfo
	editable  bool
	ui        http.Handler
	saveMu    sync.Mutex
	allowedMu sync.RWMutex
	allowed   []string
}

func NewHandler(root string, editable bool) (http.Handler, error) {
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
	if len(definitions) == 0 {
		return nil, fmt.Errorf("local source contains no Workflow packages")
	}
	static, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, err
	}
	return &handler{
		root: filepath.Clean(root), rootInfo: info, editable: editable,
		ui: http.FileServer(http.FS(static)), allowed: allowedPathsFrom(definitions),
	}, nil
}

func (h *handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if strings.HasPrefix(request.URL.Path, "/api/") {
		if err := h.validateRoot(); err != nil {
			h.writeError(writer, http.StatusBadRequest, "invalid_path", err.Error(), "")
			return
		}
		unlock, err := workflow.LockSource(h.root, request.Method == http.MethodPost)
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
		case http.MethodPost:
			if !h.editable {
				h.writeError(writer, http.StatusMethodNotAllowed, "read_only", "The bound local source is read-only.", "")
				return
			}
			h.save(writer, request.Body)
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
	h.writeJSON(writer, http.StatusOK, fileListResponse{Files: items, Editable: h.editable, SourceKind: "local"})
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

func (h *handler) save(writer http.ResponseWriter, body io.Reader) {
	decoder := json.NewDecoder(io.LimitReader(body, 4<<20))
	decoder.DisallowUnknownFields()
	var request saveRequestBody
	if err := decoder.Decode(&request); err != nil {
		h.writeError(writer, http.StatusBadRequest, "invalid_request", err.Error(), "")
		return
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		h.writeError(writer, http.StatusBadRequest, "invalid_request", "Request must contain one JSON value.", "")
		return
	}
	h.saveMu.Lock()
	defer h.saveMu.Unlock()
	root, err := h.openRoot()
	if err != nil {
		h.writeError(writer, http.StatusBadRequest, "invalid_path", err.Error(), "")
		return
	}
	defer root.Close()

	path, err := h.resolve(request.Path)
	if err != nil {
		h.writeError(writer, http.StatusBadRequest, "invalid_path", err.Error(), "")
		return
	}
	current, err := os.ReadFile(path)
	if err != nil {
		h.writeError(writer, http.StatusBadRequest, "invalid_path", err.Error(), "")
		return
	}
	currentDigest := digest(current)
	if request.ObservedSHA256 != currentDigest {
		h.writeError(writer, http.StatusConflict, "stale_file", "The file changed after it was loaded.", currentDigest)
		return
	}
	if err := h.validateCandidate(request.Path, []byte(request.Content)); err != nil {
		h.writeError(writer, http.StatusUnprocessableEntity, "invalid_content", err.Error(), "")
		return
	}
	path, err = h.resolve(request.Path)
	if err != nil {
		h.writeError(writer, http.StatusBadRequest, "invalid_path", err.Error(), "")
		return
	}
	latest, err := os.ReadFile(path)
	if err != nil {
		h.writeError(writer, http.StatusBadRequest, "invalid_path", err.Error(), "")
		return
	}
	latestDigest := digest(latest)
	if latestDigest != currentDigest {
		h.writeError(writer, http.StatusConflict, "stale_file", "The file changed after it was loaded.", latestDigest)
		return
	}
	if err := h.validateRoot(); err != nil {
		h.writeError(writer, http.StatusBadRequest, "invalid_path", err.Error(), "")
		return
	}
	if err := atomicWriteAt(root, request.Path, []byte(request.Content)); err != nil {
		h.writeError(writer, http.StatusInternalServerError, "write_failed", err.Error(), "")
		return
	}
	h.writeJSON(writer, http.StatusOK, fileResponse{Path: request.Path, Content: request.Content, SHA256: digest([]byte(request.Content))})
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
		paths = append(paths, definition.Ref.ID+"/SKILL.md")
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
	if err := h.validateRoot(); err != nil {
		return "", err
	}
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

func (h *handler) validateCandidate(relative string, content []byte) error {
	if err := h.validateRoot(); err != nil {
		return err
	}
	parts := strings.Split(relative, "/")
	if len(parts) >= 4 && parts[1] == "skills" {
		if strings.TrimSpace(string(content)) == "" {
			return fmt.Errorf("SKILL.md must not be empty")
		}
	} else if len(parts) != 2 {
		return fmt.Errorf("path is not editable")
	}
	temporary, err := os.MkdirTemp("", "fanloop-dashboard-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	if err := os.CopyFS(temporary, os.DirFS(h.root)); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(temporary, filepath.FromSlash(relative)), content, 0o600); err != nil {
		return err
	}
	_, err = workflow.ListFrom(temporary)
	return err
}

func (h *handler) validateRoot() error {
	info, err := os.Lstat(h.root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || h.editable && !os.SameFile(info, h.rootInfo) {
		return fmt.Errorf("Workflow root changed after Dashboard startup")
	}
	return nil
}

func (h *handler) openRoot() (*os.File, error) {
	if err := h.validateRoot(); err != nil {
		return nil, err
	}
	root, err := os.Open(h.root)
	if err != nil {
		return nil, err
	}
	info, err := root.Stat()
	if err != nil || !info.IsDir() || h.editable && !os.SameFile(info, h.rootInfo) {
		_ = root.Close()
		return nil, fmt.Errorf("Workflow root changed after Dashboard startup")
	}
	return root, nil
}

func atomicWriteAt(root *os.File, relative string, content []byte) error {
	parts := strings.Split(relative, "/")
	if len(parts) < 2 {
		return fmt.Errorf("path is not allowed")
	}
	fd, err := unix.Dup(int(root.Fd()))
	if err != nil {
		return err
	}
	directory := os.NewFile(uintptr(fd), root.Name())
	defer func() { _ = directory.Close() }()
	for _, part := range parts[:len(parts)-1] {
		next, err := unix.Openat(int(directory.Fd()), part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		if err != nil {
			return err
		}
		if err := directory.Close(); err != nil {
			_ = unix.Close(next)
			return err
		}
		directory = os.NewFile(uintptr(next), part)
	}
	name := parts[len(parts)-1]
	var stat unix.Stat_t
	if err := unix.Fstatat(int(directory.Fd()), name, &stat, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return err
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFREG {
		return fmt.Errorf("path is not a regular allowlisted file")
	}
	var token [8]byte
	if _, err := cryptorand.Read(token[:]); err != nil {
		return err
	}
	temporaryName := ".dashboard-" + hex.EncodeToString(token[:])
	temporaryFD, err := unix.Openat(int(directory.Fd()), temporaryName, unix.O_CREAT|unix.O_EXCL|unix.O_WRONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, uint32(stat.Mode&0o777))
	if err != nil {
		return err
	}
	temporary := os.NewFile(uintptr(temporaryFD), temporaryName)
	defer func() { _ = unix.Unlinkat(int(directory.Fd()), temporaryName, 0) }()
	_, err = temporary.Write(content)
	if err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err := unix.Renameat(int(directory.Fd()), temporaryName, int(directory.Fd()), name); err != nil {
		return err
	}
	return directory.Sync()
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
