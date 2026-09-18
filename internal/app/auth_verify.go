package app

import (
	"context"
	"net/http"
)

// Verification never creates or cancels tickets. A successful read cannot prove
// that a later write will pass the official service's business and risk checks.
type AuthVerifyResult struct {
	OK      bool   `json:"ok"`
	Valid   bool   `json:"valid"`
	Method  string `json:"method"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func handleAuthVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	writeJSON(w, runAuthVerify(r.Context()))
}

func runAuthVerify(ctx context.Context) AuthVerifyResult {
	report := RunAuthProbe(ctx, "")
	result := AuthVerifyResult{Method: "read_only", Message: "只读检查未能确认认证可用，请查看基础接口检查结果；未取号，也未取消任何单据。"}
	if report.Authenticated {
		result.OK, result.Valid = true, true
		result.Message = "认证查询通过；未创建或取消任何单据。实际预约、取号仍以提交时的官方结果为准。"
	} else if authProbeRejected(report) {
		result.OK = true
		result.Message = "官方只读接口拒绝了凭证，请重新获取通行证；未修改任何单据。"
	}
	if !report.OK {
		result.Detail = authProbeFailureSummary(report)
	}
	return result
}
