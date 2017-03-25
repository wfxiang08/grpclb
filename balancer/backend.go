package balancer

import (
	"sync/atomic"

	backendpb "github.com/bsm/grpclb/grpclb_backend_v1"
	balancerpb "github.com/bsm/grpclb/grpclb_balancer_v1"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
)

type backend struct {
	cc  *grpc.ClientConn
	cln backendpb.LoadReportClient

	target  string
	address string
	score   int64
}

func newBackend(target, address string) (*backend, error) {
	cc, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	b := &backend{
		cc:  cc,
		cln: backendpb.NewLoadReportClient(cc),

		target:  target,
		address: address,
	}

	if err := b.UpdateScore(); err != nil {
		b.Close()
		return nil, err
	}

	return b, nil
}

func (b *backend) Server() *balancerpb.Server {
	return &balancerpb.Server{
		Address: b.address,
		Score:   b.Score(),
	}
}

func (b *backend) Score() int64 {
	return atomic.LoadInt64(&b.score)
}

func (b *backend) UpdateScore() error {
	resp, err := b.cln.Load(context.Background(), &backendpb.LoadRequest{})
	if err != nil {
		if grpc.Code(err) == codes.Unimplemented {
			return nil
		}
		grpclog.Printf("error retrieving load score for %s from %s: %s", b.target, b.address, err)
		return err
	}
	atomic.StoreInt64(&b.score, resp.Score)
	return nil
}

func (b *backend) Close() error {
	return b.cc.Close()
}
