package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	defaultEtcdTTLSeconds = 10
)

func RegisterGrpcToEtcd(grpcAddr string) (func() error, error) {
	endpointsEnv := strings.TrimSpace(os.Getenv("ETCD_ENDPOINTS"))
	serviceName := strings.TrimSpace(os.Getenv("ETCD_SERVICE_NAME"))
	if endpointsEnv == "" || serviceName == "" || grpcAddr == "" {
		return func() error { return nil }, nil
	}
	endpoints := strings.Split(endpointsEnv, ",")
	for i := range endpoints {
		endpoints[i] = strings.TrimSpace(endpoints[i])
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	instanceID, err := buildInstanceID()
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	key := fmt.Sprintf("/%s/%s", serviceName, instanceID)

	lease, err := client.Grant(context.Background(), defaultEtcdTTLSeconds)
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	_, err = client.Put(context.Background(), key, grpcAddr, clientv3.WithLease(lease.ID))
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	keepAliveCh, err := client.KeepAlive(context.Background(), lease.ID)
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	go func() {
		for range keepAliveCh {
		}
	}()

	return func() error {
		_, _ = client.Revoke(context.Background(), lease.ID)
		return client.Close()
	}, nil
}

func buildInstanceID() (string, error) {
	host, err := os.Hostname()
	if err != nil {
		return "", err
	}
	if host == "" {
		return "", errors.New("empty hostname")
	}
	return fmt.Sprintf("%s-%d", host, os.Getpid()), nil
}

// DiscoverGrpcFromEtcd 从 etcd 发现 gRPC 服务实例，返回地址列表
// 读取环境变量 ETCD_ENDPOINTS 和 ETCD_SERVICE_NAME
func DiscoverGrpcFromEtcd() ([]string, error) {
	endpointsEnv := strings.TrimSpace(os.Getenv("ETCD_ENDPOINTS"))
	serviceName := strings.TrimSpace(os.Getenv("ETCD_SERVICE_NAME"))
	if endpointsEnv == "" || serviceName == "" {
		return nil, nil
	}
	endpoints := strings.Split(endpointsEnv, ",")
	for i := range endpoints {
		endpoints[i] = strings.TrimSpace(endpoints[i])
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	defer client.Close()

	prefix := fmt.Sprintf("/%s/", serviceName)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	addrs := make([]string, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		addr := string(kv.Value)
		if addr != "" {
			addrs = append(addrs, addr)
		}
	}
	return addrs, nil
}
