package metadata

import (
	"context"
	// "strings"
	// "google.golang.org/grpc/metadata"
)

func ExtractFromMetadata(ctx context.Context, key string) ([]string, bool) {
	return []string{"local-user@instill.tech"}, true
	// if data, ok := metadata.FromIncomingContext(ctx); !ok {
	// 	return []string{}, false
	// } else {
	// 	return data[strings.ToLower(key)], true
	// }
}
