package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/lonng/nano"
	"github.com/lonng/nano/examples/gamecluster/game"
	"github.com/lonng/nano/examples/gamecluster/gate"
	"github.com/lonng/nano/examples/gamecluster/store"
	"github.com/lonng/nano/serialize/json"
	"github.com/lonng/nano/session"
	"github.com/pingcap/errors"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "NanoGameClusterDemo"
	app.Author = "Nano"
	app.Description = "Nano gate and gameserver cluster demo"
	app.Commands = []cli.Command{
		{
			Name: "master",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "listen,l", Usage: "master service listen address", Value: "127.0.0.1:34567"},
			},
			Action: runMaster,
		},
		{
			Name: "gate",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "master", Usage: "master server address", Value: "127.0.0.1:34567"},
				cli.StringFlag{Name: "listen,l", Usage: "gate service listen address", Value: "127.0.0.1:34570"},
				cli.StringFlag{Name: "gate-address", Usage: "client websocket listen address", Value: "127.0.0.1:34590"},
				cli.StringFlag{Name: "redis", Usage: "redis address", Value: "127.0.0.1:6379"},
			},
			Action: runGate,
		},
		{
			Name: "game",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "master", Usage: "master server address", Value: "127.0.0.1:34567"},
				cli.StringFlag{Name: "listen,l", Usage: "gameserver service listen address", Value: "127.0.0.1:34680"},
				cli.StringFlag{Name: "redis", Usage: "redis address", Value: "127.0.0.1:6379"},
			},
			Action: runGame,
		},
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if err := app.Run(os.Args); err != nil {
		log.Fatalf("startup server error %+v", err)
	}
}

func runMaster(args *cli.Context) error {
	listen := args.String("listen")
	if listen == "" {
		return errors.Errorf("master listen address cannot empty")
	}

	log.Println("Nano gamecluster master listen address", listen)
	nano.Listen(listen,
		nano.WithMaster(),
		nano.WithSerializer(json.NewSerializer()),
		nano.WithDebugMode(),
	)
	return nil
}

func runGate(args *cli.Context) error {
	listen := args.String("listen")
	if listen == "" {
		return errors.Errorf("gate service listen address cannot empty")
	}
	gateAddr := args.String("gate-address")
	if gateAddr == "" {
		return errors.Errorf("gate client address cannot empty")
	}
	masterAddr := args.String("master")
	if masterAddr == "" {
		return errors.Errorf("master address cannot empty")
	}

	repo := store.NewRedisRepository(args.String("redis"))
	gate.Init(repo, gateAddr)
	session.Lifetime.OnClosed(gate.OnSessionClosed)

	log.Println("Nano gamecluster gate service listen address", listen)
	log.Println("Nano gamecluster gate websocket address", gateAddr)
	log.Println("Nano gamecluster master address", masterAddr)
	nano.Listen(listen,
		nano.WithAdvertiseAddr(masterAddr),
		nano.WithClientAddr(gateAddr),
		nano.WithComponents(gate.Services),
		nano.WithSerializer(json.NewSerializer()),
		nano.WithIsWebsocket(true),
		nano.WithWSPath("/nano"),
		nano.WithCheckOriginFunc(func(_ *http.Request) bool { return true }),
		nano.WithCustomerRemoteServiceRoute(gate.RouteService().Route),
		nano.WithHeartbeatInterval(5*time.Second),
		nano.WithDebugMode(),
		nano.WithNodeId(2),
	)
	return nil
}

func runGame(args *cli.Context) error {
	listen := args.String("listen")
	if listen == "" {
		return errors.Errorf("gameserver listen address cannot empty")
	}
	masterAddr := args.String("master")
	if masterAddr == "" {
		return errors.Errorf("master address cannot empty")
	}

	repo := store.NewRedisRepository(args.String("redis"))
	game.Init(repo, listen)
	session.Lifetime.OnClosed(game.OnSessionClosed)

	log.Println("Nano gamecluster gameserver listen address", listen)
	log.Println("Nano gamecluster master address", masterAddr)
	nano.Listen(listen,
		nano.WithAdvertiseAddr(masterAddr),
		nano.WithComponents(game.Services),
		nano.WithSerializer(json.NewSerializer()),
		nano.WithHeartbeatInterval(5*time.Second),
		nano.WithDebugMode(),
	)
	return nil
}
