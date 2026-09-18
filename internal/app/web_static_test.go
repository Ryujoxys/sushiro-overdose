package app

import (
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// 前端契约守卫：indexHTML 由 webui/ 下 HTML/CSS/JS 在 init 时组装而成，go build
// 只验证嵌入文件存在、不校验内容。下面的测试对其做静态结构检查，让"重排丢
// id、删函数后 onclick 还在调、id 撞车、JS 语法错"这类回归在 go test 阶段就报红，
// 而不是等用户打开页面才暴露。检查逻辑与 architecture_guard_test.go 同一风格（静态
// 扫描源码字符串，零外部依赖）。

func extractEmbeddedScript(t *testing.T) string {
	t.Helper()
	const open, closeTag = "<script>", "</script>"
	i := strings.Index(indexHTML, open)
	j := strings.LastIndex(indexHTML, closeTag)
	if i < 0 || j < 0 || j <= i {
		t.Fatalf("indexHTML 中找不到成对的 <script> 段")
	}
	return indexHTML[i+len(open) : j]
}

func extractBetween(t *testing.T, s, start, end string) string {
	t.Helper()
	i := strings.Index(s, start)
	if i < 0 {
		t.Fatalf("找不到片段起点：%s", start)
	}
	j := strings.Index(s[i:], end)
	if j < 0 {
		t.Fatalf("找不到片段终点：%s", end)
	}
	return s[i : i+j]
}

// satisfiedDOMIDs 收集所有"存在的" id：模板里的 id="X"，以及 JS 里动态创建的 .id='X'。
func satisfiedDOMIDs() map[string]bool {
	ids := map[string]bool{}
	for _, m := range regexp.MustCompile(`id="([\w-]+)"`).FindAllStringSubmatch(indexHTML, -1) {
		ids[m[1]] = true
	}
	for _, m := range regexp.MustCompile(`\.id\s*=\s*['"]([\w-]+)['"]`).FindAllStringSubmatch(indexHTML, -1) {
		ids[m[1]] = true
	}
	return ids
}

// TestEmbeddedDOMIDReferencesResolve 确保 JS 里 el('X') / getElementById('X') 引用的
// 每个静态字面量 id，都能在模板中找到 id="X" 或在 JS 中被动态创建。挡住"重排/改名丢 id"。
func TestEmbeddedDOMIDReferencesResolve(t *testing.T) {
	satisfied := satisfiedDOMIDs()
	refRe := []*regexp.Regexp{
		regexp.MustCompile(`\bel\('([\w-]+)'\)`),
		regexp.MustCompile(`getElementById\('([\w-]+)'\)`),
	}
	missing := map[string]bool{}
	for _, re := range refRe {
		for _, m := range re.FindAllStringSubmatch(indexHTML, -1) {
			if !satisfied[m[1]] {
				missing[m[1]] = true
			}
		}
	}
	if len(missing) > 0 {
		t.Fatalf("JS 引用了不存在的 DOM id（模板里没有 id=\"...\"、JS 里也没动态创建）：%s\n"+
			"如果是重排或改名导致，请补回对应元素 id；如确为动态创建，请用 element.id='...' 赋值。", sortedKeys(missing))
	}
}

// TestEmbeddedOnclickHandlersDefined 确保每个 onclick="fn(...)" 的首个调用函数 fn 在
// 脚本里有定义（function 声明或赋值/箭头）。挡住"删了/改名了函数但 HTML 还在调"。
func TestEmbeddedOnclickHandlersDefined(t *testing.T) {
	js := extractEmbeddedScript(t)
	defined := map[string]bool{}
	for _, m := range regexp.MustCompile(`function\s+([a-zA-Z_$][\w$]*)\s*\(`).FindAllStringSubmatch(js, -1) {
		defined[m[1]] = true
	}
	for _, m := range regexp.MustCompile(`\b([a-zA-Z_$][\w$]*)\s*=\s*(?:async\s+)?(?:function|\()`).FindAllStringSubmatch(js, -1) {
		defined[m[1]] = true
	}

	leadCall := regexp.MustCompile(`^\s*([a-zA-Z_$][\w$]*)\(`)
	undef := map[string]bool{}
	for _, m := range regexp.MustCompile(`onclick="([^"]*)"`).FindAllStringSubmatch(indexHTML, -1) {
		lc := leadCall.FindStringSubmatch(m[1])
		if lc == nil {
			continue // 非直接函数调用（赋值、成员表达式等）不在此检查范围
		}
		if !defined[lc[1]] {
			undef[lc[1]] = true
		}
	}
	if len(undef) > 0 {
		t.Fatalf("onclick 调用了脚本中未定义的函数：%s\n"+
			"如果改名/删除了函数，请同步更新对应 onclick。", sortedKeys(undef))
	}
}

// TestEmbeddedDOMIDsUnique 确保模板里没有重复的 id="X"。挡住 id 撞车导致 el() 取错元素。
func TestEmbeddedDOMIDsUnique(t *testing.T) {
	counts := map[string]int{}
	for _, m := range regexp.MustCompile(`id="([\w-]+)"`).FindAllStringSubmatch(indexHTML, -1) {
		counts[m[1]]++
	}
	dups := map[string]bool{}
	for id, n := range counts {
		if n > 1 {
			dups[id] = true
		}
	}
	if len(dups) > 0 {
		t.Fatalf("模板里存在重复 id：%s", sortedKeys(dups))
	}
}

func TestEmbeddedJavaScriptSyntax(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		if node, err = exec.LookPath("nodejs"); err != nil {
			t.Skip("未找到 node，跳过 JS 语法检查")
		}
	}
	js := extractEmbeddedScript(t)
	f, err := os.CreateTemp("", "sushiro-web-*.js")
	if err != nil {
		t.Fatalf("创建临时文件失败：%v", err)
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(js); err != nil {
		t.Fatalf("写临时文件失败：%v", err)
	}
	f.Close()
	if out, err := exec.Command(node, "--check", f.Name()).CombinedOutput(); err != nil {
		t.Fatalf("内嵌 JS 未通过 node --check：\n%s", out)
	}
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestEmbeddedRecordingFirstSurface(t *testing.T) {
	for _, id := range []string{"page-records", "record-stores", "record-toggle", "record-chart", "analysis-store", "store-dialog", "ticket-dialog", "auth-dialog"} {
		if !satisfiedDOMIDs()[id] {
			t.Errorf("missing %s", id)
		}
	}
	for _, forbidden := range []string{"startQuickTicket", "uiModeSwitch", "snRows", "/api/cloud", "/api/sniper/", "/api/engine/booking", "/api/queue/ticket/plan", "/api/queue/ticket/routine"} {
		if strings.Contains(indexHTML, forbidden) {
			t.Errorf("retired feature in UI: %s", forbidden)
		}
	}
	for _, required := range []string{"我的记录", "手动取号", "想几点吃", "name=\"sushiro-csrf\"", "/api/records", "/api/queue/service", "/api/queue/ticket", "/api/engine/capture"} {
		if !strings.Contains(indexHTML, required) {
			t.Errorf("missing %s", required)
		}
	}
}

func TestWebUIBehavior(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	cmd := exec.Command(node, "../../scripts/test-webui-records.mjs")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}
