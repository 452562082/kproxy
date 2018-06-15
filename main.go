package main

import (
	"kuaishangtong/common/utils/log"
	"kuaishangtong/common/zk"
	"fmt"
)

func main() {
	if !initFlags() {
		return
	}

	err := initConfig(_flags.Config)
	if err != nil {
		log.Fatal(err)
	}

	// log 设置
	logSet := defaultConfig.Log
	log.SetLogFuncCall(logSet.ShowLines)
	log.SetColor(logSet.Coloured)
	log.SetLevel(logSet.Level)
	if logSet.Enable {
		log.SetLogFile(
			logSet.File,
			logSet.Level,
			logSet.Daily,
			logSet.Coloured,
			logSet.ShowLines,
			logSet.MaxDays)
	}

	zkclient, err := zk.NewGozkClient(
		defaultConfig.Zookeeper.ZookeeperHosts,
		defaultConfig.Zookeeper.ZookeeperServiceNode,nil, false)
	if err != nil {
		log.Fatal(err)
		return
	}

	go func() {
		for {
			select {
			case c := <-zkclient.GetChildren():
				fmt.Println(c)
			}
		}
	}()

	select{}
}