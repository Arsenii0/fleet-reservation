package api

import (
	"context"
	"fmt"
	"log"
	"net"

	// pb "" TODO ArsenP : update the generated protobuf package import path
	"github.com/arsen/fleet-reservation/internal/core/domain"
	"github.com/arsen/fleet-reservation/internal/core/ports"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type GrpcAdapter struct {
	pb.UnimplementedReservationServiceServer

	app    ports.CoreApplicationPort
	port   int
	server *grpc.Server

	// RegisterReflection enables gRPC server reflection (useful for dev/test).
	RegisterReflection bool
}

func NewGrpcAdapter(app ports.CoreApplicationPort, port int) *GrpcAdapter {
	return &GrpcAdapter{
		app:  app,
		port: port,
	}
}

func (g *GrpcAdapter) Run() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", g.port))
	if err != nil {
		log.Fatalf("failed to listen on port %d: %v", g.port, err)
	}

	g.server = grpc.NewServer()
	pb.RegisterReservationServiceServer(g.server, g)

	if g.RegisterReflection {
		reflection.Register(g.server)
	}

	log.Printf("gRPC server listening on :%d", g.port)
	if err := g.server.Serve(lis); err != nil {
		log.Fatalf("gRPC server stopped: %v", err)
	}
}

func (g *GrpcAdapter) Stop() {
	if g.server != nil {
		g.server.GracefulStop()
	}
}

func (g *GrpcAdapter) CreateReservation(ctx context.Context, req *pb.CreateReservationRequest) (*pb.CreateReservationResponse, error) {
	var resources []domain.ReservationResource
	for _, r := range req.GetResourceIds() {
		resourceID, err := uuid.Parse(r.GetResourceId())
		if err != nil {
			return nil, fmt.Errorf("invalid resource_id %q: %w", r.GetResourceId(), err)
		}

		userConfig := make(map[string]interface{})
		for k, v := range r.GetConfiguration() {
			userConfig[k] = v
		}

		resources = append(resources, domain.ReservationResource{
			ResourceID: resourceID,
			UserConfig: userConfig,
		})
	}

	durationSecs := int64(req.GetDuration().GetSeconds())
	reservation := domain.NewReservation(durationSecs, resources)

	created, err := g.app.CreateReservation(ctx, reservation)
	if err != nil {
		return nil, err
	}

	return &pb.CreateReservationResponse{
		Code:   200,
		Status: "OK",
		Data: &pb.CreateReservationResponse_CreateReservationResponseData{
			ReservationId: created.ID.String(),
		},
	}, nil
}

func (g *GrpcAdapter) GetReservation(ctx context.Context, req *pb.GetReservationRequest) (*pb.GetReservationResponse, error) {
	reservationID, err := uuid.Parse(req.GetReservationId())
	if err != nil {
		return nil, fmt.Errorf("invalid reservation_id %q: %w", req.GetReservationId(), err)
	}

	reservation, err := g.app.GetReservation(ctx, reservationID)
	if err != nil {
		return nil, err
	}

	var pbResources []*pb.GetReservationResponse_ReservationResource
	for _, r := range reservation.ReservationResources {
		pbResources = append(pbResources, &pb.GetReservationResponse_ReservationResource{
			ResourceId: r.ResourceID.String(),
			InstanceId: r.InstanceID.String(),
			Status:     string(r.InstateState),
		})
	}

	return &pb.GetReservationResponse{
		Code:   200,
		Status: "OK",
		Data: &pb.GetReservationResponse_GetReservationResponseData{
			ReservationResources: pbResources,
			Status:               domainStatusToProto(reservation.Status),
		},
	}, nil
}

func (g *GrpcAdapter) ReleaseReservation(ctx context.Context, req *pb.ReleaseReservationRequest) (*pb.ReleaseReservationResponse, error) {
	reservationID, err := uuid.Parse(req.GetReservationId())
	if err != nil {
		return nil, fmt.Errorf("invalid reservation_id %q: %w", req.GetReservationId(), err)
	}

	if err := g.app.ReleaseReservation(ctx, reservationID); err != nil {
		return nil, err
	}

	return &pb.ReleaseReservationResponse{
		Code:   200,
		Status: "OK",
	}, nil
}

func (g *GrpcAdapter) UpdateReservationDuration(_ context.Context, _ *pb.UpdateReservationDurationRequest) (*pb.UpdateReservationDurationResponse, error) {
	return nil, fmt.Errorf("UpdateReservationDuration not implemented")
}

func domainStatusToProto(s domain.ReservationStatus) pb.GetReservationResponse_ReservationStatus {
	switch s {
	case domain.ReservationStatusPending:
		return pb.GetReservationResponse_RESERVATION_STATUS_PENDING
	case domain.ReservationStatusReserving:
		return pb.GetReservationResponse_RESERVATION_STATUS_RESERVING
	case domain.ReservationStatusReserved:
		return pb.GetReservationResponse_RESERVATION_STATUS_RESERVED
	case domain.ReservationStatusFailed:
		return pb.GetReservationResponse_RESERVATION_STATUS_FAILED
	case domain.ReservationStatusReleasing:
		return pb.GetReservationResponse_RESERVATION_STATUS_RELEASING
	case domain.ReservationStatusClosed:
		return pb.GetReservationResponse_RESERVATION_STATUS_CLOSED
	case domain.ReservationStatusCleaningUp:
		return pb.GetReservationResponse_RESERVATION_STATUS_CLEANING_UP
	default:
		return pb.GetReservationResponse_RESERVATION_STATUS_UNSPECIFIED
	}
}
