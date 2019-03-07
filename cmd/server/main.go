package main

import (
	"flag"
	"logauditer/cache"
	"logauditer/command"
	"logauditer/dbapi"
	"logauditer/server"
	"logauditer/web"
	_ "net/http/pprof"
	"os"

	log "github.com/laik/logger"
)

func main() {
	var httpAddr = flag.String("http", ":80", "http server addr.")
	var url = flag.String("dburl", "localhost:27017", "mgo db url addr.")

	flag.Parse()

	Addr := "127.0.0.1:9992"

	//	go http.ListenAndServe(fmt.Sprintf(":%d", 12345), nil)

	sp := dbapi.NewStorageParts()

	m, err := dbapi.NewKVMongo(*url)

	log.UnSetOutFile()
	log.SetConsole()

	log.NewLogger(
		map[string]interface{}{"level": log.DEBUG},
	)

	defer log.Flush()
	//10.10.2.104_2018-12-04_DianXin-route.log
	log.Info("start logaudit server.\n")
	log.Info("monitor file {ip}_{yyyy-mm-dd}_{hostname}-{device}.log file tail.\n")

	if err != nil {
		log.Error("[ERROR] initialization db connect occur error: %s.\n", err)
		os.Exit(1)
	}
	sp.AddStoragePart(m)

	go web.NewHttpServer(*httpAddr, &web.HttpService{SP: sp})

	c := cache.NewCache()

	server, err := server.NewServer(
		command.NewParser(c),
		c,
		sp,
		dbapi.KV,
	)
	if err != nil {
		log.Error("[ERROR] initialization server occur error: %s.\n", err)
		os.Exit(1)
	}

	if err := server.Run(Addr); err != nil {
		log.Error("[ERROR] run server occur error: %s.\n", err)
		os.Exit(1)
	}
}
