package examples

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

func TestStandaloneControlPageSettingBearerTokenHidesRawValue(t *testing.T) {
	page := newStandaloneControlPage(t)

	page.setValue("bearer-token-input", "super-secret-token")
	page.click("save-bearer-token")

	if got, want := page.text("token-status"), "Bearer token set"; got != want {
		t.Fatalf("token status = %q, want %q", got, want)
	}
	if got := page.value("bearer-token-input"); got != "" {
		t.Fatalf("token input value = %q, want empty after save", got)
	}
	if strings.Contains(page.visibleText(), "super-secret-token") {
		t.Fatal("visible page text exposed configured bearer token")
	}
}

func TestStandaloneControlPageRequestsIncludeConfiguredBearerToken(t *testing.T) {
	page := newStandaloneControlPage(t)

	page.queueJSONResponse(200, `{"status":"ok"}`)
	page.setValue("bearer-token-input", "token-123")
	page.click("save-bearer-token")
	page.click("load-health")

	request := page.fetchCall(0)
	if got, want := request.URL, "http://127.0.0.1:8080/healthz"; got != want {
		t.Fatalf("request URL = %q, want %q", got, want)
	}
	if got, want := request.Method, "GET"; got != want {
		t.Fatalf("request method = %q, want %q", got, want)
	}
	if got, want := request.Headers["Authorization"], "Bearer token-123"; got != want {
		t.Fatalf("Authorization header = %q, want %q", got, want)
	}
}

func TestStandaloneControlPageConfiguredAPITargetAppliesToAllActions(t *testing.T) {
	page := newStandaloneControlPage(t)

	page.setValue("api-base-url", "127.0.0.1:18080/api///")
	page.click("save-base-url")

	if got, want := page.text("active-api-target"), "Active target: http://127.0.0.1:18080/api"; got != want {
		t.Fatalf("active target text = %q, want %q", got, want)
	}

	page.queueJSONResponse(200, `{"status":"ok"}`)
	page.queueJSONResponse(200, `{"status":"ok"}`)
	page.queueJSONResponse(200, `{"status":"idle"}`)
	page.queueJSONResponse(200, `{"samples":[]}`)
	page.queueJSONResponse(200, `{"status":"started"}`)
	page.queueJSONResponse(200, `{"status":"altered"}`)
	page.queueJSONResponse(200, `{"status":"stopped"}`)
	page.queueTextResponse(200, "metric_one 1\n")
	page.queueTextResponse(200, "metric_one 1\n")

	page.setValue("start-scale", "10")
	page.setValue("start-clients", "3")
	page.setValue("start-duration", "60")
	page.setValue("start-warmup", "10")
	page.setValue("start-profile", "mixed")
	page.setValue("start-read-percent", "80")
	page.setValue("start-target-tps", "120")
	page.setChecked("start-reset", true)
	page.setValue("alter-clients", "4")

	page.click("load-health")
	page.click("load-ready")
	page.click("load-state")
	page.click("load-results")
	page.submit("start-form")
	page.submit("alter-form")
	page.click("stop-benchmark")
	page.click("fetch-metrics")
	page.click("open-metrics")

	wantCalls := []recordedFetchCall{
		{Method: "GET", URL: "http://127.0.0.1:18080/api/healthz"},
		{Method: "GET", URL: "http://127.0.0.1:18080/api/readyz"},
		{Method: "GET", URL: "http://127.0.0.1:18080/api/benchmark"},
		{Method: "GET", URL: "http://127.0.0.1:18080/api/benchmark/results"},
		{Method: "POST", URL: "http://127.0.0.1:18080/api/benchmark/start"},
		{Method: "POST", URL: "http://127.0.0.1:18080/api/benchmark/alter"},
		{Method: "POST", URL: "http://127.0.0.1:18080/api/benchmark/stop"},
		{Method: "GET", URL: "http://127.0.0.1:18080/api/metrics"},
		{Method: "GET", URL: "http://127.0.0.1:18080/api/metrics"},
	}
	if got, want := page.fetchCallCount(), len(wantCalls); got != want {
		t.Fatalf("fetch call count = %d, want %d", got, want)
	}

	for index, want := range wantCalls {
		call := page.fetchCall(index)
		if call.Method != want.Method || call.URL != want.URL {
			t.Fatalf("fetch call %d = %s %s, want %s %s", index, call.Method, call.URL, want.Method, want.URL)
		}
	}

	if got, want := page.openCallCount(), 1; got != want {
		t.Fatalf("window.open call count = %d, want %d", got, want)
	}
}

