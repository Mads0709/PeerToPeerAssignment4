package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	ping "github.com/Mads0709/PeerToPeerAssignment4.git/grpc"
	"google.golang.org/grpc"
)

// go run . 0
type peer struct {
	ping.UnimplementedPingServer
	id                           int32
	amountOfPings                map[int32]int32
	clients                      map[int32]ping.PingClient
	ctx                          context.Context
	timestamp                    int32
	criticalSectionAccessCounter int32
}

func main() {
	arg1, _ := strconv.ParseInt(os.Args[1], 10, 32) //svare til flag
	ownPort := int32(arg1) + 5000

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pe := &peer{
		id:                           ownPort,
		amountOfPings:                make(map[int32]int32),
		clients:                      make(map[int32]ping.PingClient),
		ctx:                          ctx,
		timestamp:                    0,
		criticalSectionAccessCounter: 0,
	}

	// Create listener tcp on port ownPort
	list, err := net.Listen("tcp", fmt.Sprintf(":%v", ownPort))
	if err != nil {
		log.Fatalf("Failed to listen on port: %v", err)
	}
	grpcServer := grpc.NewServer()
	ping.RegisterPingServer(grpcServer, pe)

	go func() {
		if err := grpcServer.Serve(list); err != nil {
			log.Fatalf("failed to server %v", err)
		}
	}()

	for i := 0; i < 3; i++ {
		port := int32(5000) + int32(i)

		if port == ownPort {
			continue
		}

		var conn *grpc.ClientConn
		fmt.Printf("Trying to dial: %v\n", port)
		conn, err := grpc.Dial(fmt.Sprintf(":%v", port), grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			log.Fatalf("Could not connect: %s", err)
		}
		defer conn.Close()
		c := ping.NewPingClient(conn)
		pe.clients[port] = c
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		pe.sendPingToAll() //Her sender man access requests
	}
}

func (p *peer) Ping(ctx context.Context, req *ping.Request) (*ping.Reply, error) {
	id := req.Id
	p.amountOfPings[id] += 1 //kan slettes
	p.timestamp += 1

	for { //Her skal peern evaluere requesten før han giver grønt lys
		if req.Timestamp < p.timestamp {
			rep := &ping.Reply{Answer: p.timestamp}
			return rep, nil
		}
		if req.Timestamp == p.timestamp && req.Id < p.id {
			rep := &ping.Reply{Answer: p.timestamp}
			return rep, nil
		}
		//vænter på at vedkommende i critcal section er færdig
		rep := &ping.Reply{Answer: p.timestamp}
		return rep, nil
	}
}

func (p *peer) Done(ctx context.Context, dn *ping.DoneMessage) (*ping.Reply, error) {

	
}

func (p *peer) sendPingToAll() {
	request := &ping.Request{Id: p.id, Timestamp: p.timestamp}
	count := 0
	for id, client := range p.clients {
		reply, err := client.Ping(p.ctx, request) //Her kaldes ping
		if err != nil {
			fmt.Println("something went wrong")
		}
		fmt.Printf("Got reply from id: %v timestamp: %v\n", id, reply.Answer)
		count++
		fmt.Printf("My timestamp: %v\n", p.timestamp)
	}
	//Her skal man tjekke at man har fået ok fra alle clients
	if count == len(p.clients) {
		p.criticalSection()
		count = 0
	}
}


func (p *peer) criticalSection() {
	p.criticalSectionAccessCounter += 1
	fmt.Printf("id: %v is in the critical section and has a csCounter of: %v\n", p.id, p.criticalSectionAccessCounter)
	time.Sleep(12 * time.Second) //sleep således at man menneskeligt kan nå at requeste fra de andre terminaler
	fmt.Printf("id: %v is done\n", p.id)
	//Her skal peeren signalere til de andre peers at den er færdig med criticalSection
}
