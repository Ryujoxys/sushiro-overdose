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

// TestEmbeddedCriticalAnchors 冒烟检查：核心面板/锚点必须存在，防止整页结构被误删。
func TestEmbeddedCriticalAnchors(t *testing.T) {
	satisfied := satisfiedDOMIDs()
	for _, id := range []string{"qdPressChart", "qdAnswer", "qdAdvisor", "qtLive", "qdTargetNo", "ntStore", "snRows", "rc", "lv", "toastWrap", "confirmOv"} {
		if !satisfied[id] {
			t.Errorf("缺少关键锚点 id=%q（模板或动态创建）", id)
		}
	}
	for _, needle := range []string{`name="sushiro-csrf"`, "function toast(", "function confirmDialog("} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少关键片段：%s", needle)
		}
	}
}

func TestEmbeddedUXCommandCenterAnchors(t *testing.T) {
	satisfied := satisfiedDOMIDs()
	for _, id := range []string{"journeyPanel", "diagNext"} {
		if !satisfied[id] {
			t.Errorf("缺少体验指挥台锚点 id=%q", id)
		}
	}
	for _, needle := range []string{
		"function renderJourneyPanel(",
		"function diagnosticAdvice(",
		"journeyStepHTML('read','只读'",
		"journeyStepHTML('auth','通行证'",
		"journeyStepHTML('action','会执行'",
		"今天该走哪条路",
		"先处理这件事",
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少体验指挥台片段：%s", needle)
		}
	}
}

func TestEmbeddedHomeDecisionOnboarding(t *testing.T) {
	satisfied := satisfiedDOMIDs()
	for _, id := range []string{"homeDecisionPanel", "journeyPanel"} {
		if !satisfied[id] {
			t.Errorf("缺少首页决策入口锚点 id=%q", id)
		}
	}
	if satisfied["mechanismMap"] {
		t.Errorf("首页不应再保留独立 mechanismMap；机制说明应合并进 homeDecisionPanel，避免和 journeyPanel 重复")
	}
	for _, needle := range []string{
		"你现在是哪种情况",
		"今天去吃",
		"我有当天排队号",
		"想约未来某天",
		"看排队和预测不用登录",
		`class="home-decision-card read" onclick="go('qt')"`,
		`class="home-decision-card read" onclick="go('qd')"`,
		`class="home-decision-card auth" onclick="currentUIMode()==='advanced'?go('ca'):enterAdvanced('ca')"`,
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少首页决策说明片段：%s", needle)
		}
	}
	hero := strings.Index(indexHTML, `id="heroBox"`)
	decision := strings.Index(indexHTML, `id="homeDecisionPanel"`)
	journey := strings.Index(indexHTML, `id="journeyPanel"`)
	live := strings.Index(indexHTML, `id="homeLive"`)
	if hero < 0 || decision < 0 || journey < 0 || live < 0 {
		t.Fatalf("首页关键区块索引异常：hero=%d decision=%d journey=%d live=%d", hero, decision, journey, live)
	}
	if !(hero < decision && decision < journey && journey < live) {
		t.Fatalf("首页首屏顺序应为 hero -> 决策入口 -> 状态建议 -> 实时排队：hero=%d decision=%d journey=%d live=%d", hero, decision, journey, live)
	}
	if strings.Contains(indexHTML, "现在想去吃") {
		t.Fatalf("首页术语应统一为“现在去吃”，不要保留“现在想去吃”")
	}
}

func TestEmbeddedHomeHeroKeepsReadOnlyFirst(t *testing.T) {
	hero := extractBetween(t, indexHTML, `<div class="hero" id="heroBox">`, `<div id="homeDecisionPanel"`)
	for _, needle := range []string{
		`id="heroReadOnlyPrimary"`,
		`onclick="openGuestStorePicker()"`,
		"选门店看排队",
		"先看机制图",
		"不用登录、不用通行证",
	} {
		if !strings.Contains(hero, needle) {
			t.Errorf("首页 hero 应先给只读入口，缺少：%s", needle)
		}
	}
	for _, forbidden := range []string{
		`id="bc" onclick="startAuth()"`,
		"我要抢预约：获取通行证",
	} {
		if strings.Contains(hero, forbidden) {
			t.Fatalf("首页 hero 不应把通行证作为首屏主动作：%s", forbidden)
		}
	}

	js := extractEmbeddedScript(t)
	us := extractBetween(t, js, "function uD()", "function uE()")
	for _, needle := range []string{
		"pick.classList.remove('hid')",
		"bc.classList.add('hid')",
	} {
		if !strings.Contains(us, needle) {
			t.Errorf("首页状态渲染应保持只读优先，缺少：%s", needle)
		}
	}
	if strings.Contains(us, "我要抢预约：获取通行证") {
		t.Fatalf("首次使用状态不应再把获取通行证作为 hero 主按钮")
	}
}

func TestEmbeddedSetupCardPrioritizesQueueOverGuide(t *testing.T) {
	setup := extractBetween(t, indexHTML, `<div class="card" id="setupCard">`, `</details>`)
	for _, needle := range []string{
		`<button class="bt bt-r bt-s" onclick="openGuestStorePicker()">选门店看排队</button>`,
		`<button class="bt bt-w bt-s" onclick="openFirstUseWizard()">新手引导</button>`,
		`<button class="bt bt-w bt-s" onclick="go('gu')">机制图</button>`,
	} {
		if !strings.Contains(setup, needle) {
			t.Errorf("准备清单按钮应先给只读排队入口，缺少：%s", needle)
		}
	}
	firstQueue := strings.Index(setup, "选门店看排队")
	firstGuide := strings.Index(setup, "新手引导")
	if firstQueue < 0 || firstGuide < 0 || firstQueue > firstGuide {
		t.Fatalf("准备清单应先给“选门店看排队”，再给新手引导：queue=%d guide=%d", firstQueue, firstGuide)
	}

	js := extractEmbeddedScript(t)
	render := extractBetween(t, js, "function renderSetupCard()", "function journeyStepHTML")
	storeIdx := strings.Index(render, "items.push({t:'常用门店'")
	authIdx := strings.Index(render, "items.push({t:'寿司郎通行证")
	if storeIdx < 0 || authIdx < 0 {
		t.Fatalf("准备清单渲染缺少常用门店或通行证项：store=%d auth=%d", storeIdx, authIdx)
	}
	if storeIdx > authIdx {
		t.Fatalf("准备清单应先选常用门店，再提示通行证：store=%d auth=%d", storeIdx, authIdx)
	}
}

func TestEmbeddedBeginnerGuideMechanismPage(t *testing.T) {
	satisfied := satisfiedDOMIDs()
	for _, id := range []string{"p-gu", "guideFlow", "guideActionLegend"} {
		if !satisfied[id] {
			t.Errorf("缺少新手机制页锚点 id=%q", id)
		}
	}
	for _, needle := range []string{
		"{id:'home',label:'首页',pages:[['da','概览'],['gu','新手入门']]}",
		"else{go('da',null,true);loadHomeLive(true);maybeShowIntro()}",
		"新手入门：寿司郎排队机制",
		"今天去吃",
		"约未来",
		"当天排队号",
		"通行证不是排队号",
		"只读 · 直接用",
		"会执行操作",
		`onclick="go('gu')"`,
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少新手机制页片段：%s", needle)
		}
	}
	home := strings.Index(indexHTML, `id="p-da"`)
	guide := strings.Index(indexHTML, `id="p-gu"`)
	calendar := strings.Index(indexHTML, `id="p-ca"`)
	if home < 0 || guide < 0 || calendar < 0 {
		t.Fatalf("页面索引异常：home=%d guide=%d calendar=%d", home, guide, calendar)
	}
	if !(home < guide && guide < calendar) {
		t.Fatalf("新手机制页应放在首页之后、进阶页面之前：home=%d guide=%d calendar=%d", home, guide, calendar)
	}
}

func TestEmbeddedBeginnerGuideEndsWithDecisionActions(t *testing.T) {
	satisfied := satisfiedDOMIDs()
	if !satisfied["guideNextActions"] {
		t.Fatalf("新手入门页应有看完流程图后的去向卡 guideNextActions")
	}
	guide := extractBetween(t, indexHTML, `<section id="p-gu"`, `<section id="p-ca"`)
	for _, needle := range []string{
		`id="guideNextActions"`,
		"看完流程图，下一步去哪",
		"先看今天排队",
		"算这个号几点能吃上",
		"查未来可约日历",
		`onclick="go('qt')"`,
		`onclick="go('qd')"`,
		`onclick="currentUIMode()==='advanced'?go('ca'):enterAdvanced('ca')"`,
	} {
		if !strings.Contains(guide, needle) {
			t.Errorf("新手入门页去向卡缺少：%s", needle)
		}
	}
	if strings.Contains(guide, `<button class="bt bt-o" onclick="startAuth()">获取通行证</button>`) {
		t.Fatalf("新手入门页末尾不应直接把获取通行证作为并列主动作；应先按用户场景分流")
	}
}

