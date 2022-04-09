package bot

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Mrs4s/MiraiGo/client"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	_console            = bufio.NewReader(os.Stdin)
	_errSMSRequestError = errors.New("sms request error")
	_log                = zap.New(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(os.Stdout),
			zapcore.DebugLevel,
		),
	)
)

func readLine() (str string) {
	str, _ = _console.ReadString('\n')
	str = strings.TrimSpace(str)
	return
}

func readLineTimeout(t time.Duration, de string) (str string) {
	r := make(chan string)
	go func() {
		select {
		case r <- readLine():
		case <-time.After(t):
		}
	}()
	str = de
	select {
	case str = <-r:
	case <-time.After(t):
	}
	return
}

func (bot Bot) loginResponseProcessor(res *client.LoginResponse) error {
	if res.Success {
		return nil
	}

	var text string

	switch res.Error {
	case client.SliderNeededError:
		_log.Warn(ErrSliderNeed)
		return bot.LoginWithQR()
	case client.NeedCaptcha:
		_log.Warn("登录需要验证码.")

		_ = os.WriteFile("captcha.jpg", res.CaptchaImage, 0o644)

		_log.Warn("请输入验证码 (captcha.jpg)： (Enter 提交)")
		text = readLine()

		os.Remove("captcha.jpg")

		res, err := bot.client.SubmitCaptcha(text, res.CaptchaSign)
		if err != nil {
			return ErrMsg{
				Err: err,
				Msg: ErrSubmitCaptcha,
			}
		}

		return bot.loginResponseProcessor(res)
	case client.SMSNeededError:
		_log.Warn(fmt.Sprintf("账号已开启设备锁, 按 Enter 向手机 %v 发送短信验证码.", res.SMSPhone))

		readLine()
		if !bot.client.RequestSMS() {
			return ErrMsg{
				Err: _errSMSRequestError,
				Msg: ErrSMSRequest,
			}
		}

		_log.Warn("请输入短信验证码： (Enter 提交)")
		text = readLine()

		res, err := bot.client.SubmitSMS(text)
		if err != nil {
			return ErrMsg{
				Err: err,
				Msg: ErrSMSRequest,
			}
		}

		return bot.loginResponseProcessor(res)
	case client.SMSOrVerifyNeededError:
		_log.Warn("账号已开启设备锁，请选择验证方式:")
		_log.Warn(fmt.Sprintf("1. 向手机 %v 发送短信验证码", res.SMSPhone))
		_log.Warn("2. 使用手机QQ扫码验证.")
		_log.Warn("请输入(1 - 2) (将在10秒后自动选择2)：")

		text = readLineTimeout(time.Second*10, "2")
		if !strings.Contains(text, "1") {
			if !bot.client.RequestSMS() {
				return ErrMsg{
					Msg: ErrSMSRequest,
				}
			}
			_log.Warn("请输入短信验证码： (Enter 提交)")
			text = readLine()

			res, err := bot.client.SubmitSMS(text)
			if err != nil {
				return ErrMsg{
					Err: err,
					Msg: ErrSMSRequest,
				}
			}
			return bot.loginResponseProcessor(res)
		}

		fallthrough
	case client.UnsafeDeviceError:
		return ErrMsg{
			Msg: fmt.Sprintf(ErrUnSafe, res.VerifyUrl),
		}
	case client.OtherLoginError, client.UnknownLoginError, client.TooManySMSRequestError:
		msg := res.ErrorMessage
		if strings.Contains(msg, "版本") {
			msg = "密码错误或账号被冻结"
		}

		if strings.Contains(msg, "冻结") {
			msg = "账号被冻结"
		}

		return ErrMsg{
			Err: errors.New(ErrLogin),
			Msg: msg,
		}
	}

	return nil
}
