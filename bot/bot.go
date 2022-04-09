package bot

import (
	"bytes"
	"errors"
	"os"
	"time"

	qrcodeTerminal "github.com/Baozisoftware/qrcode-terminal-go"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/gocq/qrcode"
)

type Bot struct {
	Controller
}

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

func (bot Bot) LoginWithToken(token []byte) error {
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

func (bot Bot) Reload() error {
	if err := bot.client.ReloadFriendList(); err != nil {
		return err
	}

	if err := bot.client.ReloadGroupList(); err != nil {
		return err
	}

	return nil
}

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