func TestEmbeddedBeginnerGuideFlowHasInlineActions(t *testing.T) {
	guide := extractBetween(t, indexHTML, `<section id="p-gu"`, `<section id="p-ca"`)
	for _, needle := range []string{
		`class="flow-lane-actions"`,
		`onclick="openGuestStorePicker()"`,
		`onclick="currentUIMode()==='advanced'?go('ca'):enterAdvanced('ca')"`,
		"从看排队开始",
		"先查未来日历",
	} {
		if !strings.Contains(guide, needle) {
			t.Errorf("新手流程图标题旁缺少直接行动入口：%s", needle)
		}
	}
	if strings.Contains(guide, "获取通行证</button>") && !strings.Contains(guide, "先查未来日历") {
		t.Fatalf("新手流程图不应把获取通行证作为第一主动作；应先让用户按场景进入")
	}
}

func TestEmbeddedFirstUseWizardStartsWithThreeLightChoices(t *testing.T) {
	js := extractEmbeddedScript(t)
	wizard := extractBetween(t, js, "function openFirstUseWizard()", "function closeFirstUseWizard()")
	for _, needle := range []string{
		"第一次用，先选一条路",
		"选门店看排队",
		"我有号码",
		"算这个号几点能吃上",
		"想约未来",
		"查未来预约",
		"closeFirstUseWizard();openGuestStorePicker()",
		"closeFirstUseWizard();go(\\'qd\\')",
		"closeFirstUseWizard();currentUIMode()===\\'advanced\\'?go(\\'ca\\'):enterAdvanced(\\'ca\\')",
	} {
		if !strings.Contains(wizard, needle) {
			t.Errorf("首启浮层应是轻量三选一，缺少：%s", needle)
		}
	}
	if !strings.Contains(wizard, "先看机制图") || !strings.Contains(wizard, "closeFirstUseWizard();go(\\'gu\\')") {
		t.Fatalf("首启浮层仍应保留机制图入口，但只能作为辅助动作")
	}
	if strings.Contains(wizard, `class="first-use-card primary" type="button" onclick="closeFirstUseWizard();go(\'gu\')"`) {
		t.Fatalf("首启浮层不应把“先看机制图”作为三张主卡之一，应优先按真实场景分流")
	}
	css := extractBetween(t, indexHTML, `<style>`, `</style>`)
	if !strings.Contains(css, ".first-use-card.auth span") {
		t.Fatalf("首启浮层的未来预约卡应有 auth 视觉标识")
	}
	for _, forbidden := range []string{
		"欢迎来吃寿司",
		"想吃寿司郎？先看看现在排多久",
		"需要通行证 🎫",
		"firstUseGo('ca',true)",
	} {
		if strings.Contains(wizard, forbidden) {
			t.Fatalf("首启浮层不应在第一屏混入复杂/进阶任务：%s", forbidden)
		}
	}
}

func TestEmbeddedRecordsPageHasActionableEmptyStates(t *testing.T) {
	js := extractEmbeddedScript(t)
	if !strings.Contains(indexHTML, `<h2 class="ph">我的单据 `) {
		t.Fatalf("我的单据页标题应短而明确，不要继续用长标题堆叠概念")
	}
	for _, needle := range []string{
		"function recordsEmptyHTML(",
		"我的单据只用来看已经成功的预约或排队号",
		"还没有单据",
		"已有预约或排队号",
		"先看今天排队",
		"去约未来",
		"获取通行证查看",
		`onclick="go(\'qt\')"`,
		`onclick="enterAdvanced(\'ca\')"`,
		`onclick="startAuth()"`,
		"record-empty-grid",
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("我的单据空状态缺少：%s", needle)
		}
	}
	lr := extractBetween(t, js, "async function lR()", "function localRecordsFooter")
	for _, needle := range []string{`recordsEmptyHTML('needs_auth')`, `recordsEmptyHTML('empty')`} {
		if !strings.Contains(lr, needle) {
			t.Errorf("lR 应通过统一空状态渲染单据页，缺少：%s", needle)
		}
	}
	if strings.Contains(lr, "当前没有预约或排队号。<div") ||
		strings.Contains(lr, "查看官方预约和排队号需要先获取一次通行证") {
		t.Fatalf("我的单据页不应继续用长段落空状态：\n%s", lr)
	}
}

func TestEmbeddedCalendarPageHasGuidedEmptyStates(t *testing.T) {
	js := extractEmbeddedScript(t)
	for _, needle := range []string{
		"function calendarEmptyHTML(",
		"约未来先看日历，再决定要不要预约或蹲点",
		"还没有通行证也能先看今天排队",
		"选择门店看日历",
		"只看可预约",
		"已满就蹲点",
		`onclick="openStorePicker({selected:selStores,onConfirm:applyCalendarStores})"`,
		`onclick="go(\'qt\')"`,
		`onclick="startAuth()"`,
		`onclick="go(\'sn\')"`,
		"calendar-empty-grid",
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("约未来空状态缺少新人分流内容：%s", needle)
		}
	}
	lc := extractBetween(t, js, "async function lC()", "function rStoreChoices")
	for _, needle := range []string{`calendarEmptyHTML('needs_auth')`, `calendarEmptyHTML('no_store')`} {
		if !strings.Contains(lc, needle) {
			t.Errorf("lC 应使用统一的约未来空状态，缺少：%s", needle)
		}
	}
	if strings.Contains(lc, "想查看未来可预约时段，需要先获取一次通行证") ||
		strings.Contains(lc, "还没选门店。选好后看看未来哪天有可约时段") {
		t.Fatalf("约未来页不应继续用长句空状态：\n%s", lc)
	}
}

func TestEmbeddedSniperPageStartsWithScenarioChooser(t *testing.T) {
	satisfied := satisfiedDOMIDs()
	for _, id := range []string{"snDecisionPanel", "snImmediateBox", "snScheduleBox"} {
		if !satisfied[id] {
			t.Errorf("自动抢预约页缺少场景分流锚点 id=%q", id)
		}
	}
	sn := extractBetween(t, indexHTML, `<section id="p-sn"`, `<section id="p-re"`)
	for _, needle := range []string{
		`id="snDecisionPanel"`,
		"先选你现在是哪种情况",
		"时段已经放出",
		"按偏好马上抢",
		"还没放出",
		"蹲开放瞬间",
		"不确定有没有放出",
		"先看可约日历",
		`onclick="scrollSnSection('snImmediateBox')"`,
		`onclick="scrollSnSection('snScheduleBox')"`,
		`onclick="go('ca')"`,
		"执行前会再次确认",
	} {
		if !strings.Contains(sn, needle) {
			t.Errorf("自动抢预约场景分流缺少：%s", needle)
		}
	}
	firstAction := strings.Index(sn, `id="snDecisionPanel"`)
	immediate := strings.Index(sn, `id="snImmediateBox"`)
	schedule := strings.Index(sn, `id="snScheduleBox"`)
	if firstAction < 0 || immediate < 0 || schedule < 0 || !(firstAction < immediate && immediate < schedule) {
		t.Fatalf("自动抢预约页层级应为场景分流 -> 已放出 -> 未放出：decision=%d immediate=%d schedule=%d", firstAction, immediate, schedule)
	}
	if !strings.Contains(extractEmbeddedScript(t), "function scrollSnSection(") {
		t.Fatalf("自动抢预约场景卡应通过 scrollSnSection 定位到对应操作区")
	}
	lSn := extractBetween(t, extractEmbeddedScript(t), "async function lSn()", "async function ensureStores")
	if strings.Contains(lSn, "setTimeout(expandSnPrefs") {
		t.Fatalf("自动抢预约页进入时不应自动滚走场景分流区：\n%s", lSn)
	}
}

func TestEmbeddedUIModeSwitchContracts(t *testing.T) {
	satisfied := satisfiedDOMIDs()
	for _, id := range []string{"uiModeSwitch", "uiModeSimple", "uiModeAdvanced", "uiModeSettings"} {
		if !satisfied[id] {
			t.Errorf("缺少界面模式锚点 id=%q", id)
		}
	}
	for _, needle := range []string{
		"function currentUIMode(",
		"function setUIMode(",
		"function applyUIMode(",
		"function enterAdvanced(",
		"function isAdvancedPage(",
		"function ensurePrefsLoaded(",
		"await ensurePrefsLoaded()",
		"简化版",
		"进阶版",
		"在进阶版中",
		"advanced-only",
		"simple-mode",
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少简化/进阶模式片段：%s", needle)
		}
	}
}

func TestEmbeddedSimpleModeAdvancedDeepLinksUseUpgradePrompt(t *testing.T) {
	js := extractEmbeddedScript(t)
	goFn := extractBetween(t, js, "function go(n,e,noPush)", "window.addEventListener('popstate'")
	for _, needle := range []string{
		"const target=n",
		"history.replaceState(null,'','#da')",
		"setTimeout(()=>enterAdvanced(target),80)",
		"n='da'",
	} {
		if !strings.Contains(goFn, needle) {
			t.Errorf("简化版进阶深链应先回首页再弹切换确认，缺少：%s", needle)
		}
	}
	if strings.Contains(goFn, "toast('该功能在进阶版中") {
		t.Fatalf("简化版打开进阶页不应只给短 toast；应弹出明确的切换确认")
	}
	enter := extractBetween(t, js, "async function enterAdvanced(target)", "function renderSubnav")
	if !strings.Contains(js, "function advancedPageName(page)") {
		t.Errorf("进阶确认弹窗应能显示目标页名称")
	}
	for _, needle := range []string{
		"切换到进阶版？",
		"advancedPageName(target||'')+'在进阶版中。进阶版会显示",
		"await setUIMode('advanced')",
		"if(target)go(target)",
	} {
		if !strings.Contains(enter, needle) {
			t.Errorf("进阶确认弹窗缺少必要说明或跳转逻辑：%s", needle)
		}
	}
}

