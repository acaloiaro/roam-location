package starlink

import (
	"context"
	"fmt"
	"os"

	device "github.com/clarkzjw/starlink-grpc-golang/pkg/spacex.com/api/device"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func host() string {
	if h := os.Getenv("STARLINK_HOST"); h != "" {
		return h
	}
	return "192.168.100.1:9200"
}

// GetLocation fetches the current GPS coordinates from the Starlink dish.
func GetLocation(ctx context.Context) (lat, lon float64, err error) {
	conn, err := grpc.NewClient(host(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return 0, 0, fmt.Errorf("connecting to starlink: %w", err)
	}
	defer conn.Close()

	resp, err := device.NewDeviceClient(conn).Handle(ctx, &device.Request{
		Request: &device.Request_GetLocation{
			GetLocation: &device.GetLocationRequest{
				Source: device.PositionSource_AUTO,
			},
		},
	})
	if err != nil {
		return 0, 0, fmt.Errorf("fetching location: %w", err)
	}

	lla := resp.GetGetLocation().GetLla()
	return lla.GetLat(), lla.GetLon(), nil
}
