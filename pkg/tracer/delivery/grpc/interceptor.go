package grpc

import (
	"context"

	"github.com/dehwyy/tracerfx/pkg/tracer/dspan"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req any,
		reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx, span := dspan.Start(
			ctx,
			"grpc.client.unary",
			dspan.Attr("method", method),
		)
		defer span.End()

		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.MD{}
		}

		otel.GetTextMapPropagator().Inject(
			ctx,
			&MetadataCarrier{MD: &md},
		)

		ctx = metadata.NewOutgoingContext(ctx, md)

		err := invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			span.Err(err)
			return err
		}

		return nil
	}
}

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.MD{}
		}

		ctx = otel.GetTextMapPropagator().Extract(
			ctx,
			&MetadataCarrier{MD: &md},
		)

		ctx, span := dspan.Start(
			ctx,
			"grpc.server.unary",
			dspan.Attr("method", info.FullMethod),
		)
		defer span.End()

		resp, err := handler(ctx, req)
		if err != nil {
			span.Err(err)
			return nil, err
		}

		return resp, nil
	}
}

func StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		ctx, span := dspan.Start(
			ctx,
			"grpc.client.stream",
			dspan.Attr("method", method),
		)
		defer span.End()

		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.MD{}
		}

		otel.GetTextMapPropagator().Inject(
			ctx,
			&MetadataCarrier{MD: &md},
		)

		ctx = metadata.NewOutgoingContext(ctx, md)

		stream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			span.Err(err)
			return nil, err
		}

		return stream, nil
	}
}

func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := ss.Context()

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.MD{}
		}

		ctx = otel.GetTextMapPropagator().Extract(
			ctx,
			&MetadataCarrier{MD: &md},
		)

		ctx, span := dspan.Start(
			ctx,
			"grpc.server.stream",
			dspan.Attr("method", info.FullMethod),
		)
		defer span.End()

		err := handler(srv, &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		})
		if err != nil {
			span.Err(err)
			return err
		}

		return nil
	}
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