func TestEmbeddedUIModeSwitchIsImmediate(t *testing.T) {
	block := regexp.MustCompile(`async function setUIMode\(mode\)\{[\s\S]*?\n\}`).FindString(indexHTML)
	if block == "" {
		t.Fatalf("找不到 setUIMode 函数")
	}
	for _, needle := range []string{
		"function cacheUIMode(",
		"function persistUIMode(",
		"cacheUIMode(mode);applyUIMode();",
		"persistUIMode(uiMode);",
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少即时切换片段：%s", needle)
		}
	}
	applyIdx := strings.Index(block, "applyUIMode()")
	loadIdx := strings.Index(block, "await ensurePrefsLoaded()")
	if applyIdx < 0 {
		t.Fatalf("setUIMode 缺少 applyUIMode")
	}
	if loadIdx >= 0 && loadIdx < applyIdx {
		t.Fatalf("setUIMode 不应先等待偏好接口再切 UI：\n%s", block)
	}
}

func TestEmbeddedUIModeSwitchIgnoresStalePreferenceResponses(t *testing.T) {
	for _, needle := range []string{
		"uiModeSeq=0",
		"uiModeSeq++;cacheUIMode(mode);applyUIMode();",
		"const modeSeq=uiModeSeq;await ensurePrefsLoaded();if(modeSeq===uiModeSeq)cacheUIMode(",
		"const modeSeq=uiModeSeq",
		"const serverMode=pr.ui_mode==='advanced'?'advanced':'simple'",
		"if(modeSeq===uiModeSeq||serverMode===currentUIMode())cacheUIMode(serverMode);else pr={...pr,ui_mode:currentUIMode()};",
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少防旧偏好回包覆盖模式片段：%s", needle)
		}
	}
}

func TestEmbeddedApplyUIModeRefreshesModeSensitiveHomeState(t *testing.T) {
	js := extractEmbeddedScript(t)
	block := extractBetween(t, js, "function applyUIMode()", "async function persistUIMode")
	for _, needle := range []string{"renderSettingsStatus();", "renderSetupCard();"} {
		if !strings.Contains(block, needle) {
			t.Fatalf("applyUIMode 应刷新模式相关首页/设置状态，缺少：%s\n%s", needle, block)
		}
	}
}

func TestEmbeddedPersistUIModeDoesNotRepaintPreferenceForm(t *testing.T) {
	block := regexp.MustCompile(`async function persistUIMode\(mode\)\{[\s\S]*?\n\}`).FindString(indexHTML)
	if block == "" {
		t.Fatalf("找不到 persistUIMode 函数")
	}
	for _, forbidden := range []string{"fF(pr)", "dP(pr)", "renderBookingStores()", "uD()"} {
		if strings.Contains(block, forbidden) {
			t.Fatalf("persistUIMode 不应重绘偏好表单，避免覆盖用户未保存输入；发现：%s\n%s", forbidden, block)
		}
	}
	if !strings.Contains(block, "applyUIMode();") {
		t.Fatalf("persistUIMode 仍需在保存成功后刷新模式状态")
	}
}

func TestEmbeddedSettingsAuthActionsAreLayered(t *testing.T) {
	satisfied := satisfiedDOMIDs()
	for _, id := range []string{"authPrimaryActions", "authTroubleshootFold", "authVerifyState", "certCheckState", "mobileAuthState"} {
		if !satisfied[id] {
			t.Errorf("缺少设置页认证分层锚点 id=%q", id)
		}
	}
	primary := regexp.MustCompile(`<div id="authPrimaryActions"[\s\S]*?</div>`).FindString(indexHTML)
	if primary == "" {
		t.Fatalf("找不到设置页认证日常动作区")
	}
	for _, needle := range []string{"openAuthWizard()", "resetAuthOnly(true)"} {
		if !strings.Contains(primary, needle) {
			t.Errorf("认证日常动作区缺少：%s", needle)
		}
	}
	for _, forbidden := range []string{"verifyAuthTicket()", "testAuthProbe()", "checkCert()", "验证通行证（取号测试）", "测试基础接口", "证书自检"} {
		if strings.Contains(primary, forbidden) {
			t.Fatalf("认证日常动作区不应直接露出排障检查：%s", forbidden)
		}
	}
	fold := regexp.MustCompile(`<details id="authTroubleshootFold"[\s\S]*?</details>`).FindString(indexHTML)
	if fold == "" {
		t.Fatalf("找不到认证更多检查折叠区")
	}
	for _, needle := range []string{"更多检查", "verifyAuthTicket()", "testAuthProbe()", "checkCert()", "验证通行证（取号测试）", "测试基础接口", "证书自检"} {
		if !strings.Contains(fold, needle) {
			t.Errorf("认证更多检查折叠区缺少：%s", needle)
		}
	}
	primaryIdx := strings.Index(indexHTML, `id="authPrimaryActions"`)
	foldIdx := strings.Index(indexHTML, `id="authTroubleshootFold"`)
	mobileIdx := strings.Index(indexHTML, `id="mobileAuthState"`)
	if primaryIdx < 0 || foldIdx < 0 || mobileIdx < 0 {
		t.Fatalf("认证分层索引异常：primary=%d fold=%d mobile=%d", primaryIdx, foldIdx, mobileIdx)
	}
	if !(primaryIdx < foldIdx && foldIdx < mobileIdx) {
		t.Fatalf("认证区应先显示日常动作，再放更多检查，最后显示状态：primary=%d fold=%d mobile=%d", primaryIdx, foldIdx, mobileIdx)
	}
}

func TestEmbeddedSettingsStartsWithQuickActionsAndCollapsedDetails(t *testing.T) {
	satisfied := satisfiedDOMIDs()
	for _, id := range []string{"settingsQuickActions", "fold-auth", "fold-notify"} {
		if !satisfied[id] {
			t.Errorf("缺少设置页聚焦入口锚点 id=%q", id)
		}
	}
	settings := extractBetween(t, indexHTML, `<section id="p-se"`, `</section>`)
	for _, needle := range []string{
		"先选你要做什么",
		"不用配置，先看排队",
		"获取通行证",
		"配置通知",
		`class="settings-quick-card read" onclick="go('qt')"`,
		`class="settings-quick-card auth" onclick="openAuthWizard()"`,
		`class="settings-quick-card read" onclick="focusNotifySettings()"`,
	} {
		if !strings.Contains(settings, needle) {
			t.Errorf("设置页快捷入口缺少片段：%s", needle)
		}
	}
	for _, forbidden := range []string{
		`id="fold-auth" open`,
		`id="fold-notify" open`,
		`class="cd setting-fold settings-wide" open`,
		`class="cd setting-fold" open`,
	} {
		if strings.Contains(settings, forbidden) {
			t.Fatalf("设置页细节不应默认展开，发现：%s", forbidden)
		}
	}
	quickIdx := strings.Index(settings, `id="settingsQuickActions"`)
	statusIdx := strings.Index(settings, `id="settingsStatus"`)
	modeIdx := strings.Index(settings, `id="uiModeSettings"`)
	firstSectionIdx := strings.Index(settings, `<div class="sect-divider"><span class="sect-no">1</span>`)
	if quickIdx < 0 || statusIdx < 0 || modeIdx < 0 || firstSectionIdx < 0 {
		t.Fatalf("设置页关键区块索引异常：quick=%d status=%d mode=%d first=%d", quickIdx, statusIdx, modeIdx, firstSectionIdx)
	}
	if !(quickIdx < statusIdx && quickIdx < modeIdx && quickIdx < firstSectionIdx) {
		t.Fatalf("设置页应先给用户任务入口，再显示状态和细节：quick=%d status=%d mode=%d first=%d", quickIdx, statusIdx, modeIdx, firstSectionIdx)
	}
	js := extractEmbeddedScript(t)
	focus := extractBetween(t, js, "function focusNotifySettings()", "async function lS()")
	for _, needle := range []string{
		"el('fold-notify')",
		"d.open=true",
		"el('nf')",
	} {
		if !strings.Contains(focus, needle) {
			t.Errorf("focusNotifySettings 应先展开通知设置再聚焦输入框，缺少：%s", needle)
		}
	}
}

