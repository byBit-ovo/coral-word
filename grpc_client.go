package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	pb "github.com/byBit-ovo/coral_word/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// CoralWordClient 封装 gRPC 客户端，支持 etcd 发现与调用
type CoralWordClient struct {
	conn   *grpc.ClientConn
	client pb.CoralWordServiceClient
	mu     sync.RWMutex
}

// NewCoralWordClient 从 etcd 发现 gRPC 服务并建立连接；若无 etcd 或发现为空则使用 GRPC_ADDR
func NewCoralWordGrpcClient() (*CoralWordClient, error) {
	addrs, err := DiscoverGrpcFromEtcd()
	if err != nil {
		log.Printf("DiscoverGrpcFromEtcd error: %v, fallback to GRPC_ADDR", err)
	}
	if len(addrs) == 0 {
		direct := strings.TrimSpace(os.Getenv("GRPC_ADDR"))
		if direct != "" {
			addrs = []string{direct}
		}
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no gRPC address: set ETCD_ENDPOINTS+ETCD_SERVICE_NAME or GRPC_ADDR")
	}

	// 取第一个可用地址建立连接（可扩展为健康检查或负载均衡）
	addr := addrs[0]
	// passthrough 使 IP:port 直接连接，不经过 DNS 解析（与旧 Dial 行为一致）
	if !strings.Contains(addr, "://") {
		addr = "passthrough:///" + addr
	}
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("grpc NewClient %s: %w", addr, err)
	}
	return &CoralWordClient{
		conn:   conn,
		client: pb.NewCoralWordServiceClient(conn),
	}, nil
}

// QueryWord 调用服务端 QueryWord
func (c *CoralWordClient) QueryWord(ctx context.Context, word string) (*pb.WordDescList, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()
	if client == nil {
		return nil, fmt.Errorf("client closed")
	}
	return client.QueryWord(ctx, &pb.WordRequest{Word: word})
}

// Close 关闭连接
func (c *CoralWordClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.client = nil
		return err
	}
	return nil
}

func showWordDesc(wd *pb.WordDesc) {
	fmt.Println(wd.Word)
	fmt.Println(wd.GetPronunciation())
	for _, def := range wd.GetDefinitions() {
		fmt.Println(def.GetPos())
		for _, meaning := range def.GetMeaning() {
			fmt.Println(meaning)
		}
	}
	fmt.Println(wd.GetDerivatives())
	fmt.Println(wd.GetExamTags())
	fmt.Println(wd.GetExample())
	fmt.Println(wd.GetExampleCn())
	for _, phrase := range wd.GetPhrases() {
		fmt.Println(phrase.GetExample())
		fmt.Println(phrase.GetExampleCn())
	}
	for _, synonym := range wd.GetSynonyms() {
		fmt.Println(synonym)
	}
}