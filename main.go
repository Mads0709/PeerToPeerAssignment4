package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
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
	arg1, _ := strconv.ParseInt(os.Args[1], 10, 32) //Could also use flags
	ownPort := int32(arg1) + 5000

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pe := &peer{
		id:                           ownPort,
		clients:                      make(map[int32]ping.PingClient),
		ctx:                          ctx,
		timestamp:                    0,
		criticalSectionAccessCounter: 0,
	}

	// Create listener tcp on port ownPort
	list, err := net.Listen("tcp", fmt.Sprintf(":%v", ownPort))
	if err != nil {
		log.Fatalf("Failed to listen on port: %v\n", err)
	}
	grpcServer := grpc.NewServer()
	ping.RegisterPingServer(grpcServer, pe)

	go func() {
		if err := grpcServer.Serve(list); err != nil {
			log.Fatalf("failed to server %v\n", err)
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
			log.Fatalf("Could not connect: %s\n", err)
		}
		defer conn.Close()
		c := ping.NewPingClient(conn)
		pe.clients[port] = c
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		pe.sendPingToAll() //Here a peer sends an acces to critical section request every time you write in the terminal
	}
}

//****************************************************************
//****************************************************************
//****************************************************************

func (p *peer) Ping(ctx context.Context, req *ping.Request) (*ping.Reply, error) { //evaluate request
	p.timestamp += 1
	fmt.Printf("req.Timestamp: %v , p.timestamp: %v \n", req.Timestamp, p.timestamp)

	if req.Timestamp < p.timestamp {
		rep := &ping.Reply{Answer: p.timestamp, Id: p.id}
		return rep, nil
	}
	if req.Timestamp == p.timestamp && req.Id < p.id {
		rep := &ping.Reply{Answer: p.timestamp, Id: p.id}
		return rep, nil
	}

	wgDefer.Add(1)
	wgDefer.Wait()

	rep := &ping.Reply{Answer: p.timestamp, Id: p.id}
	return rep, nil

}

var wgRequests sync.WaitGroup
var wgDefer sync.WaitGroup

func Done(ctx context.Context, dm *ping.DoneMessage) {
	fmt.Printf("release")
	wgDefer.Done()
}

func (p *peer) sendPingToAll() {
	request := &ping.Request{Id: p.id, Timestamp: p.timestamp}
	for _, client := range p.clients {

		go func() {
			wgRequests.Add(1)
			reply, err := client.Ping(p.ctx, request) //Here the ping function is called
			if err != nil {
				fmt.Println("something went wrong")
			}
			fmt.Printf("Got reply from id: %v timestamp: %v\n", reply.Id, reply.Answer)
			wgRequests.Done()
		}()
		fmt.Printf("My timestamp: %v\n", p.timestamp)
	}
	wgRequests.Wait()
	p.criticalSection()

}

func (p *peer) criticalSection() {
	p.criticalSectionAccessCounter += 1
	fmt.Printf("id: %v is in the critical section and has a csCounter of: %v\n", p.id, p.criticalSectionAccessCounter)
	time.Sleep(7 * time.Second)
	fmt.Printf("id: %v is done\n", p.id)
	

	doneMessage := &ping.DoneMessage{DoneBool: true} //here you signal to the other peers that you are done with the critical section
	for _, client := range p.clients {
		client.Done(p.ctx, doneMessage)
	}

}
