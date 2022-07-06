package module

import (
	"os"
	"path"
	"sync"
	"time"

	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
// _ Moduler = (*Log)(nil)
)

type Log struct {
	log *zap.Logger
}

func (l *Log) MiraiGoModule() string {
	return "log"
}

func (l *Log) Init() error {
	// 初始化过程
	// 在此处可以进行 Module 的初始化配置
	// 如配置读取
	// l.log = bot.GetModuleLogger("logging")
	writer, err := rotatelogs.New(
		path.Join("./log", "%Y-%m-%d.log"),
		rotatelogs.WithMaxAge(7*24*time.Hour),
		rotatelogs.WithRotationTime(24*time.Hour),
	)
	if err != nil {
		return err
	}

	// 将日志文件写入文件和终端
	core := zapcore.NewTee(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(writer),
			zapcore.InfoLevel,
		),
		zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(os.Stdout),
			zapcore.DebugLevel,
		),
	)

	l.log = zap.New(core)

	return nil
}

func (l *Log) PostInit() {
	// 第二次初始化
	// 再次过程中可以进行跨Module的动作
	// 如通用数据库等等
}

func (l *Log) Serve(c *client.QQClient) {
	// 注册服务函数部分
	registerLog(c, l.log)
}

func (l *Log) Start(c *client.QQClient) {
	// 此函数会新开携程进行调用
	// ```go
	// 		go exampleModule.Start()
	// ```

	// 可以利用此部分进行后台操作
	// 如http服务器等等
}

func (m *Log) Stop(wg *sync.WaitGroup) {
	// 别忘了解锁
	defer wg.Done()
	// 结束部分
	// 一般调用此函数时，程序接收到 os.Interrupt 信号
	// 即将退出
	// 在此处应该释放相应的资源或者对状态进行保存
}

func logGroupMessage(msg *message.GroupMessage, entry *zap.Logger) {
	entry.With(
		zap.String("from", "GroupMessage"),
		zap.Int32("MessageID", msg.Id),
		zap.Int32("MessageIID", msg.InternalId),
		zap.Int64("GroupCode", msg.GroupCode),
		zap.Int64("SenderID", msg.Sender.Uin),
	).Info(msg.ToString())
}

func logPrivateMessage(msg *message.PrivateMessage, entry *zap.Logger) {
	entry.With(
		zap.String("from", "PrivateMessage"),
		zap.Int32("MessageID", msg.Id),
		zap.Int32("MessageIID", msg.InternalId),
		zap.Int64("SenderID", msg.Sender.Uin),
		zap.Int64("Target", msg.Target),
	).Info(msg.ToString())
}

func registerLog(b *client.QQClient, entry *zap.Logger) {
	b.GroupMessageEvent.Subscribe(func(qqClient *client.QQClient, groupMessage *message.GroupMessage) {
		logGroupMessage(groupMessage, entry)
	})

	b.PrivateMessageEvent.Subscribe(func(qqClient *client.QQClient, privateMessage *message.PrivateMessage) {
		logPrivateMessage(privateMessage, entry)
	})
}