func TestEmbeddedAuthEverydayCopyAvoidsTechnicalJargon(t *testing.T) {
	for _, needle := range []string{
		"<h2>通行证获取进度</h2>",
		"capturing:'正在获取通行证'",
		"正在获取通行证",
		"获取中",
		"获取到必要信息后下方进度会自动点亮",
		"<b>寿司郎通行证</b>",
		"获取通行证",
		"重置通行证",
		"重置寿司郎通行证？",
		"手机获取中",
		"通行证需更新",
		"验证通行证（真实取号测试）",
		"通行证过期了",
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("通行证日常流程缺少易懂文案：%s", needle)
		}
	}

	homeAuth := extractBetween(t, indexHTML, `<div id="cb" class="card hid mt16">`, `</div>
      </div>
      <aside class="side">`)
	capturing := extractBetween(t, extractEmbeddedScript(t), "if(es.status==='capturing')", "}else if(es.status==='booking'||es.status==='sniping')")
	settingsAuth := extractBetween(t, indexHTML, `<div class="sect-divider"><span class="sect-no">1</span>`, `<details id="authTroubleshootFold"`)
	for label, block := range map[string]string{
		"首页通行证进度": homeAuth,
		"首页运行态":   capturing,
		"设置通行证主区": settingsAuth,
	} {
		for _, forbidden := range []string{"捕获", "抓包", "凭证与认证", "认证凭证", "重置认证"} {
			if strings.Contains(block, forbidden) {
				t.Fatalf("%s 不应暴露技术词 %q：\n%s", label, forbidden, block)
			}
		}
	}

	for _, forbidden := range []string{
		"正在捕获通行证",
		"手机捕获中",
		"重置寿司郎认证",
		"重置认证失败",
		"凭证需更新",
		"验证凭证（真实取号测试）",
		"重置并重新认证",
		"凭证过期了",
		"拿通行证",
		"拿通行证（向导）",
		"拿一次通行证",
		"现在去拿？",
		"改用手机抓包",
		"重置抓包状态",
		"重新获取凭证",
		"抓不到包",
		"抓包链路就绪",
	} {
		if strings.Contains(indexHTML, forbidden) {
			t.Fatalf("通行证日常流程不应再出现旧术语：%s", forbidden)
		}
	}
}

func TestEmbeddedSimpleModeDoesNotNagMissingNotifications(t *testing.T) {
	js := extractEmbeddedScript(t)
	uAuth := extractBetween(t, js, "function uAuth()", "function healthStripHTML")
	if strings.Contains(uAuth, "else if(!nfc)") {
		t.Fatalf("顶部状态胶囊不应在简化版因为通知未配置变黄；通知是提醒/抢预约前置项，不是只读模式前置项")
	}
	if !strings.Contains(uAuth, "else if(currentUIMode()==='advanced'&&!nfc)") {
		t.Fatalf("顶部状态胶囊仍应在进阶版提示通知未配置")
	}

	health := extractBetween(t, js, "function openHealthPanel()", "function closeHealthPanel()")
	initialItems := extractBetween(t, health, "const items=[", " ];")
	if strings.Contains(initialItems, "t:'通知渠道'") {
		t.Fatalf("简化版运行前置条件不应默认列出通知渠道；它会让只读用户误以为必须配置")
	}
	if !strings.Contains(health, "if(currentUIMode()==='advanced')items.push({t:'通知渠道'") {
		t.Fatalf("通知渠道应只作为进阶版前置条件出现")
	}

	setup := extractBetween(t, js, "function renderSetupCard()", "function journeyStepHTML")
	if strings.Contains(extractBetween(t, setup, "items.push({t:'常用门店'", "const spOK="), "t:'通知渠道'") {
		t.Fatalf("首页准备清单在简化版不应把通知渠道列为待办")
	}
	if !strings.Contains(setup, "if(currentUIMode()==='advanced')items.push({t:'通知渠道'") {
		t.Fatalf("首页准备清单仍应在进阶版提醒通知渠道")
	}
	if !strings.Contains(js, "async function ensureNotifyConfigured(actionLabel)") ||
		!strings.Contains(js, "提醒已生成，但还没配通知渠道") {
		t.Fatalf("移除简化版全局提示时，必须保留具体操作前的通知配置提醒")
	}
}

func TestEmbeddedAdvancedOnlyMutationMarkers(t *testing.T) {
	for _, needle := range []string{
		`id="p-ca" class="hid advanced-page"`,
		`id="p-sn" class="hid advanced-page"`,
		`id="p-re" class="hid advanced-page"`,
		`id="qdSamplingFold" class="card adv mt16 advanced-only"`,
		`id="qtAutoTicketFold" class="adv mt16 advanced-only"`,
		`<details class="cd setting-fold settings-wide advanced-only" id="fold-sm"`,
		`<details class="cd setting-fold settings-wide advanced-only" id="fold-in"`,
		`<details class="cd setting-fold settings-wide advanced-only" id="fold-lo"`,
		`<details class="cd setting-fold settings-wide advanced-only" id="fold-safe"`,
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少进阶门控片段：%s", needle)
		}
	}

	if got := strings.Count(indexHTML, `if(currentUIMode()==='advanced')items.push({t:'预测数据'`); got < 2 {
		t.Fatalf("简化版的准备清单和运行前置条件不应直接露出预测数据采集入口，advanced-only gates=%d", got)
	}

	for _, needle := range []string{
		`buttons:[{l:'查看我的单据',f:"enterAdvanced('re')"},{l:'几点能吃上',f:"go('qd')"}]`,
		`b.onclick=()=>enterAdvanced('re')`,
		`buttons:[{l:'回首页',f:"go('da')"},{l:'查可约时段',f:"enterAdvanced('ca')"}]`,
		`currentUIMode()==='advanced'?'门店、叫号、在等桌数为公开实时信息；远程取号是会执行操作的实验性功能，确认后才会提交。':'门店、叫号、在等桌数为公开实时信息；简化版保持只读，不会替你取号。'`,
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("可见入口应通过进阶确认而不是直接跳转：%s", needle)
		}
	}
}

func TestEmbeddedQueueLiveAutoTicketPlanIsSecondary(t *testing.T) {
	satisfied := satisfiedDOMIDs()
	for _, id := range []string{"qtNextSteps", "qtAutoTicketFold", "ntStore", "ntStatus"} {
		if !satisfied[id] {
			t.Errorf("缺少现在去吃页自动取号锚点 id=%q", id)
		}
	}
	for _, needle := range []string{
		`id="qtAutoTicketFold" class="adv mt16 advanced-only"`,
		"自动取号计划",
		"会向寿司郎提交操作，需要时再展开。",
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少现在去吃页次要取号片段：%s", needle)
		}
	}
	if strings.Contains(indexHTML, `id="qtAutoTicketFold" class="adv mt16 advanced-only" open`) ||
		strings.Contains(indexHTML, `<details class="adv mt16 advanced-only" open>`) {
		t.Fatalf("现在去吃页不应默认展开自动取号计划；主路径应先看实时排队")
	}
	nextSteps := strings.Index(indexHTML, `id="qtNextSteps"`)
	autoPlan := strings.Index(indexHTML, `id="qtAutoTicketFold"`)
	if nextSteps < 0 || autoPlan < 0 {
		t.Fatalf("现在去吃页关键区块索引异常：nextSteps=%d autoPlan=%d", nextSteps, autoPlan)
	}
	if !(nextSteps < autoPlan) {
		t.Fatalf("自动取号计划应放在看排队后的次要区域：nextSteps=%d autoPlan=%d", nextSteps, autoPlan)
	}
}

func TestEmbeddedQueueLiveNextStepsSplitPredictionModes(t *testing.T) {
	satisfied := satisfiedDOMIDs()
	for _, id := range []string{"qtPickupForecastEntry", "qtTicketForecastEntry"} {
		if !satisfied[id] {
			t.Errorf("现在去吃页缺少预测分流入口 id=%q", id)
		}
	}
	nextSteps := regexp.MustCompile(`<div class="card mt16" id="qtNextSteps"[\s\S]*?</div>\s*</div>\s*<details id="qtAutoTicketFold"`).FindString(indexHTML)
	if nextSteps == "" {
		t.Fatalf("找不到现在去吃页下一步区域")
	}
	for _, needle := range []string{
		"现在取号，几点能吃上",
		"已有号码，算几点能吃上",
		"openPickupForecastFromQueue()",
		"openTicketForecastFromQueue()",
	} {
		if !strings.Contains(nextSteps, needle) {
			t.Errorf("现在去吃页下一步区域缺少预测分流片段：%s", needle)
		}
	}
	if strings.Contains(nextSteps, "算几点叫到我（已拿号）") {
		t.Fatalf("现在去吃页不应只用“已拿号”入口，未拿号用户也应能直接去算现在取号")
	}
	js := extractEmbeddedScript(t)
	for _, needle := range []string{
		"function openPickupForecastFromQueue()",
		"setPlanDir('pickup')",
		"function openTicketForecastFromQueue()",
		"el('qdTargetNo')",
		"qtSelected[0]",
	} {
		if !strings.Contains(js, needle) {
			t.Errorf("缺少现在去吃页预测跳转辅助逻辑：%s", needle)
		}
	}
}