func TestStandaloneControlPageClearingAndReplacingBearerTokenChangesRequestHeaders(t *testing.T) {
	page := newStandaloneControlPage(t)

	page.queueJSONResponse(200, `{"status":"ok"}`)
	page.queueJSONResponse(200, `{"status":"ok"}`)
	page.queueJSONResponse(200, `{"status":"ok"}`)

	page.setValue("bearer-token-input", "first-token")
	page.click("save-bearer-token")
	page.click("load-health")

	page.setValue("bearer-token-input", "second-token")
	page.click("save-bearer-token")
	page.click("load-ready")

	page.click("clear-bearer-token")
	page.click("load-state")

	if got, want := page.fetchCall(0).Headers["Authorization"], "Bearer first-token"; got != want {
		t.Fatalf("first request Authorization header = %q, want %q", got, want)
	}
	if got, want := page.fetchCall(1).Headers["Authorization"], "Bearer second-token"; got != want {
		t.Fatalf("second request Authorization header = %q, want %q", got, want)
	}
	if got := page.fetchCall(2).Headers["Authorization"]; got != "" {
		t.Fatalf("cleared token Authorization header = %q, want empty", got)
	}
	if got, want := page.text("token-status"), "Bearer token not set"; got != want {
		t.Fatalf("token status after clear = %q, want %q", got, want)
	}
}

type standaloneControlPage struct {
	t       *testing.T
	runtime *goja.Runtime
}

type recordedFetchCall struct {
	URL     string
	Method  string
	Body    string
	Headers map[string]string
}

func newStandaloneControlPage(t *testing.T) *standaloneControlPage {
	t.Helper()

	sourcePath := filepath.Join("standalone-control-page.html")
	sourceBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read standalone page: %v", err)
	}
	source := string(sourceBytes)

	scriptPattern := regexp.MustCompile(`(?s)<script>\s*(.*?)\s*</script>`)
	scriptMatch := scriptPattern.FindStringSubmatch(source)
	if scriptMatch == nil {
		t.Fatal("standalone page does not contain inline script")
	}

	idPattern := regexp.MustCompile(`id="([^"]+)"`)
	idMatches := idPattern.FindAllStringSubmatch(source, -1)
	ids := make([]string, 0, len(idMatches))
	for _, match := range idMatches {
		ids = append(ids, match[1])
	}

	idsJSON, err := json.Marshal(ids)
	if err != nil {
		t.Fatalf("marshal element ids: %v", err)
	}

	runtime := goja.New()
	prelude := `
var __pageElementIds = ` + string(idsJSON) + `;
var __elements = {};
var __fetchCalls = [];
var __fetchQueue = [];
var __openCalls = [];

function __makeElement(id) {
  return {
    id: id,
    value: "",
    textContent: "",
    checked: false,
    disabled: false,
    dataset: {},
    listeners: {},
    addEventListener: function(type, listener) {
      if (!this.listeners[type]) {
        this.listeners[type] = [];
      }
      this.listeners[type].push(listener);
    },
    dispatchEvent: function(type) {
      var listeners = this.listeners[type] || [];
      var event = {
        type: type,
        defaultPrevented: false,
        preventDefault: function() {
          this.defaultPrevented = true;
        }
      };
      for (var index = 0; index < listeners.length; index += 1) {
        listeners[index](event);
      }
      return event;
    }
  };
}

for (var index = 0; index < __pageElementIds.length; index += 1) {
  __elements[__pageElementIds[index]] = __makeElement(__pageElementIds[index]);
}

var document = {
  getElementById: function(id) {
    return Object.prototype.hasOwnProperty.call(__elements, id) ? __elements[id] : null;
  }
};

var window = {
  localStorage: {
    __store: {},
    getItem: function(key) {
      return Object.prototype.hasOwnProperty.call(this.__store, key) ? this.__store[key] : null;
    },
    setItem: function(key, value) {
      this.__store[key] = String(value);
    }
  },
  fetch: function(url, requestInit) {
    __fetchCalls.push({
      url: String(url),
      method: requestInit && requestInit.method ? String(requestInit.method) : "GET",
      body: requestInit && requestInit.body ? String(requestInit.body) : "",
      headers: requestInit && requestInit.headers ? requestInit.headers : {}
    });
    if (__fetchQueue.length === 0) {
      throw new Error("unexpected fetch call with empty response queue");
    }
    return __fetchQueue.shift();
  },
  open: function(url, target, features) {
    __openCalls.push({
      url: String(url),
      target: target ? String(target) : "",
      features: features ? String(features) : ""
    });
    return {};
  }
};

function __click(id) {
  var element = document.getElementById(id);
  if (element === null) {
    throw new Error("missing element: " + id);
  }
  element.dispatchEvent("click");
}

function __setValue(id, value) {
  var element = document.getElementById(id);
  if (element === null) {
    throw new Error("missing element: " + id);
  }
  element.value = value;
}

function __setChecked(id, checked) {
  var element = document.getElementById(id);
  if (element === null) {
    throw new Error("missing element: " + id);
  }
  element.checked = Boolean(checked);
}

function __getValue(id) {
  var element = document.getElementById(id);
  if (element === null) {
    throw new Error("missing element: " + id);
  }
  return element.value;
}

function __getText(id) {
  var element = document.getElementById(id);
  if (element === null) {
    throw new Error("missing element: " + id);
  }
  return element.textContent;
}

function __submit(id) {
  var element = document.getElementById(id);
  if (element === null) {
    throw new Error("missing element: " + id);
  }
  element.dispatchEvent("submit");
}

function __visibleText() {
  var parts = [];
  for (var index = 0; index < __pageElementIds.length; index += 1) {
    var text = __elements[__pageElementIds[index]].textContent;
    if (text) {
      parts.push(text);
    }
  }
  return parts.join("\n");
}

function __makeHeaders(contentType) {
  return {
    get: function(name) {
      if (String(name).toLowerCase() === "content-type") {
        return contentType || "";
      }
      return "";
    }
  };
}

function __queueFetchResponse(status, statusText, contentType, bodyText) {
  __fetchQueue.push({
    ok: status >= 200 && status < 300,
    status: status,
    statusText: statusText,
    headers: __makeHeaders(contentType),
    text: async function() {
      return bodyText;
    }
  });
}

function __getFetchCall(index) {
  return __fetchCalls[index];
}

function __getFetchCallCount() {
  return __fetchCalls.length;
}

function __getOpenCallCount() {
  return __openCalls.length;
}
`

	if _, err := runtime.RunString(prelude); err != nil {
		t.Fatalf("run standalone page test prelude: %v", err)
	}
	if _, err := runtime.RunString(scriptMatch[1]); err != nil {
		t.Fatalf("run standalone page script: %v", err)
	}

	return &standaloneControlPage{
		t:       t,
		runtime: runtime,
	}
}

