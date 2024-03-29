package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"

	greetv1 "example/gen/greet/v1"
	"example/gen/greet/v1/greetv1connect"
)

var (
	scanner *bufio.Scanner
	client  greetv1connect.GreetServiceClient
)

func main() {
	fmt.Println("start connect client.")

	// 標準入力から文字列を受け取るスキャナを用意
	scanner = bufio.NewScanner(os.Stdin)

	// クライアントを生成
	httpClient := &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
	}
	client = greetv1connect.NewGreetServiceClient(
		httpClient,
		"http://localhost:8080",
		connect.WithGRPC(),
	)

	for {
		fmt.Println("1: send Request")
		fmt.Println("2: HelloServerStream")
		fmt.Println("3: HelloClientStream")
		fmt.Println("4: HelloBiStream")
		fmt.Println("5: exit")
		fmt.Print("please enter >")

		scanner.Scan()
		in := scanner.Text()

		switch in {
		case "1":
			Hello()

		case "2":
			HelloServerStream()

		case "3":
			HelloClientStream()

		case "4":
			HelloBidiStream()

		case "5":
			fmt.Println("bye.")
			goto M
		}
	}
M:
}

// Unary RPC
func Hello() {
	fmt.Println("Please enter your name.")
	scanner.Scan()
	name := scanner.Text()

	req := &greetv1.GreetRequest{
		Name: name,
	}
	res, err := client.Greet(
		context.Background(),
		connect.NewRequest(req),
	)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(res.Msg.Greeting)
}

// Server streaming RPC
func HelloServerStream() {
	fmt.Println("Please enter your name.")
	scanner.Scan()
	name := scanner.Text()

	req := &greetv1.GreetRequest{
		Name: name,
	}
	stream, err := client.HelloServerStream(context.Background(), connect.NewRequest(req))
	if err != nil {
		fmt.Println(err)
		return
	}

	for stream.Receive() {
		fmt.Println(stream.Msg().Greeting)
	}
	stream.Close()
}

// Client streaming RPC
func HelloClientStream() {
	stream := client.HelloClientStream(context.Background())

	sendCount := 5
	fmt.Printf("Please enter %d names.\n", sendCount)
	for i := 0; i < sendCount; i++ {
		scanner.Scan()
		name := scanner.Text()

		if err := stream.Send(&greetv1.GreetRequest{
			Name: name,
		}); err != nil {
			fmt.Println(err)
			return
		}
	}

	res, err := stream.CloseAndReceive()
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(res.Msg.Greeting)
	}
}

// Bidirectional streaming RPC
func HelloBidiStream() {
	stream := client.HelloBidiStream(context.Background())

	sendNum := 5
	fmt.Printf("Please enter %d names.\n", sendNum)

	var sendEnd, recvEnd bool
	sendCount := 0
	for !(sendEnd && recvEnd) {
		// 送信処理
		if !sendEnd {
			scanner.Scan()
			name := scanner.Text()
			req := &greetv1.GreetRequest{
				Name: name,
			}

			sendCount++
			if err := stream.Send(req); err != nil {
				fmt.Println(err)
				return
			}
			if sendCount == sendNum {
				sendEnd = true
				if err := stream.CloseRequest(); err != nil {
					fmt.Println(err)
				}
			}
		}

		// 受信処理
		if !recvEnd {
			if res, err := stream.Receive(); err != nil {
				if !errors.Is(err, io.EOF) {
					fmt.Println(err)
				}
				stream.CloseResponse()
				recvEnd = true
			} else {
				fmt.Println(res)
			}
		}
	}
}
