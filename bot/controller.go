package bot

import (
	"fmt"

	"github.com/Mrs4s/MiraiGo/client"
)

type Controller struct {
	client  *client.QQClient
	modules map[string]Moduler
}

func NewController() Controller {
	return Controller{
		client:  client.NewClientEmpty(),
		modules: make(map[string]Moduler),
	}
}

// RegisterModule - 向全局添加 Module
func (c Controller) RegisterModule(instances ...Moduler) error {
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

		if _, ok := c.modules[name]; ok {
			return ErrMsg{
				Msg: fmt.Sprintf(ErrModExist, name),
			}
		}
		c.modules[name] = instance

		if err := instance.Init(); err != nil {
			return ErrMsg{
				Err: err,
				Msg: ErrModInit,
			}
		}
		instance.PostInit()
		instance.Serve(c.client)
		go instance.Start(c.client)
	}

	return nil
}