func TestEmbeddedQueueLiveRequiresFocusedStoreSelection(t *testing.T) {
	queuePage := extractBetween(t, indexHTML, `<section id="p-qt"`, `<section id="p-sn"`)
	if strings.Contains(queuePage, `<div id="qtStores" class="chips mt8"><span class="mu">尚未选择门店</span></div>`) {
		t.Fatalf("现在去吃页初始占位不应直接写“尚未选择门店”；门店配置还在加载时应先用确认中状态")
	}
	if !strings.Contains(queuePage, `<div id="qtStores" class="chips mt8"><span class="mu">正在确认关注门店</span></div>`) {
		t.Fatalf("现在去吃页门店初始占位应使用“正在确认关注门店”")
	}
	for _, needle := range []string{
		"function queueStarterHTML(",
		"先选一家常去门店",
		"选门店看排队",
		"我有号码",
		"先看机制图",
		`onclick="openStorePicker({selected:qtSelected,onConfirm:applyQueueStores})"`,
		`onclick="go(\'qd\')"`,
		`onclick="go(\'gu\')"`,
		"queue-starter-grid",
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("现在去吃页无门店入口缺少片段：%s", needle)
		}
	}
	js := extractEmbeddedScript(t)
	initFilters := extractBetween(t, js, "function initQueueTrendFilters", "function renderQueueTrendStores")
	if strings.Contains(initFilters, "stores.map") {
		t.Fatalf("现在去吃页不应在未选择门店时自动加载所有已配置门店")
	}
	if !strings.Contains(initFilters, "else if(stores.length)qtSelected=[String(stores[0].id)]") {
		t.Fatalf("现在去吃页应在已有配置门店时自动带入第一家，减少重复选择：\n%s", initFilters)
	}
	loadLive := extractBetween(t, js, "async function loadQueueLive", "let qtPanels")
	if !strings.Contains(loadLive, "queueStarterHTML()") {
		t.Fatalf("现在去吃页无关注门店时应展示分流入口，而不是默认门店列表")
	}
	for _, needle := range []string{
		"p.set('limit','8')",
		"/api/queue/stores?'+p.toString()",
	} {
		if strings.Contains(loadLive, needle) {
			t.Errorf("现在去吃页无关注门店时不应继续拉取全国门店列表：%s", needle)
		}
	}
}

func TestEmbeddedQueueLiveCardsUseFriendlyStatusText(t *testing.T) {
	js := extractEmbeddedScript(t)
	for _, needle := range []string{
		"function queueLiveOpen(",
		"function queueLiveStatusLabel(",
		"function queueLiveEtaText(",
		"function queueLiveNowMealText(",
		"暂停营业",
		"线上取号暂停",
		"线上可取号",
	} {
		if !strings.Contains(js, needle) {
			t.Errorf("实时排队卡片缺少状态中文化逻辑：%s", needle)
		}
	}
	panels := extractBetween(t, js, "function renderQueueLivePanels", "function renderQueueLive(rows)")
	for _, needle := range []string{
		"queueLiveOpen(s)",
		"queueLiveStatusLabel(s)",
		"queueLiveEtaText(s,open)",
		"queueLiveNowMealText(s,open)",
		"现在取号",
	} {
		if !strings.Contains(panels, needle) {
			t.Errorf("关注门店实时卡片未使用友好状态逻辑：%s", needle)
		}
	}
	if strings.Contains(panels, `[s.store_status||'-',s.net_ticket_status||'-'].join(' · ')`) {
		t.Fatalf("关注门店实时卡片不应直接显示 CLOSED/OFFLINE_CLOSED 等原始状态码")
	}
	list := extractBetween(t, js, "function renderQueueLive(rows)", "function queueStatusText")
	for _, needle := range []string{
		"queueLiveOpen(s)",
		"queueLiveStatusLabel(s)",
		"queueLiveEtaText(s,open)",
		"queueLiveNowMealText(s,open)",
		"现在取号",
	} {
		if !strings.Contains(list, needle) {
			t.Errorf("默认实时门店列表未使用友好状态逻辑：%s", needle)
		}
	}
	if strings.Contains(list, `esc(status)+' · '+esc(ticket)`) {
		t.Fatalf("默认实时门店列表不应直接显示 OPEN/CLOSED/OFFLINE_CLOSED 等原始状态码")
	}
}

func TestEmbeddedPlanConverterIsFirstClass(t *testing.T) {
	// 时间换算（几点取号 ⇄ 几点吃）是产品核心价值，必须对所有模式可见（非 advanced-only），
	// 且双向用 ⇄ 换向、输入即算（debounce），不再藏在折叠/进阶门后。
	for _, needle := range []string{
		`id="qdPlanFold" class="plan-card"`,
		`onclick="swapPlanDir()"`,
		`oninput="runPlanCalcDebounced()"`,
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少时间换算一等公民片段：%s", needle)
		}
	}
	if strings.Contains(indexHTML, `id="qdPlanFold" class="card adv mt16 advanced-only"`) {
		t.Fatalf("时间换算不应再是 advanced-only 折叠，应对所有模式可见")
	}
}

func TestEmbeddedQueuePredictionPickupModeStartsWithNowTicket(t *testing.T) {
	js := extractEmbeddedScript(t)
	modeFn := extractBetween(t, js, "function setQueuePredictionMode(", "function setPlanDir(")
	for _, needle := range []string{
		"if(mode==='pickup')",
		"setPlanDir('pickup')",
		"applyPlanDir()",
	} {
		if !strings.Contains(modeFn, needle) {
			t.Errorf("切到“还没拿号”时应回到现在取号口径，缺少：%s\n%s", needle, modeFn)
		}
	}
}

func TestEmbeddedQueuePredictionSplitsNowTicketAndExistingTicket(t *testing.T) {
	satisfied := satisfiedDOMIDs()
	for _, id := range []string{"qdModeTabs", "qdModeTicket", "qdModePickup", "qdPredictionModes", "qdNowTicketCard", "qdExistingTicketCard", "qdTargetNo", "qpPickup"} {
		if !satisfied[id] {
			t.Errorf("缺少排队预测分流锚点 id=%q", id)
		}
	}
	for _, needle := range []string{
		"现在取号，几点能吃上",
		"还没拿号时用这个",
		"用现在估算",
		"我已经取到号，这个号几点能吃上",
		"拿到当天排队号后用这个",
		"function setQueuePredictionMode(",
		"function useNowForPickupPlan(",
		`onclick="setQueuePredictionMode('ticket')"`,
		`onclick="setQueuePredictionMode('pickup')"`,
		`onclick="useNowForPickupPlan()"`,
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少排队预测分流片段：%s", needle)
		}
	}
	if strings.Contains(indexHTML, "输入当天排队号，判断几点到店") {
		t.Fatalf("已拿号页标题不应继续把「现在取号」和「已有号码」混成一个预测入口")
	}
	if strings.Contains(indexHTML, `id="qpPickup" type="time" value="12:10"`) {
		t.Fatalf("现在取号预测不应内置固定 12:10，默认应按用户当前时间估算")
	}
	nowCard := strings.Index(indexHTML, `id="qdNowTicketCard"`)
	existingCard := strings.Index(indexHTML, `id="qdExistingTicketCard"`)
	evidence := strings.Index(indexHTML, `id="qdEvidence"`)
	if nowCard < 0 || existingCard < 0 || evidence < 0 {
		t.Fatalf("排队预测分流索引异常：now=%d existing=%d evidence=%d", nowCard, existingCard, evidence)
	}
	if !(nowCard < existingCard && existingCard < evidence) {
		t.Fatalf("已拿号页应先展示现在取号预测，再展示已有号码预测，最后才是图表：now=%d existing=%d evidence=%d", nowCard, existingCard, evidence)
	}
}

func TestEmbeddedQueuePredictionShowsOneScenarioAtATime(t *testing.T) {
	nowCard := regexp.MustCompile(`<div id="qdNowTicketCard" class="prediction-panel now hid">`).FindString(indexHTML)
	if nowCard == "" {
		t.Fatalf("我有号码页默认应先聚焦已有号码，把“现在取号”面板收起")
	}
	if strings.Contains(indexHTML, `<div id="qdExistingTicketCard" class="prediction-panel existing hid">`) {
		t.Fatalf("已有号码面板不应默认隐藏")
	}
	js := extractEmbeddedScript(t)
	modeFn := extractBetween(t, js, "function setQueuePredictionMode(", "function setPlanDir(")
	for _, needle := range []string{
		"qdNowTicketCard",
		"qdExistingTicketCard",
		"classList.toggle('hid',mode!=='pickup')",
		"classList.toggle('hid',mode!=='ticket')",
		"qdModeTicket",
		"qdModePickup",
	} {
		if !strings.Contains(modeFn, needle) {
			t.Errorf("预测模式切换缺少：%s", needle)
		}
	}
	qtHelpers := extractBetween(t, js, "function openPickupForecastFromQueue()", "function explainMsg(")
	if !strings.Contains(qtHelpers, "setQueuePredictionMode('pickup')") ||
		!strings.Contains(qtHelpers, "setQueuePredictionMode('ticket')") {
		t.Fatalf("从现在去吃页跳转预测时，应切换到对应模式：\n%s", qtHelpers)
	}
	pickupHelper := extractBetween(t, js, "function openPickupForecastFromQueue()", "function openTicketForecastFromQueue()")
	if !strings.Contains(pickupHelper, "el('qdTargetNo')") || !strings.Contains(pickupHelper, ".value=''") {
		t.Fatalf("从现在去吃页进入“现在取号”预测时，应清掉旧排队号，避免套用已有号码口径：\n%s", pickupHelper)
	}
	lQD := extractBetween(t, js, "async function lQD()", "function dashboardParams()")
	if !strings.Contains(lQD, "setQueuePredictionMode('ticket')") {
		t.Fatalf("直接进入我有号码页时，应默认回到已有号码模式：\n%s", lQD)
	}
	for _, needle := range []string{
		"const saved=recallStores('sushiro_qd_store').slice(0,1)",
		"else if(stores.length)qdSelected=[String(stores[0].id)]",
	} {
		if !strings.Contains(lQD, needle) {
			t.Errorf("我有号码页应自动带入已配置门店，减少重复选择，缺少：%s", needle)
		}
	}
}

