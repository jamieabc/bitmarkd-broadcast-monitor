package configuration

import (
	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"
)

func parseLuaConfigurationFile(fileName string, config interface{}) error {
	L := lua.NewState()
	defer L.Close()

	L.OpenLibs()

	arg := &lua.LTable{}
	arg.Insert(0, lua.LString(fileName))
	L.SetGlobal("arg", arg)

	if err := L.DoFile(fileName); nil != err {
		return err
	}

	mapperOption := gluamapper.Option{
		NameFunc: func(s string) string {
			return s
		},
		TagName: "gluamapper",
	}

	mapper := gluamapper.Mapper{Option: mapperOption}
	err := mapper.Map(L.Get(L.GetTop()).(*lua.LTable), config)
	return err
}
