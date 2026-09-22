package dashboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestEmbeddedStudioBrowserPollingRaces(t *testing.T) {
	if testing.Short() {
		t.Skip("requires a local Playwright browser")
	}
	python := findPlaywrightPython()
	if python == "" {
		t.Skip("no Python interpreter with Playwright")
	}
	root := copyWorkflowTree(t)
	if err := os.RemoveAll(filepath.Join(root, "pstack-native")); err != nil {
		t.Fatal(err)
	}
	base, err := NewHandler(root)
	if err != nil {
		t.Fatal(err)
	}
	var firstTopology atomic.Bool
	var holdNextTopology atomic.Bool
	var topologyHeld atomic.Bool
	releaseTopology := make(chan struct{}, 1)
	var holdNextForcedTopology atomic.Bool
	var forcedTopologyHeld atomic.Bool
	releaseForcedTopology := make(chan struct{}, 1)
	var holdNextSave atomic.Bool
	var saveHeld atomic.Bool
	releaseSave := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/test/hold-next-topology" {
			holdNextTopology.Store(true)
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		if request.URL.Path == "/test/topology-held" {
			if !topologyHeld.Load() {
				writer.WriteHeader(http.StatusTooEarly)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		if request.URL.Path == "/test/release-topology" {
			select {
			case releaseTopology <- struct{}{}:
			default:
			}
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		if request.URL.Path == "/test/hold-next-forced-topology" {
			holdNextForcedTopology.Store(true)
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		if request.URL.Path == "/test/forced-topology-held" {
			if !forcedTopologyHeld.Load() {
				writer.WriteHeader(http.StatusTooEarly)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		if request.URL.Path == "/test/release-forced-topology" {
			select {
			case releaseForcedTopology <- struct{}{}:
			default:
			}
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		if request.URL.Path == "/test/hold-next-save" {
			holdNextSave.Store(true)
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		if request.URL.Path == "/test/save-held" {
			if !saveHeld.Load() {
				writer.WriteHeader(http.StatusTooEarly)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		if request.URL.Path == "/test/release-save" {
			select {
			case releaseSave <- struct{}{}:
			default:
			}
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		if request.URL.Path == "/test/insert" {
			path := filepath.Join(root, "fanloop-maintainer", "workflow.yaml")
			content, err := os.ReadFile(path)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			updated := []byte("# inserted before selected fragment\n" + string(content))
			if err := os.WriteFile(path, updated, 0o600); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		if request.URL.Path == "/test/corrupt" {
			path := filepath.Join(root, "fanloop-maintainer", "workflow.yaml")
			if err := os.WriteFile(path, []byte("invalid: true\n"), 0o600); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		if request.URL.Path == "/api/topology" && firstTopology.CompareAndSwap(false, true) {
			time.Sleep(2500 * time.Millisecond)
		}
		if request.URL.Path == "/api/topology" && request.URL.Query().Get("refresh") == "" && holdNextTopology.CompareAndSwap(true, false) {
			response := httptest.NewRecorder()
			base.ServeHTTP(response, request)
			topologyHeld.Store(true)
			<-releaseTopology
			for key, values := range response.Header() {
				for _, value := range values {
					writer.Header().Add(key, value)
				}
			}
			writer.WriteHeader(response.Code)
			_, _ = writer.Write(response.Body.Bytes())
			return
		}
		if request.URL.Path == "/api/topology" && request.URL.Query().Get("refresh") != "" && holdNextForcedTopology.CompareAndSwap(true, false) {
			response := httptest.NewRecorder()
			base.ServeHTTP(response, request)
			forcedTopologyHeld.Store(true)
			<-releaseForcedTopology
			for key, values := range response.Header() {
				for _, value := range values {
					writer.Header().Add(key, value)
				}
			}
			writer.WriteHeader(response.Code)
			_, _ = writer.Write(response.Body.Bytes())
			return
		}
		if request.URL.Path == "/api/file" && request.Method == http.MethodPost && holdNextSave.CompareAndSwap(true, false) {
			response := httptest.NewRecorder()
			base.ServeHTTP(response, request)
			saveHeld.Store(true)
			<-releaseSave
			for key, values := range response.Header() {
				for _, value := range values {
					writer.Header().Add(key, value)
				}
			}
			writer.WriteHeader(response.Code)
			_, _ = writer.Write(response.Body.Bytes())
			return
		}
		base.ServeHTTP(writer, request)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, python, "-c", browserRaceScript, server.URL, os.Getenv("FANLOOP_DASHBOARD_SCREENSHOT"))
	output, err := command.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatal(ctx.Err())
	}
	if err != nil {
		t.Fatalf("browser polling regression: %v\n%s", err, output)
	}
}

func findPlaywrightPython() string {
	if value := os.Getenv("FANLOOP_PLAYWRIGHT_PYTHON"); value != "" && pythonHasPlaywright(value) {
		return value
	}
	for _, name := range []string{"python3.14", "python3"} {
		if path, err := exec.LookPath(name); err == nil && pythonHasPlaywright(path) {
			return path
		}
	}
	return ""
}

func pythonHasPlaywright(path string) bool {
	command := exec.Command(path, "-c", "import playwright.sync_api")
	return command.Run() == nil
}

const browserRaceScript = `
import re
import sys
from playwright.sync_api import sync_playwright

with sync_playwright() as playwright:
    print("launch", flush=True)
    browser = playwright.chromium.launch(channel="chrome", headless=True)
    page = browser.new_page(viewport={"width": 1280, "height": 800})
    page.goto(sys.argv[1], wait_until="domcontentloaded")
    print("loaded", flush=True)
    page.wait_for_selector(".step-node", timeout=8000)
    page.wait_for_function("!document.querySelector('#editor').disabled", timeout=3000)
    page.evaluate("""() => {
      const nativeRequestAnimationFrame = window.requestAnimationFrame.bind(window)
      window.heldAnimationFrames = []
      window.holdNextAnimationFrame = false
      window.requestAnimationFrame = callback => {
        if (!window.holdNextAnimationFrame) return nativeRequestAnimationFrame(callback)
        window.holdNextAnimationFrame = false
        window.heldAnimationFrames.push(callback)
        return -window.heldAnimationFrames.length
      }
      window.releaseHeldAnimationFrames = () => {
        const callbacks = window.heldAnimationFrames.splice(0)
        callbacks.forEach(callback => callback(performance.now()))
      }
    }""")
    assert page.locator("#status").get_attribute("role") == "status"
    assert page.locator("#status").get_attribute("aria-live") == "polite"
    page.wait_for_function("parseInt(document.querySelector('#zoomValue').textContent, 10) >= 65", timeout=3000)
    assert page.locator(".step-node").first.bounding_box()["width"] >= 110
    page.select_option("#workflowSelect", "remnote-flashcards")
    page.wait_for_function("selectedWorkflow === 'remnote-flashcards' && document.querySelectorAll('.step-node').length === 4", timeout=5000)
    page.locator("#fit").click()
    page.wait_for_function("parseInt(document.querySelector('#zoomValue').textContent, 10) >= 80", timeout=3000)
    assert page.locator(".step-node").first.bounding_box()["width"] >= 137
    readable = page.locator(".step-node").first.locator("strong").evaluate("node => parseFloat(getComputedStyle(node).fontSize) * scale")
    assert readable >= 8
    for _ in range(7):
        page.locator("#zoomIn").click()
    viewport = page.locator("#viewport")
    assert viewport.evaluate("node => ['auto', 'scroll'].includes(getComputedStyle(node).overflowX)")
    assert viewport.evaluate("node => node.scrollWidth > node.clientWidth")
    viewport.evaluate("node => { node.scrollLeft = node.scrollWidth }")
    page.wait_for_function("document.querySelector('#viewport').scrollLeft > 0", timeout=3000)
    viewport_right = viewport.bounding_box()["x"] + viewport.bounding_box()["width"]
    last_stage = page.locator(".stage-band").last.bounding_box()
    done = page.locator("#terminalNode").bounding_box()
    assert last_stage["x"] + last_stage["width"] <= viewport_right + 1
    assert done["x"] + done["width"] <= viewport_right + 1
    page.select_option("#workflowSelect", "fanloop-maintainer")
    page.wait_for_function("selectedWorkflow === 'fanloop-maintainer' && document.querySelectorAll('.step-node').length === 4", timeout=5000)
    page.locator("#files button").filter(has_text="condition.yaml").click()
    page.wait_for_function("activeTab === 'file'")
    page.locator(".step-node.active").click()
    page.wait_for_function("""() =>
      activeTab === 'step' &&
      selectedFragment?.path.endsWith('/workflow.yaml') &&
      document.querySelector('#tabs button.active').dataset.tab === 'step' &&
      document.querySelector('#title').textContent.endsWith('/workflow.yaml')
    """)
    editor = page.locator("#editor")
    page.wait_for_function("document.querySelector('#editor').value.includes('{id: define_verification_contract')", timeout=5000)
    original = editor.input_value()
    renamed = re.sub(r"(\{id: define_verification_contract, name: )[^,]+", r"\1Dashboard 保存刷新", original, count=1)
    assert renamed != original
    page.wait_for_function("topologyRequest === null", timeout=5000)
    page.evaluate("async () => { await fetch('/test/hold-next-topology', {method: 'POST'}); void loadTopology(selectedWorkflow, false) }")
    page.wait_for_function("async () => (await fetch('/test/topology-held')).status === 204", timeout=5000)
    editor.fill(renamed)
    page.locator("#save").click()
    refreshed_before_release = True
    try:
        page.wait_for_function("document.querySelector('.step-node.active strong').textContent === 'Dashboard 保存刷新'", timeout=5000)
    except Exception:
        refreshed_before_release = False
    finally:
        page.request.post(page.url + "test/release-topology")
    if not refreshed_before_release:
        live = page.evaluate("async () => await (await fetch('/api/file?path=fanloop-maintainer%2Fworkflow.yaml')).json()")
        projected = page.evaluate("async () => await (await fetch('/api/topology?workflow=fanloop-maintainer')).json()")
        print("save-refresh", page.locator("#status").inner_text(), page.locator(".step-node.active strong").inner_text(), "disk", "Dashboard 保存刷新" in live["content"], "api", projected["workflow"]["stages"][0]["jobs"][0]["steps"][0]["name"], flush=True)
    assert refreshed_before_release
    page.wait_for_timeout(300)
    assert page.locator(".step-node.active strong").inner_text() == "Dashboard 保存刷新"
    page.request.post(page.url + "test/hold-next-forced-topology")
    editor.fill(renamed + "\n# response before topology")
    page.locator("#save").click()
    page.wait_for_function("async () => (await fetch('/test/forced-topology-held')).status === 204", timeout=5000)
    page.evaluate("window.holdNextAnimationFrame = true")
    page.request.post(page.url + "test/release-forced-topology")
    page.wait_for_function("window.heldAnimationFrames.length === 1", timeout=5000)
    editor.fill(renamed + "\n# edited after selection frame queued")
    refresh_selection = editor.evaluate("node => [node.selectionStart, node.selectionEnd]")
    page.evaluate("window.releaseHeldAnimationFrames()")
    page.wait_for_function("!document.querySelector('#save').disabled && document.querySelector('#status').textContent === 'Unsaved changes'", timeout=5000)
    assert editor.evaluate("node => [node.selectionStart, node.selectionEnd]") == refresh_selection
    page.request.post(page.url + "test/hold-next-save")
    editor.fill(renamed + "\n# saved request")
    page.locator("#save").click()
    page.wait_for_function("async () => (await fetch('/test/save-held')).status === 204", timeout=5000)
    editor.fill(renamed + "\n# still unsaved")
    dirty_selection = editor.evaluate("node => [node.selectionStart, node.selectionEnd]")
    page.request.post(page.url + "test/release-save")
    page.wait_for_function("!document.querySelector('#save').disabled && document.querySelector('#status').textContent === 'Unsaved changes'", timeout=5000)
    assert page.locator("#status").inner_text() == "Unsaved changes"
    assert page.locator("#save").is_enabled()
    assert editor.evaluate("node => [node.selectionStart, node.selectionEnd]") == dirty_selection
    editor.fill("not: [valid")
    page.locator("#save").click()
    page.wait_for_function("document.querySelector('#status').textContent !== 'Saving...'", timeout=5000)
    page.wait_for_timeout(2300)
    assert page.locator("#status").inner_text() != "Unsaved changes"
    assert page.locator("#status").inner_text() != "Saved"
    assert page.locator("#save").is_enabled()
    editor.fill(renamed)
    assert page.locator("#status").inner_text() == "Unsaved changes"
    page.locator("#discard").click()
    page.wait_for_function("document.querySelector('#status').textContent === 'Saved' && document.querySelector('#editor').value.endsWith('# saved request')", timeout=5000)
    assert not page.locator("#status").evaluate("node => node.classList.contains('error')")

    mobile = browser.new_page(viewport={"width": 390, "height": 844}, has_touch=True, is_mobile=True)
    mobile.goto(sys.argv[1], wait_until="domcontentloaded")
    mobile.wait_for_selector(".step-node", timeout=8000)
    mobile.select_option("#workflowSelect", "remnote-flashcards")
    mobile.wait_for_function("selectedWorkflow === 'remnote-flashcards' && document.querySelectorAll('.step-node').length === 4", timeout=5000)
    assert mobile.locator("#viewport").evaluate("node => getComputedStyle(node).touchAction") == "none"
    before = mobile.locator("#world").evaluate("node => node.style.transform")
    box = mobile.locator("#viewport").bounding_box()
    cdp = mobile.context.new_cdp_session(mobile)
    start = {"x": box["x"] + box["width"] / 2, "y": box["y"] + box["height"] / 2}
    cancel_start = {"x": box["x"] + 12, "y": box["y"] + 24}
    cdp.send("Input.dispatchTouchEvent", {"type": "touchStart", "touchPoints": [start]})
    cdp.send("Input.dispatchTouchEvent", {"type": "touchMove", "touchPoints": [{"x": start["x"] + 55, "y": start["y"] + 35}]})
    cdp.send("Input.dispatchTouchEvent", {"type": "touchEnd", "touchPoints": []})
    mobile.wait_for_function("before => document.querySelector('#world').style.transform !== before", arg=before, timeout=3000)
    assert not mobile.locator("#viewport").evaluate("node => node.classList.contains('dragging')")
    cdp.send("Input.dispatchTouchEvent", {"type": "touchStart", "touchPoints": [cancel_start]})
    mobile.wait_for_function("document.querySelector('#viewport').classList.contains('dragging')", timeout=3000)
    cdp.send("Input.dispatchTouchEvent", {"type": "touchCancel", "touchPoints": []})
    mobile.wait_for_function("!document.querySelector('#viewport').classList.contains('dragging')", timeout=3000)
    mobile.close()
    if sys.argv[2]:
        page.screenshot(path=sys.argv[2], full_page=True)
    print("editor", flush=True)
    page.evaluate("fetch('/test/insert', {method: 'POST'})")
    page.wait_for_function("document.querySelector('#editor').value.startsWith('# inserted before selected fragment')", timeout=5000)
    page.wait_for_function("""() => {
      const editor = document.querySelector('#editor')
      const expected = lineOffsets(editor.value, selectedFragment)
      return editor.selectionStart === expected.start && editor.selectionEnd === expected.end
    }""", timeout=5000)
    print("reselected", flush=True)
    original = editor.input_value()
    editor.fill(original + "\nlocal dirty text")
    page.evaluate("fetch('/test/corrupt', {method: 'POST'})")
    page.wait_for_function("document.body.classList.contains('is-stale')", timeout=5000)
    print("stale", flush=True)
    assert editor.input_value().endswith("local dirty text")
    assert page.locator("#save").is_disabled()
    browser.close()
`