func TestEmbeddedQueuePredictionHidesHistoricalAdviceUntilReady(t *testing.T) {
	satisfied := satisfiedDOMIDs()
	if !satisfied["qdAdvisorBlock"] {
		t.Fatalf("我有号码页应把历史规律建议区包成 qdAdvisorBlock，便于输入前隐藏")
	}
	existing := extractBetween(t, indexHTML, `<div id="qdExistingTicketCard"`, `<details id="qdAnalysisFold"`)
	for _, needle := range []string{
		`id="qdAdvisorBlock" class="hid"`,
		"历史规律 · 到店建议",
		`id="qdAdvisor"`,
	} {
		if !strings.Contains(existing, needle) {
			t.Errorf("已有号码面板应把次要历史建议默认收起，缺少：%s", needle)
		}
	}
	js := extractEmbeddedScript(t)
	for _, needle := range []string{
		"function updateQueuePredictionReadiness(",
		"const ready=!!qdSelected.length&&target>0",
		"el('qdAdvisorBlock')?.classList.toggle('hid',!ready)",
	} {
		if !strings.Contains(js, needle) {
			t.Errorf("缺少我有号码页输入就绪显隐逻辑：%s", needle)
		}
	}
	for label, block := range map[string]string{
		"输入号码": extractBetween(t, js, "function qdInputDebounced()", "function refreshCloudDependentViews()"),
		"选择门店": extractBetween(t, js, "function applyDashboardStores(", "function renderDashboardStores()"),
		"加载答案": extractBetween(t, js, "async function loadQueueAdvisorCard()", "function renderQueueAnswer("),
	} {
		if !strings.Contains(block, "updateQueuePredictionReadiness()") {
			t.Errorf("%s 后应刷新历史建议显隐状态", label)
		}
	}
}

func TestEmbeddedQueuePredictionStorePlaceholderIsNotMisleading(t *testing.T) {
	qd := extractBetween(t, indexHTML, `<section id="p-qd"`, `<section id="p-qt"`)
	if strings.Contains(qd, "默认自动选择本机样本最多的门店") {
		t.Fatalf("我有号码页加载门店前不应承诺“默认自动选择”，避免门店接口未回时误导用户")
	}
	if !strings.Contains(qd, "正在确认常用门店") {
		t.Fatalf("我有号码页门店占位应使用加载态文案")
	}
}

func TestEmbeddedQueuePredictionUsesMealTimeLanguageOnPrimaryPaths(t *testing.T) {
	guide := extractBetween(t, indexHTML, `<section id="p-gu"`, `<section id="p-ca"`)
	queueStarter := extractBetween(t, extractEmbeddedScript(t), "function queueStarterHTML()", "async function lQT()")
	queueAnswer := extractBetween(t, indexHTML, `<div id="qdExistingTicketCard"`, `<details id="qdAnalysisFold"`)
	queueJS := extractBetween(t, extractEmbeddedScript(t), "async function loadQueueAdvisorCard()", "function answerChip")

	for label, block := range map[string]string{
		"新手页":     guide,
		"现在去吃空状态": queueStarter,
		"我有号码答案区": queueAnswer + queueJS,
	} {
		if !strings.Contains(block, "几点能吃上") {
			t.Errorf("%s 主路径应使用“几点能吃上”的用户语言", label)
		}
		for _, forbidden := range []string{"估几点叫到", "算几点叫到", "大概几点叫到", "给你「几点叫到"} {
			if strings.Contains(block, forbidden) {
				t.Fatalf("%s 主路径不应保留旧的“叫到”预测口径：%s", label, forbidden)
			}
		}
	}
}

func TestEmbeddedQueuePredictionKeepsAnalysisFolded(t *testing.T) {
	satisfied := satisfiedDOMIDs()
	for _, id := range []string{"qdAnalysisFold", "qdEvidence", "qdInsights", "qdMoreTools"} {
		if !satisfied[id] {
			t.Errorf("缺少排队预测分析折叠锚点 id=%q", id)
		}
	}
	fold := regexp.MustCompile(`<details id="qdAnalysisFold"[\s\S]*?</details>\s*<details id="qdMoreTools"`).FindString(indexHTML)
	if fold == "" {
		t.Fatalf("排队预测页应把走势大图和历史规律收进 qdAnalysisFold，并放在提醒工具之前")
	}
	for _, needle := range []string{`<summary>`, "为什么这样判断", `id="qdEvidence"`, `id="qdInsights"`} {
		if !strings.Contains(fold, needle) {
			t.Errorf("qdAnalysisFold 缺少必要内容：%s", needle)
		}
	}
	if strings.Contains(fold, `<details id="qdAnalysisFold" class="card adv mt16" open`) ||
		strings.Contains(fold, `<details id="qdAnalysisFold" class="card adv mt16 advanced-only"`) {
		t.Fatalf("分析证据应默认收起，且不应只对进阶模式可见")
	}
	prediction := strings.Index(indexHTML, `id="qdPredictionModes"`)
	analysis := strings.Index(indexHTML, `id="qdAnalysisFold"`)
	tools := strings.Index(indexHTML, `id="qdMoreTools"`)
	if prediction < 0 || analysis < 0 || tools < 0 || !(prediction < analysis && analysis < tools) {
		t.Fatalf("预测页层级应为答案区 -> 分析依据 -> 提醒工具：prediction=%d analysis=%d tools=%d", prediction, analysis, tools)
	}
}

func TestEmbeddedUXM1PrimaryActions(t *testing.T) {
	for _, needle := range []string{
		`id="qdPrimaryActions"`,
		`class="bt bt-r bt-s" onclick="openStorePicker({selected:qdSelected.slice(0,1),multi:false,onConfirm:applyDashboardStores})">选门店`,
		`class="bt bt-w bt-s" onclick="loadQueueDashboard()">刷新`,
		`id="sc"><div class="empty"><div class="mascot-wrap">`,
		`onclick="openStorePicker({selected:selStores,onConfirm:applyCalendarStores})">选择门店`,
		`function recordsEmptyHTML(`,
		`<button class="record-empty-card read" onclick="go(\'qt\')" type="button">`,
		`<button class="record-empty-card auth" onclick="enterAdvanced(\'ca\')" type="button">`,
		`<button class="record-empty-card auth" onclick="startAuth()" type="button">`,
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少 M1 主路径片段：%s", needle)
		}
	}

	autoPlan := regexp.MustCompile(`<details[^>]*id="qtAutoTicketFold"[\s\S]*?onclick="saveNetTicketPlan\(true\)">启用`)
	m := autoPlan.FindString(indexHTML)
	if m == "" {
		t.Fatalf("找不到自动取号计划启用按钮片段")
	}
	if strings.Contains(m, `bt bt-r bt-s`) {
		t.Fatalf("自动取号计划启用按钮不应使用红色主按钮，避免把会执行动作做得过强")
	}
	if !strings.Contains(m, `bt bt-o bt-s`) {
		t.Fatalf("自动取号计划启用按钮应使用次要描边按钮")
	}

	if !strings.Contains(indexHTML, `class="bt bt-o bt-s advanced-only" onclick="takeTicket`) {
		t.Fatalf("远程取号按钮应使用次要描边按钮且仅进阶版展示，让页面保持只读优先")
	}
}

func TestEmbeddedQueueChartsAreFirstClassSections(t *testing.T) {
	for _, needle := range []string{
		`id="qdEvidence"`,
		`id="qdPressChart"`,
		`id="qdInsights"`,
		"整合走势大图",
		"这家店的历史规律",
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Fatalf("indexHTML 缺少排队图表片段：%s", needle)
		}
	}
	for _, folded := range []string{
		`<details class="card adv mt16" id="qdEvidence"`,
		`<details class="card adv mt16" id="qdInsights"`,
	} {
		if strings.Contains(indexHTML, folded) {
			t.Fatalf("排队图表不应藏在折叠区：%s", folded)
		}
	}

	advisor := strings.Index(indexHTML, `id="qdAdvisor"`)
	evidence := strings.Index(indexHTML, `id="qdEvidence"`)
	chart := strings.Index(indexHTML, `id="qdPressChart"`)
	insights := strings.Index(indexHTML, `id="qdInsights"`)
	reminder := strings.Index(indexHTML, `id="qdReminderCard"`)
	if advisor < 0 || evidence < 0 || chart < 0 || insights < 0 || reminder < 0 {
		t.Fatalf("排队页关键区块索引异常：advisor=%d evidence=%d chart=%d insights=%d reminder=%d", advisor, evidence, chart, insights, reminder)
	}
	if !(advisor < evidence && evidence < chart && chart < insights && insights < reminder) {
		t.Fatalf("排队页图表应在建议后、提醒前展示：advisor=%d evidence=%d chart=%d insights=%d reminder=%d", advisor, evidence, chart, insights, reminder)
	}
}

