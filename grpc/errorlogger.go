package grpc

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ErrorLogger( /* Add configs here */ ) grpc.UnaryServerInterceptor {
	return func(ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (resp interface{}, err error) {

		onPanic := func() {
			r := recover()
			if r != nil {
				err = status.Errorf(codes.Internal, "panic: %s", r)
				errmsg := fmt.Sprintf("[PANIC] %s\n\n%s", r, string(debug.Stack()))
				log.Println(errmsg)
			}
		}
		defer onPanic()

		resp, err = handler(ctx, req)
		errCode := status.Code(err)
		if errCode == codes.Unknown || errCode == codes.Internal {
			log.Println("Request handler returned an internal error - reporting it")
			return
		}
		return
	}
}
