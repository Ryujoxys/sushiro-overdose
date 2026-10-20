package app

import api "github.com/Ryujoxys/sushiro-overdose/internal/api"

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

func isOfficialServerHTTPError(err error) bool {
	return api.IsHTTPStatus(err, http.StatusInternalServerError)
}

func isKnownOfficialServerError(err error) bool {
	var apiErr *api.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusInternalServerError {
		return false
	}
	body := strings.ToLower(apiErr.Body)
	return strings.Contains(body, `"code":"e010"`) ||
		strings.Contains(body, `"code": "e010"`) ||
		strings.Contains(body, "error.server")
}

func isCredentialRefreshLikelyError(err error) bool {
	return isKnownOfficialServerError(err)
}

func isTicketAlreadyIssuedText(text string) bool {
	body := strings.ToLower(text)
	return strings.Contains(body, `"code":"e034"`) ||
		strings.Contains(body, `"code": "e034"`) ||
		strings.Contains(body, "too_many_tickets") ||
		strings.Contains(body, "already issued")
}

func isTicketAlreadyIssuedError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusConflict && isTicketAlreadyIssuedText(apiErr.Body)
	}
	return isTicketAlreadyIssuedText(err.Error())
}

func friendlyNetTicketError(err error) string {
	if isTicketAlreadyIssuedError(err) {
		return "官方提示这台终端已经发过排队号，但本地没有拿到号码；不要重复取号，请打开 PC 微信寿司郎小程序的排队/我的预约页查看，或开启接口调试恢复号码"
	}
	return friendlyOfficialAPIError(err)
}

func friendlyOfficialAPIError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, api.ErrActiveReservationExists) {
		return "官方仍认为当前账号已有预约；如果你刚在手机上取消，请等小程序状态同步后再抢，或重新打开寿司郎小程序确认“我的预约”已清空"
	}
	if isCredentialRefreshLikelyError(err) {
		return "官方接口返回 E010/error.server，通常是通行证已经过期或被手机端登录顶掉；请先重置通行证，再重新获取通行证后重试"
	}
	if isOfficialServerHTTPError(err) {
		return "官方接口返回 HTTP 500，已保留凭证；如果小程序也失败，通常是官方临时异常"
	}
	if msg := friendlyNetworkError(err); msg != "" {
		return msg
	}
	return err.Error()
}

// friendlyNetworkError 把 Go 底层网络/超时/解析错误归为人话；
// 对应前端 explainMsg 里 network|timeout|超时|不可达|connection 的归类。
// 未命中已知模式时回退到原始错误文本，保证调用方永远拿到非空消息。
func friendlyNetworkError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "请求超时：寿司郎接口响应较慢，请稍后重试"
	}
	text := strings.ToLower(err.Error())
	switch {
	case strings.Contains(text, "timeout") || strings.Contains(text, "超时") || strings.Contains(text, "deadline"):
		return "请求超时：寿司郎接口响应较慢，请稍后重试"
	case strings.Contains(text, "connection refused") || strings.Contains(text, "no such host") ||
		strings.Contains(text, "unreachable") || strings.Contains(text, "不可达") ||
		strings.Contains(text, "network") || strings.Contains(text, "dns") ||
		strings.Contains(text, "connection reset") || strings.Contains(text, "broken pipe") ||
		strings.Contains(text, "proxyconnect") || strings.Contains(text, "tls:") ||
		strings.Contains(text, "certificate") || strings.HasSuffix(text, "eof"):
		return "网络不可达：确认网络能访问寿司郎接口，检查代理后重试"
	case strings.Contains(text, "unexpected end of json") || strings.Contains(text, "invalid character") ||
		strings.Contains(text, "json"):
		return "官方返回内容无法解析：可能是临时故障或被代理拦截，请稍后重试"
	}
	return err.Error()
}

// Machine-readable live-data error codes returned next to the Chinese message.
const (
	liveErrAuthRequired = "live_auth_required" // 没有连接，或网关拒绝了连接（401/403/451）
	liveErrRateLimited  = "live_rate_limited"
	liveErrUnavailable  = "live_unavailable"
)

const liveAuthRequiredText = "实时叫号需要连接：打开寿司郎小程序刷新一下，再在应用里连接（约 1 小时有效）；已有记录和历史规律照常可看"

// liveErrorCode classifies a live-data failure for the page and the collector.
func liveErrorCode(err error) string {
	if errors.Is(err, ErrLiveAuthRequired) {
		return liveErrAuthRequired
	}
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden, http.StatusUnavailableForLegalReasons:
			return liveErrAuthRequired
		case http.StatusTooManyRequests:
			return liveErrRateLimited
		}
	}
	return liveErrUnavailable
}

// writeLiveError writes a live-data failure with its code; the status stays 502 so
// existing callers keep treating it as an upstream failure.
func writeLiveError(w http.ResponseWriter, err error) {
	writeErrorCode(w, http.StatusBadGateway, liveErrorCode(err), friendlyPublicQueueError(err))
}

// friendlyPublicQueueError explains failures of live queue reads. Since 2026-10 the
// mini program uses a gateway that needs the user's own member token; an old-host
// rejection or an expired connection both mean "connect again", never a VPN hint.
func friendlyPublicQueueError(err error) string {
	if errors.Is(err, ErrLiveAuthRequired) {
		return liveAuthRequiredText
	}
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		code := strconv.Itoa(apiErr.StatusCode)
		html := strings.Contains(strings.ToLower(apiErr.Body), "<html")
		switch {
		case apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden || apiErr.StatusCode == http.StatusUnavailableForLegalReasons:
			return "实时数据连接未建立或已失效：打开寿司郎小程序刷新一下，再在应用里连接一次；已有记录和历史规律照常可看"
		case apiErr.StatusCode == http.StatusTooManyRequests:
			return "请求太频繁（HTTP 429）：请稍后再试"
		case apiErr.StatusCode >= 500:
			return "寿司郎服务暂时不可用（HTTP " + code + "），请稍后重试"
		case html:
			return "寿司郎接口返回了异常页面（HTTP " + code + "），请稍后重试"
		}
	}
	return friendlyNetworkError(err)
}