func TestEmbeddedQDSecondaryToolsAreFolded(t *testing.T) {
	satisfied := satisfiedDOMIDs()
	for _, id := range []string{"qdMoreTools", "qdReminderCard", "qdSamplingFold"} {
		if !satisfied[id] {
			t.Errorf("缺少已拿号页次要工具锚点 id=%q", id)
		}
	}
	for _, needle := range []string{
		`<details id="qdMoreTools" class="card adv mt16">`,
		"提醒和进阶工具",
		"想被叫号前提醒、每日取号提醒或提升曲线准确度时再展开。",
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少已拿号页折叠工具片段：%s", needle)
		}
	}
	if strings.Contains(indexHTML, "提醒 · 时间换算 · 采集配置") {
		t.Fatalf("已拿号页不应再把提醒、换算、采集并列成主标题；时间换算已是一等入口，提醒/采集应按需展开")
	}

	insights := strings.Index(indexHTML, `id="qdInsights"`)
	moreTools := strings.Index(indexHTML, `id="qdMoreTools"`)
	reminder := strings.Index(indexHTML, `id="qdReminderCard"`)
	sampling := strings.Index(indexHTML, `id="qdSamplingFold"`)
	if insights < 0 || moreTools < 0 || reminder < 0 || sampling < 0 {
		t.Fatalf("已拿号页次要工具索引异常：insights=%d moreTools=%d reminder=%d sampling=%d", insights, moreTools, reminder, sampling)
	}
	if !(insights < moreTools && moreTools < reminder && reminder < sampling) {
		t.Fatalf("已拿号页次要工具应在历史规律后折叠，并保持提醒在采集前：insights=%d moreTools=%d reminder=%d sampling=%d", insights, moreTools, reminder, sampling)
	}
}

func TestEmbeddedDailyReminderAvoidsDeveloperJargon(t *testing.T) {
	js := extractEmbeddedScript(t)
	for _, forbidden := range []string{
		"未开启 Routine",
		"Routine 只是提醒",
		"Routine 明天",
		"Routine 保存失败",
		"启用 Routine 前",
		"启用每日取号提醒 Routine",
		"关闭每日取号提醒 Routine",
		"已开启取号提醒 Routine",
		"已关闭 Routine",
		"保存 Routine 失败",
	} {
		if strings.Contains(js, forbidden) {
			t.Fatalf("每日取号提醒不应在用户可见文案里出现开发者术语：%s", forbidden)
		}
	}
	for _, needle := range []string{
		"每日取号提醒",
		"只是提醒你手动取号",
		"不会自动向寿司郎提交取号请求",
		"明天会重新规划提醒时间",
		"已开启每日取号提醒",
		"已关闭每日取号提醒",
	} {
		if !strings.Contains(js, needle) {
			t.Errorf("每日取号提醒缺少用户可理解文案：%s", needle)
		}
	}
}

func TestEmbeddedDailyReminderHidesOnceOnlyCreateButton(t *testing.T) {
	if !strings.Contains(indexHTML, `id="qdrCreateBtn"`) {
		t.Fatalf("当次叫号提醒的“生成提醒”按钮需要独立 id，便于每日提醒模式隐藏")
	}
	js := extractEmbeddedScript(t)
	remTab := extractBetween(t, js, "function remTab(t)", "function expandSnPrefs")
	for _, needle := range []string{
		"el('qdrCreateBtn')",
		"classList.toggle('hid',!once)",
	} {
		if !strings.Contains(remTab, needle) {
			t.Errorf("remTab 应在每日提醒模式隐藏当次“生成提醒”按钮，缺少：%s", needle)
		}
	}
}

func TestEmbeddedCloudAuthVerifiesBaselineAfterLogin(t *testing.T) {
	for _, needle := range []string{
		"cloudVerifyOnLoad",
		"const connected=p.get('cloud_connected')",
		"cloudVerifyOnLoad=true",
		"toast('云端 GitHub 登录已完成')",
		"const verifyCloud=cloudVerifyOnLoad;cloudVerifyOnLoad=false;await loadCloudAuth(verifyCloud)",
		"catch(e){await loadCloudAuth(true);toast('云端连接失败：'",
		"chip('线上数据库'",
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少云端基准验证片段：%s", needle)
		}
	}
}

func TestEmbeddedDashboardExplainsCloudBaselineUse(t *testing.T) {
	for _, needle := range []string{
		`id="qdDataSource"`,
		`class="data-source mt16"`,
		".data-source{display:grid",
		"function dashboardBaselineStatusHTML(",
		"const b=(d&&d.baseline)||{}",
		"used=!!b.used",
		"图表数据来源",
		"线上数据库基准",
		"rollup_count",
		"latest_count",
		"d.warnings",
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少图表云端基准可见化片段：%s", needle)
		}
	}
}

func TestEmbeddedDashboardFusesTursoTrendIntoMainChart(t *testing.T) {
	for _, needle := range []string{
		"function historicalQueueTrendPoints(",
		"(d&&d.trend)||[]",
		"legend-turso-trend",
		"trendMax=Math.max(1,...trend.map",
		"trendPts.length>1",
		"历史排队趋势：绿色虚线是",
		"线上数据库基准",
		"total_queue_groups",
		"sample_count",
		"if(!points.length&&!hist.length&&!trend.length)",
		"renderPressureChart(pc,{points:[],message:'选门店后",
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少主图融合历史趋势片段：%s", needle)
		}
	}

	// 反向断言：旧文案与错误字段名必须已清除，防止回滚。
	for _, stale := range []string{
		"归一化到右侧压力轴",
		"未选门店时为全国，选门店后为本机",
		"qdDashboardData.scope.scope==='all'",
		"独立归一化", // 图例文案已精简，不再出现
		"开店数",   // 用户不需要开店数，已从 tooltip/趋势条移除
	} {
		if strings.Contains(indexHTML, stale) {
			t.Errorf("indexHTML 仍含应已删除的旧片段：%s", stale)
		}
	}

	// 图例里的趋势项必须按数据条件渲染：图例块内 legend-turso-trend 出现且其前缀
	// 必须紧跟 trendPts.length>1?，避免有人改回无条件渲染而测试漏过。
	legendBlock := regexp.MustCompile(`<div class="chart-legend">[\s\S]*?</div>`).FindString(indexHTML)
	if legendBlock == "" {
		t.Fatalf("找不到 chart-legend 块")
	}
	if strings.Count(legendBlock, "legend-turso-trend") != 1 {
		t.Fatalf("chart-legend 块应恰好包含 1 个 legend-turso-trend，实际 %d", strings.Count(legendBlock, "legend-turso-trend"))
	}
	if !strings.Contains(legendBlock, `trendPts.length>1?'<span class="legend-turso-trend"`) {
		t.Fatalf("趋势图例项必须由 trendPts.length>1? 条件包裹，避免无数据时误导")
	}

	noStore := regexp.MustCompile(`if\(!store\)\{[\s\S]*?return\}`).FindString(indexHTML)
	if noStore == "" {
		t.Fatalf("找不到 loadQueueAdvisorCard 的未选门店分支")
	}
	if strings.Contains(noStore, "qdDashboardData={}") {
		t.Fatalf("未选门店时不应清空 qdDashboardData，否则会把已加载的全局历史趋势覆盖成空态")
	}
}

func TestEmbeddedDashboardMainChartReadableOnMobile(t *testing.T) {
	// 移动端主图：容器可横向滚动兜底，但 svg 不再固定 680px 宽（会顶破窄屏），
	// 而是允许收缩（min-width:0）+ 用 viewBox/preserveAspectRatio 自适应缩放。
	if !strings.Contains(indexHTML, "#qdPressChart{overflow:auto}") {
		t.Errorf("indexHTML 缺少移动端主图容器滚动兜底：#qdPressChart{overflow:auto}")
	}
	if !strings.Contains(indexHTML, "#qdPressChart svg{min-width:0;height:auto}") {
		t.Errorf("indexHTML 缺少移动端主图自适应样式：#qdPressChart svg{min-width:0;height:auto}")
	}
	// 旧的固定 680px 宽规则会在窄屏横向溢出，必须移除。
	if strings.Contains(indexHTML, "#qdPressChart svg{min-width:680px;height:260px}") {
		t.Fatalf("indexHTML 仍包含会顶破窄屏的 #qdPressChart svg{min-width:680px}")
	}
}

func TestEmbeddedSettingsDoesNotOverstateCloudBaseline(t *testing.T) {
	for _, needle := range []string{
		"const cloudBaseOK=!!cloudAuth.baseline_connected",
		"GitHub 已登录，线上数据库已验证",
		"GitHub 已登录，线上数据库待验证",
		"验证前图表会继续优先用本机数据",
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少设置页云端基准状态片段：%s", needle)
		}
	}
	if strings.Contains(indexHTML, "全国排队基准已接入") {
		t.Fatalf("设置页不应在只登录 GitHub 时宣称全国排队基准已接入")
	}
	// 不应再向用户暴露 Turso 字样。
	if strings.Contains(indexHTML, "Turso") {
		t.Errorf("indexHTML 不应再向用户暴露 Turso 字样")
	}
}

