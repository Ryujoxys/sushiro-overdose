package app

import api "github.com/Ryujoxys/sushiro-overdose/internal/api"

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const reservationServerErrorCooldown = 30 * time.Second

func bookingSlotKey(storeID, date, start string) string {
	return storeID + "|" + date + "|" + start
}

func isTemporaryBookingSkipped(skips map[string]time.Time, key string, now time.Time) bool {
	if skips == nil {
		return false
	}
	until, ok := skips[key]
	if !ok {
		return false
	}
	if now.Before(until) {
		return true
	}
	delete(skips, key)
	return false
}

func markTemporaryBookingSkip(skips map[string]time.Time, key string, now time.Time) {
	if skips != nil {
		skips[key] = now.Add(reservationServerErrorCooldown)
	}
}

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
		strings.Contains(text, "network") || strings.Contains(text, "dns"):
		return "网络不可达：确认网络能访问寿司郎接口，检查代理后重试"
	case strings.Contains(text, "unexpected end of json") || strings.Contains(text, "invalid character") ||
		strings.Contains(text, "json"):
		return "官方返回内容无法解析：可能是临时故障或被代理拦截，请稍后重试"
	}
	return err.Error()
}

func bookingServerErrorLog(slotLabel string, err error) string {
	if isCredentialRefreshLikelyError(err) {
		return fmt.Sprintf("%s — %s，仍会短暂跳过当前时段 %d 秒；建议立即重置通行证", slotLabel, friendlyOfficialAPIError(err), int(reservationServerErrorCooldown.Seconds()))
	}
	return fmt.Sprintf("%s — %s，跳过当前时段 %d 秒后再试", slotLabel, friendlyOfficialAPIError(err), int(reservationServerErrorCooldown.Seconds()))
}
