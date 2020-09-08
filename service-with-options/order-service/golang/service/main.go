package main

import (
	"context"
	"fmt"
	wrapper "github.com/golang/protobuf/ptypes/wrappers"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"log"
	"net"
	pb "ordermgt/service/ecommerce"
)
const (
	port = ":50051"
)

// server is used to implement order management service.
type service struct {
	orderMap map[string]*pb.Order
	oauth2Provider *pb.OAuth2Provider
	methods map[string]*desc.MethodDescriptor
}

// Simple RPC
func (s *service) AddOrder(ctx context.Context, orderReq *pb.Order) (*wrapper.StringValue, error) {
	log.Printf("Order Added. ID : %v", orderReq.Id)
	s.orderMap[orderReq.Id] = orderReq
	return &wrapper.StringValue{Value: "Order Added: " + orderReq.Id}, nil
}

// Simple RPC
func (s *service) GetOrder(ctx context.Context, orderId *wrapper.StringValue) (*pb.Order, error) {
	order, exists := s.orderMap[orderId.Value]
	if exists {
		return order, status.New(codes.OK, "").Err()
	}
	return nil, status.Errorf(codes.NotFound, "Order does not exist. : ", orderId)
}


func main() {
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	server := grpc.NewServer()
	service := service{}
	initSampleData(&service)
	pb.RegisterOrderManagementService(server, &pb.OrderManagementService{AddOrder: service.AddOrder, GetOrder: service.GetOrder})

	// when done registering services, you can get the descriptors from the
	// gRPC server and stash them into the map that the interceptor uses:
	// (error handling omitted for brevity)
	sds, _ := grpcreflect.LoadServiceDescriptors(server)
	for _, sd := range sds {
		sopts := sd.GetServiceOptions() // we get the descriptor's options
		// and then get the custom option from that message
		log.Printf("Ops: %v", sopts.String())
		value := proto.GetExtension(sopts, pb.E_Oauth2Provider)
		provider, ok := value.(*pb.OAuth2Provider)

		if ok {
			service.oauth2Provider = provider
			log.Printf("oauth2 provider url: %v", provider.GetUrl())
		}

		for _, md := range sd.GetMethods() {
			methodName := fmt.Sprintf("/%s/%s", sd.GetFullyQualifiedName(), md.GetName())
			log.Printf("methodName: %v, md: %v", methodName, md.String())
			if service.methods == nil {
				service.methods = make(map[string]*desc.MethodDescriptor)
			}
			service.methods[methodName] = md
		}
	}

	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func initSampleData(s *service) {
	if s.orderMap == nil {
		s.orderMap = make(map[string]*pb.Order)
	}
	s.orderMap["102"] = &pb.Order{Id: "102", Items: []string{"Google Pixel 3A", "Mac Book Pro"}, Destination: "Mountain View, CA", Price: 1800.00}
	s.orderMap["103"] = &pb.Order{Id: "103", Items: []string{"Apple Watch S4"}, Destination: "San Jose, CA", Price: 400.00}
	s.orderMap["104"] = &pb.Order{Id: "104", Items: []string{"Google Home Mini", "Google Nest Hub"}, Destination: "Mountain View, CA", Price: 400.00}
	s.orderMap["105"] = &pb.Order{Id: "105", Items: []string{"Amazon Echo"}, Destination: "San Jose, CA", Price: 30.00}
	s.orderMap["106"] = &pb.Order{Id: "106", Items: []string{"Amazon Echo", "Apple iPhone XS"}, Destination: "Mountain View, CA", Price: 300.00}
}

// Redact clears every sensitive field in pb.
func redact(msg proto.Message) {
	msgReflect := msg.ProtoReflect()
	msgReflect.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		log.Printf(v.String())
		opts := fd.Options().(*descriptorpb.FieldOptions)
		log.Printf(opts.String())
		if proto.GetExtension(opts, pb.E_Sensitive).(bool) {
			msgReflect.Clear(fd)
			return true
		}
		return true
	})
}
