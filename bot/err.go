package bot

import "fmt"

const (
	ErrModNil    = "Module 为空"
	ErrModNoName = "Module 名称为空"
	ErrModExist  = "Module %s 已经注册"
	ErrModInit   = "Module 初始化错误"
)

const (
	ErrLogin         = "登录错误"
	ErrLoadQR        = "二维码解析错误"
	ErrQRTimeout     = "二维码过期"
	ErrQRCancel      = "扫码被用户取消"
	ErrSliderNeed    = "登录需要滑条验证码, 请使用手机QQ扫描二维码以继续登录"
	ErrSubmitCaptcha = "验证码错误"
	ErrSMSRequest    = "发送验证码失败"
	ErrUnSafe        = "账号已开启设备锁，请前往 -> %v <- 验证后重启Bot."
)

const (
	ErrDeviceLoad = "device 文件错误"
	ErrLoad       = "解析错误"
)

type ErrMsg struct {
	Err error
	Msg string
}

var _ error = (*ErrMsg)(nil)

func (errMsg ErrMsg) Error() string {
	if errMsg.Err == nil {
		return errMsg.Msg
	}

	return fmt.Sprintf("%s: %s", errMsg.Msg, errMsg.Err.Error())
}