func TestEmbeddedDashboardDataSourceDoesNotTreatConfiguredCloudAsLoggedIn(t *testing.T) {
	for _, needle := range []string{
		"const configured=!!b.configured,authenticated=!!b.authenticated",
		"else if(authenticated)",
		"else if(configured)",
		"云端服务已配置；登录 GitHub 后可验证线上基准并叠加参考。",
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少图表数据源云端登录状态区分片段：%s", needle)
		}
	}
	if strings.Contains(indexHTML, "cfg=!!b.configured||!!b.authenticated") {
		t.Fatalf("图表数据源不应把云端已配置等同于 GitHub 已登录")
	}
}

func TestEmbeddedCloudLoginRefreshesQueueCharts(t *testing.T) {
	for _, needle := range []string{
		"cloudRefreshPending",
		"cloudRefreshPending=true",
		"if(cloudRefreshPending&&(n==='qd'||n==='qt'))",
		"setTimeout(refreshCloudDependentViews,120)",
		"function refreshCloudDependentViews(",
		"refreshCloudDependentViews()",
		"if(cp==='qd')",
		"loadQueueDashboard()",
		"if(cp==='qt')",
		"refreshQueueView()",
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少 GitHub 登录后刷新图表片段：%s", needle)
		}
	}
}

func TestEmbeddedDashboardCloudChartsDoNotRequireSushiroAuth(t *testing.T) {
	// cloud（GitHub 线上基准）图表入口与寿司郎通行证解耦：进阶版下 cloud 不需寿司郎认证即可叠加。
	// 简化版把 cloud 入口收敛隐藏（见 TestSimpleModeHidesQDCloudEntry），但解耦逻辑保留。
	for _, needle := range []string{
		"await loadCloudAuth(false);await loadSampling();",
		"const cloudReady=!!(cloudAuth.baseline_connected||(qdDashboardData.baseline&&qdDashboardData.baseline.used))",
		"const cloudLoggedIn=!!cloudAuth.connected",
		"const localNeedsAuth=!hc||q.needs_auth||q.auth_ok===false",
		"const cloudButton=cloudReady||cloudLoggedIn?'<button class=\"bt bt-w bt-s\" onclick=\"loadQueueDashboard()\">刷新图表</button>':'<button class=\"bt bt-w bt-s\" onclick=\"startCloudLogin()\">登录 GitHub 获取线上基准</button>'",
		// 进阶版下 cloud 入口与小程序采集解耦的文案保留
		"图表走 GitHub + 线上数据库；小程序通行证只用于本机采集补强。",
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少 GitHub 图表与小程序采集解耦片段：%s", needle)
		}
	}
	// 简化版收敛：cloud 入口被 adv 门控
	if !strings.Contains(indexHTML, "const adv=currentUIMode()==='advanced';") {
		t.Error("indexHTML 缺少采样卡的进阶模式门控")
	}
	if !strings.Contains(indexHTML, "const actions=adv&&localNeedsAuth?cloudButton") {
		t.Error("indexHTML 采样卡 actions 未按进阶模式收敛 cloud 入口")
	}
}

func TestEmbeddedQueueDashboardRefreshesSamplingCardAfterBaselineLoad(t *testing.T) {
	for _, needle := range []string{
		"qdDashboardData=d||{};renderQueueDashboard(d);renderDashboardSamplingCard()",
		"qdDashboardData={};adv.innerHTML=loadErrBoxHTML(e,'loadQueueDashboard()','到店建议');renderDashboardSamplingCard()",
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少图表基准加载后刷新采集卡片段：%s", needle)
		}
	}
}

func TestEmbeddedQueueDashboardInvalidatesAdvisorOnReloadStart(t *testing.T) {
	block := regexp.MustCompile(`async function loadQueueDashboard\(\)\{[\s\S]*?\n?function loadQueueAdvisorCard`).FindString(indexHTML)
	if block == "" {
		t.Fatalf("找不到 loadQueueDashboard 函数")
	}
	for _, needle := range []string{
		"const token=++qdDashToken;qdRefreshToken++;",
		"if(token!==qdDashToken)return",
		"if(token===qdDashToken)loadQueueAdvisorCard()",
	} {
		if !strings.Contains(block, needle) {
			t.Errorf("loadQueueDashboard 缺少防旧请求覆盖片段：%s\n%s", needle, block)
		}
	}
	if strings.Contains(block, "const token=++qdDashToken;try") {
		t.Fatalf("loadQueueDashboard 应在发起 dashboard 请求时同步递增 qdRefreshToken，让旧 advisor/curve 回包失效")
	}
}

func TestEmbeddedQueueDashboardKeepsBaselineWhenTargetHasNoStore(t *testing.T) {
	block := regexp.MustCompile(`async function loadQueueDashboard\(\)\{[\s\S]*?\n?function loadQueueAdvisorCard`).FindString(indexHTML)
	if block == "" {
		t.Fatalf("找不到 loadQueueDashboard 函数")
	}
	for _, stale := range []string{
		"if(target>0&&!qdSelected.length){qdDashboardData={}",
		"选择门店后才能判断你的号码，避免用其他门店曲线误判。</div>';renderDashboardSamplingCard();loadQueueAdvisorCard();return",
	} {
		if strings.Contains(block, stale) {
			t.Fatalf("填号码但未选门店时不应提前清空/返回，否则 GitHub/全局基准图会被置空：%s\n%s", stale, block)
		}
	}
}

func TestEmbeddedMobileMediaQueriesDoNotOverrideNarrowPhones(t *testing.T) {
	if !strings.Contains(indexHTML, "@media(min-width:601px) and (max-width:768px)") {
		t.Fatalf("600-768px 双列规则必须带 min-width:601px，避免覆盖 max-width:600px 的单列手机布局")
	}
	if strings.Contains(indexHTML, "/* 中等宽度（平板竖屏 / 大手机 600-768px）：多列网格降为 2 列，避免拥挤 */\n@media(max-width:768px)") {
		t.Fatalf("中等宽度媒体查询不应是裸 max-width:768px，否则 600px 以下也会被改回两列")
	}
}

func TestEmbeddedQueueEtaLabelsUseNumberUnits(t *testing.T) {
	if !strings.Contains(indexHTML, "预计 '+shortTime(er.early)+'-'+shortTime(er.late)+' 能吃上'+(adv.eta.remaining_groups>0?('（还差 '+fmtN(adv.eta.remaining_groups)+' 号）'):'')") {
		t.Fatalf("ETA 区间带应把 remaining_groups 展示为还差 N 号")
	}
	if strings.Contains(indexHTML, "adv.eta.remaining_groups)+' 桌") {
		t.Fatalf("ETA remaining_groups 代表叫号差值，不应展示为“桌”")
	}
}

// TestEmbeddedJavaScriptSyntax 用 node --check 校验内嵌 JS 语法；环境没有 node 时跳过，
// 因此不引入硬依赖。CI 的 runner 自带 node，可作为语法门禁。
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

// TestSimpleModeHidesQDCloudEntry 锁住第三批收敛：简化版 #qd 页的 emptyTrendHTML
// 走纯本机采集分支，不推 GitHub/线上基准；进阶版才保留 cloud 入口。
func TestSimpleModeHidesQDCloudEntry(t *testing.T) {
	// emptyTrendHTML 必须先判 currentUIMode()!=='advanced' 走简化分支
	if !strings.Contains(indexHTML, "if(currentUIMode()!=='advanced'){") {
		t.Error("emptyTrendHTML 未按 currentUIMode 收敛 cloud 入口")
	}
	// 简化版分支文案应是纯本机采集，不含 GitHub
	simpleBranch := "copy='开启本机采集后，这家店的叫号趋势会随着使用越来越准。'"
	if !strings.Contains(indexHTML, simpleBranch) {
		t.Error("简化版 emptyTrendHTML 缺少纯本机采集引导文案")
	}
	// renderDashboardDataSource 简化版应整体隐藏
	if !strings.Contains(indexHTML, "if(currentUIMode()!=='advanced'){box.className='data-source mt16 hid'") {
		t.Error("renderDashboardDataSource 未在简化版隐藏图表数据来源块")
	}
}

// TestEmbeddedNavScrollUsesViewportGeometry 小屏顶部导航必须用相对视口几何
// 把当前栏目滚进可视区；offsetLeft 在 flex 布局下可能相对错误的 offsetParent。
func TestEmbeddedNavScrollUsesViewportGeometry(t *testing.T) {
	js := extractEmbeddedScript(t)
	if !strings.Contains(js, "function keepActiveTopNavVisible") {
		t.Fatal("missing keepActiveTopNavVisible")
	}
	if !strings.Contains(js, "getBoundingClientRect") {
		t.Fatal("keepActiveTopNavVisible should use getBoundingClientRect relative to .nav.top")
	}
	if strings.Contains(js, "a.offsetLeft") || strings.Contains(js, "offsetLeft,r=l+a.offsetWidth") {
		t.Fatal("keepActiveTopNavVisible should not rely on element.offsetLeft against nav.scrollLeft")
	}
}

// TestEmbeddedNotifySettingsFoldTarget 「设置通知」深链要落到可展开的通知卡。
func TestEmbeddedNotifySettingsFoldTarget(t *testing.T) {
	if !strings.Contains(indexHTML, `id="fold-notify"`) {
		t.Fatal(`settings page missing id="fold-notify" for focusNotifySettings`)
	}
	js := extractEmbeddedScript(t)
	if !strings.Contains(js, "fold-notify") {
		t.Fatal("focusNotifySettings should open fold-notify")
	}
}