func (page *standaloneControlPage) click(id string) {
	page.t.Helper()
	page.mustRun(`__click("` + id + `")`)
}

func (page *standaloneControlPage) setValue(id string, value string) {
	page.t.Helper()
	page.mustRun(`__setValue("` + id + `", ` + mustJSONString(page.t, value) + `)`)
}

func (page *standaloneControlPage) value(id string) string {
	page.t.Helper()
	return page.mustRun(`__getValue("` + id + `")`).String()
}

func (page *standaloneControlPage) setChecked(id string, checked bool) {
	page.t.Helper()
	page.mustRun(`__setChecked("` + id + `", ` + mustJSONBool(page.t, checked) + `)`)
}

func (page *standaloneControlPage) text(id string) string {
	page.t.Helper()
	return page.mustRun(`__getText("` + id + `")`).String()
}

func (page *standaloneControlPage) visibleText() string {
	page.t.Helper()
	return page.mustRun(`__visibleText()`).String()
}

func (page *standaloneControlPage) submit(id string) {
	page.t.Helper()
	page.mustRun(`__submit("` + id + `")`)
}

func (page *standaloneControlPage) queueJSONResponse(status int, body string) {
	page.t.Helper()
	page.mustRun(`__queueFetchResponse(` + mustJSONInt(page.t, status) + `, "OK", "application/json", ` + mustJSONString(page.t, body) + `)`)
}

func (page *standaloneControlPage) queueTextResponse(status int, body string) {
	page.t.Helper()
	page.mustRun(`__queueFetchResponse(` + mustJSONInt(page.t, status) + `, "OK", "text/plain", ` + mustJSONString(page.t, body) + `)`)
}

func (page *standaloneControlPage) fetchCall(index int) recordedFetchCall {
	page.t.Helper()
	value := page.mustRun(`JSON.stringify(__getFetchCall(` + mustJSONInt(page.t, index) + `))`)

	var call recordedFetchCall
	if err := json.Unmarshal([]byte(value.String()), &call); err != nil {
		page.t.Fatalf("decode fetch call: %v", err)
	}
	return call
}

func (page *standaloneControlPage) fetchCallCount() int {
	page.t.Helper()
	return int(page.mustRun(`__getFetchCallCount()`).ToInteger())
}

func (page *standaloneControlPage) openCallCount() int {
	page.t.Helper()
	return int(page.mustRun(`__getOpenCallCount()`).ToInteger())
}

func (page *standaloneControlPage) mustRun(script string) goja.Value {
	page.t.Helper()
	value, err := page.runtime.RunString(script)
	if err != nil {
		page.t.Fatalf("run test script %q: %v", script, err)
	}
	return value
}

func mustJSONString(t *testing.T, value string) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal string %q: %v", value, err)
	}
	return string(encoded)
}

func mustJSONInt(t *testing.T, value int) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal int %d: %v", value, err)
	}
	return string(encoded)
}

func mustJSONBool(t *testing.T, value bool) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal bool %t: %v", value, err)
	}
	return string(encoded)
}
