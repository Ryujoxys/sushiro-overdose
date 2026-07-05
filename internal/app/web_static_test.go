package app

import (
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// 前端契约守卫：web_static.go 里的 indexHTML 是一个内嵌 HTML/JS 字符串，go build
// 只验证它是合法 Go 字符串、不校验内容。下面的测试对其做静态结构检查，让"重排丢
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
		"该功能在进阶版中",
		"advanced-only",
		"simple-mode",
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少简化/进阶模式片段：%s", needle)
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

func TestEmbeddedAdvancedOnlyMutationMarkers(t *testing.T) {
	for _, needle := range []string{
		`id="p-ca" class="hid advanced-page"`,
		`id="p-sn" class="hid advanced-page"`,
		`id="p-re" class="hid advanced-page"`,
		`id="qdSamplingFold" class="card adv mt16 advanced-only"`,
		`<details class="adv mt16 advanced-only" open>`,
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
		`buttons:[{l:'查看我的单据',f:"enterAdvanced('re')"},{l:'几点叫到我',f:"go('qd')"}]`,
		`b.onclick=()=>enterAdvanced('re')`,
		`buttons:[{l:'回首页',f:"go('da')"},{l:'查可约时段',f:"enterAdvanced('ca')"}]`,
		`currentUIMode()==='advanced'?'门店、叫号、在等桌数为公开实时信息；远程取号是会执行操作的实验性功能，确认后才会提交。':'门店、叫号、在等桌数为公开实时信息；简化版保持只读，不会替你取号。'`,
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("可见入口应通过进阶确认而不是直接跳转：%s", needle)
		}
	}
}

func TestEmbeddedPlanConverterIsFirstClass(t *testing.T) {
	// 时间换算（几点取号 ⇄ 几点吃）是产品核心价值，必须对所有模式可见（非 advanced-only），
	// 且双向用 ⇄ 换向、输入即算（debounce），不再藏在折叠/进阶门后。
	for _, needle := range []string{
		`id="qdPlanFold" class="plan-card mt16"`,
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

func TestEmbeddedUXM1PrimaryActions(t *testing.T) {
	for _, needle := range []string{
		`id="qdPrimaryActions"`,
		`class="bt bt-r bt-s" onclick="openStorePicker({selected:qdSelected.slice(0,1),multi:false,onConfirm:applyDashboardStores})">选门店`,
		`class="bt bt-w bt-s" onclick="loadQueueDashboard()">刷新`,
		`id="sc"><div class="empty"><div class="mascot-wrap">`,
		`onclick="openStorePicker({selected:selStores,onConfirm:applyCalendarStores})">选择门店`,
		`当前没有预约或排队号。<div class="mt8"><button class="bt bt-r bt-s" onclick="go(\'ca\')">约未来</button><button class="bt bt-w bt-s" onclick="go(\'qt\')">看排队</button></div>`,
	} {
		if !strings.Contains(indexHTML, needle) {
			t.Errorf("indexHTML 缺少 M1 主路径片段：%s", needle)
		}
	}

	autoPlan := regexp.MustCompile(`<summary>[^<]*自动取号计划[^<]*</summary>[\s\S]*?onclick="saveNetTicketPlan\(true\)">启用`)
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
	if !strings.Contains(indexHTML, "预计 '+shortTime(er.early)+'-'+shortTime(er.late)+' 叫到你'+(adv.eta.remaining_groups>0?('（还差 '+fmtN(adv.eta.remaining_groups)+' 号）'):'')") {
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
