package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	greetv1 "example/gen/greet/v1" // generated by protoc-gen-go
	"example/gen/greet/v1/greetv1connect" // generated by protoc-gen-connect-go
)

type GreetServer struct{}

// Unary RPC
func (s *GreetServer) Greet(
	ctx context.Context,
	req *connect.Request[greetv1.GreetRequest],
) (*connect.Response[greetv1.GreetResponse], error) {
	log.Println("Request headers: ", req.Header())
	res := connect.NewResponse(&greetv1.GreetResponse{
		Greeting: fmt.Sprintf("Hello, %s!", req.Msg.Name),
	})
	res.Header().Set("Greet-Version", "v1")
	return res, nil
}

func main() {
	greeter := &GreetServer{}
	api := http.NewServeMux()
	api.Handle(greetv1connect.NewGreetServiceHandler(greeter))

	mux := http.NewServeMux()
	// mux.Handle("/", newHTMLHandler())
	mux.Handle("/grpc/", http.StripPrefix("/grpc", api))
	http.ListenAndServe(
		"localhost:8080",
		// Use h2c so we can serve HTTP/2 without TLS.
		h2c.NewHandler(mux, &http2.Server{}),
	)
}

// Server streaming RPC
func (s *GreetServer) HelloServerStream(
	ctx context.Context,
	req *connect.Request[greetv1.GreetRequest],
	stream *connect.ServerStream[greetv1.GreetResponse],
) error {
	resCount := 5
	for i := 0; i < resCount; i++ {
		if err := stream.Send(&greetv1.GreetResponse{
			Greeting: fmt.Sprintf("[%d] Hello, %s!", i, req.Msg.Name),
		}); err != nil {
			return err
		}
		time.Sleep(time.Second * 1)
	}
	return nil
}

// Client streaming RPC
func (s *GreetServer) HelloClientStream(
	ctx context.Context,
	stream *connect.ClientStream[greetv1.GreetRequest],
) (*connect.Response[greetv1.GreetResponse], error) {
	log.Println("Request headers: ", stream.RequestHeader())
  var greeting strings.Builder
  for stream.Receive() {
    g := fmt.Sprintf("Hello, %s!\n", stream.Msg().Name)
    if _, err := greeting.WriteString(g); err != nil {
      return nil, connect.NewError(connect.CodeInternal, err)
    }
  }
  if err := stream.Err(); err != nil {
    return nil, connect.NewError(connect.CodeUnknown, err)
  }
  res := connect.NewResponse(&greetv1.GreetResponse{
    Greeting: greeting.String(),
  })
  res.Header().Set("Greet-Version", "v1")
  return res, nil
}
