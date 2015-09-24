package main

import (
	"bufio"
	"log"
	"os"
	"runtime"
	"strconv"
	"sync"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	a "github.com/chrislusf/glow/agent"
	r "github.com/chrislusf/glow/receiver"
	s "github.com/chrislusf/glow/sender"
	l "github.com/chrislusf/glow/service_discovery/leader"
)

var (
	app = kingpin.New("glow", "A command-line net channel.")

	leader     = app.Command("leader", "Start a leader process")
	leaderPort = leader.Flag("port", "listening port").Default("8930").Int()
	leaderIp   = leader.Flag("ip", "listening IP adress").Default("localhost").String()

	agent             = app.Command("agent", "Channel Agent")
	agentDir          = agent.Flag("dir", "agent folder to store computed data").Default(os.TempDir()).String()
	agentPort         = agent.Flag("port", "agent listening port").Default("8931").Int()
	agentLeaderServer = agent.Flag("leader", "leader address").Default("localhost:8930").String()
	agentDataCenter   = agent.Flag("dataCenter", "data center name").Default("defaultDataCenter").String()
	agentRack         = agent.Flag("rack", "rack name").Default("defaultRack").String()
	agentMaxExecutor  = agent.Flag("max.executors", "upper limit of executors").Default(strconv.Itoa(runtime.NumCPU())).Int()
	agentCPULevel     = agent.Flag("cpu.level", "relative computing power of single cpu core").Default("1").Int()
	agentMemoryMB     = agent.Flag("memory", "memory size in MB").Default("1024").Int()

	sender          = app.Command("send", "Send data to a channel")
	sendToChanName  = sender.Flag("to", "Name of a channel").Required().String()
	sendFile        = sender.Flag("file", "file to post.").ExistingFile()
	senderAgentPort = sender.Flag("port", "agent listening port").Default("8931").Int()
	// sendDelimiter  = sender.Flag("delimiter", "Verbose mode.").Short('d').String()

	receiver            = app.Command("receive", "Receive data from a channel")
	receiveFromChanName = receiver.Flag("from", "Name of a source channel").Required().String()
	receiverLeader      = receiver.Flag("leader", "ip:port format").Default("localhost:8930").String()
)

func main() {
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case leader.FullCommand():
		println("listening on", (*leaderIp)+":"+strconv.Itoa(*leaderPort))
		l.RunLeader((*leaderIp) + ":" + strconv.Itoa(*leaderPort))
	case sender.FullCommand():
		var wg sync.WaitGroup
		sendChan, err := s.NewChannel(*sendToChanName, *senderAgentPort, &wg)
		if err != nil {
			panic(err)
		}

		file := os.Stdin
		if *sendFile != "" {
			file, err = os.Open(*sendFile)
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()
		}

		counter := 0
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			sendChan <- scanner.Bytes()
			counter++
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
		close(sendChan)
		wg.Wait()

	case receiver.FullCommand():
		rc := r.NewReceiveChannel(*receiveFromChanName, *receiverLeader, 0)
		recvChan, err := rc.GetChannel()
		if err != nil {
			panic(err)
		}
		for m := range recvChan {
			println(string(m))
		}

	case agent.FullCommand():
		agentServer := a.NewAgentServer(*agentDir, *agentPort,
			*agentLeaderServer,
			*agentDataCenter, *agentRack,
			*agentMaxExecutor, *agentCPULevel,
			*agentMemoryMB)
		agentServer.Run()
	}
}