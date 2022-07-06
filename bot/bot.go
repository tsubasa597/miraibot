package bot

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	qrcodeTerminal "github.com/Baozisoftware/qrcode-terminal-go"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/gocq/qrcode"
	"github.com/tsubasa597/miraibot/module"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Bot struct {
	client  *client.QQClient
	modules map[string]module.Moduler
}

// New 初始化 Bot
func New() *Bot {
	return &Bot{
		client:  client.NewClientEmpty(),
		modules: make(map[string]module.Moduler),
	}
}

// RegisterModule - 向全局添加 Module
func (b Bot) RegisterModule(instances ...module.Moduler) error {
	for _, instance := range instances {
		if instance == nil {
			return ErrMsg{
				Msg: ErrModNil,
			}
		}

		name := instance.MiraiGoModule()
		if name == "" {
			return ErrMsg{
				Msg: ErrModNoName,
			}
		}

		if _, ok := b.modules[name]; ok {
			return ErrMsg{
				Msg: fmt.Sprintf(ErrModExist, name),
			}
		}
		b.modules[name] = instance

		if err := instance.Init(); err != nil {
			return ErrMsg{
				Err: err,
				Msg: ErrModInit,
			}
		}
		instance.PostInit()
		instance.Serve(b.client)
		go instance.Start(b.client)
	}

	return nil
}

// LoginWithPwd 使用账号密码登录
func (bot *Bot) LoginWithPwd(account int64, password string) error {
	if err := loadDevice(); err != nil {
		return ErrMsg{
			Err: err,
			Msg: ErrDeviceLoad,
		}
	}

	bot.client = client.NewClient(account, password)

	res, err := bot.client.Login()
	if err != nil {
		return ErrMsg{
			Err: err,
			Msg: ErrLogin,
		}
	}

	if err := bot.loginResponseProcessor(res); err != nil {
		return ErrMsg{
			Err: err,
			Msg: ErrLogin,
		}
	}

	return nil
}

// LoginWithToken 使用 Token 登录
func (bot Bot) LoginWithToken() error {
	if err := loadDevice(); err != nil {
		return ErrMsg{
			Err: err,
			Msg: ErrDeviceLoad,
		}
	}

	if pathExists("session.token") {
		token, err := os.ReadFile("session.token")
		if err != nil {
			return ErrMsg{
				Err: err,
				Msg: ErrLoad,
			}
		}

		if err = bot.client.TokenLogin(token); err != nil {
			return ErrMsg{
				Err: err,
				Msg: ErrLogin,
			}
		}
	}

	return nil
}

// LoginWithQR 使用二维码登录
func (bot *Bot) LoginWithQR() error {
	if err := loadDevice(); err != nil {
		return ErrMsg{
			Err: err,
			Msg: ErrDeviceLoad,
		}
	}

	bot.client = client.NewClientEmpty()

	rsp, err := bot.client.FetchQRCode()
	if err != nil {
		return ErrMsg{
			Err: err,
			Msg: ErrLogin,
		}
	}

	fi, err := qrcode.Decode(bytes.NewReader(rsp.ImageData))
	if err != nil {
		return ErrMsg{
			Err: err,
			Msg: ErrLoadQR,
		}
	}
	_ = os.WriteFile("qrcode.png", rsp.ImageData, 0o644)
	defer func() { _ = os.Remove("qrcode.png") }()

	time.Sleep(time.Second)

	_log.Info("请使用手机QQ扫描二维码 (qrcode.png) : ")
	qrcodeTerminal.New2(
		qrcodeTerminal.ConsoleColors.BrightBlack,
		qrcodeTerminal.ConsoleColors.BrightWhite,
		qrcodeTerminal.QRCodeRecoveryLevels.Low,
	).Get(fi.Content).Print()

	s, err := bot.client.QueryQRCodeStatus(rsp.Sig)
	if err != nil {
		return err
	}

	prevState := s.State
	for {
		time.Sleep(time.Second)

		s, _ = bot.client.QueryQRCodeStatus(rsp.Sig)
		if s == nil {
			continue
		}

		if prevState == s.State {
			continue
		}

		prevState = s.State
		switch s.State {
		case client.QRCodeCanceled:
			return ErrMsg{
				Msg: ErrQRCancel,
			}
		case client.QRCodeTimeout:
			return ErrMsg{
				Msg: ErrQRTimeout,
			}
		case client.QRCodeWaitingForConfirm:
			_log.Info("扫码成功, 请在手机端确认登录")
		case client.QRCodeConfirmed:
			res, err := bot.client.QRCodeLogin(s.LoginInfo)
			if err != nil {
				return ErrMsg{
					Err: err,
					Msg: ErrLogin,
				}
			}
			return bot.loginResponseProcessor(res)
		case client.QRCodeImageFetch, client.QRCodeWaitingForScan:
			// ignore
		}
	}
}

// Reload 刷新 好友 和 群组 列表
func (bot Bot) Reload() error {
	if err := bot.client.ReloadFriendList(); err != nil {
		return err
	}

	if err := bot.client.ReloadGroupList(); err != nil {
		return err
	}

	return nil
}

// SaveToken 保存登录 Token
func (bot Bot) SaveToken() error {
	return os.WriteFile("session.token", bot.client.GenToken(), 0o644)
}

func loadDevice() error {
	if !pathExists("./device.json") {
		client.GenRandomDevice()
		if err := os.WriteFile(
			"device.json",
			client.SystemDeviceInfo.ToJson(),
			0o644,
		); err != nil {
			return err
		}
	}

	deviceInfo, err := os.ReadFile("device.json")
	if err != nil {
		deviceInfo = []byte{}
	}

	if err := client.SystemDeviceInfo.ReadJson(deviceInfo); err != nil {
		return err
	}

	return nil
}

func pathExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil || errors.Is(err, os.ErrExist)
}

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

func (bot *Bot) loginResponseProcessor(res *client.LoginResponse) error {
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
