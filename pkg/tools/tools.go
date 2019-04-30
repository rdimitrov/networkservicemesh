// Copyright (c) 2018 Cisco and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tools

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/go-errors/errors"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc"
)

const (
	// location of network namespace for a process
	netnsfile = "/proc/self/ns/net"
	// MaxSymLink is maximum length of Symbolic Link
	MaxSymLink = 8192
)

// GetCurrentNS discoveres the namespace of a running process and returns in a string.
func GetCurrentNS() (string, error) {
	buf := make([]byte, MaxSymLink)
	numBytes, err := syscall.Readlink(netnsfile, buf)
	if err != nil {
		return "", err
	}
	link := string(buf[0:numBytes])
	nsRegExp := regexp.MustCompile("net:\\[(.*)\\]")
	submatches := nsRegExp.FindStringSubmatch(link)
	if len(submatches) >= 1 {
		return submatches[1], nil
	}
	return "", fmt.Errorf("namespace is not found")
}

// SocketCleanup check for the presense of a stale socket and if it finds it, removes it.
func SocketCleanup(listenEndpoint string) error {
	fi, err := os.Stat(listenEndpoint)
	if err == nil && (fi.Mode()&os.ModeSocket) != 0 {
		if err := os.Remove(listenEndpoint); err != nil {
			return fmt.Errorf("cannot remove listen endpoint %s with error: %+v", listenEndpoint, err)
		}
	}
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failure stat of socket file %s with error: %+v", listenEndpoint, err)
	}
	return nil
}

// Unix socket file path.
type SocketPath string

func (socket SocketPath) Network() string {
	return "unix"
}

func (socket SocketPath) String() string {
	return string(socket)
}

func NewOSSignalChannel() chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c,
		os.Interrupt,
		// More Linux signals here
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	return c
}

// SocketOperationCheck checks for liveness of a gRPC server socket.
func SocketOperationCheck(endpoint net.Addr) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return SocketOperationCheckContext(ctx, endpoint)
}
func SocketOperationCheckContext(ctx context.Context, listenEndpoint net.Addr) (*grpc.ClientConn, error) {
	conn, err := dial(ctx, listenEndpoint)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func dial(ctx context.Context, endpoint net.Addr) (*grpc.ClientConn, error) {
	tracer := opentracing.GlobalTracer()
	c, err := grpc.DialContext(ctx, endpoint.String(), grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout(endpoint.Network(), addr, timeout)
		}),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(tracer)))

	return c, err
}

func WaitForPortAvailable(ctx context.Context, protoType string, registryAddress string, idleSleep time.Duration) error {
	if idleSleep < 0 {
		return errors.New("idleSleep must be positive")
	}
	logrus.Infof("Waiting for liveness probe: %s:%s", protoType, registryAddress)
	last := time.Now()

	for {
		select {
		case <-ctx.Done():
			return errors.New("timeout waiting for: " + protoType + ":" + registryAddress)
		default:
			var d net.Dialer
			conn, err := d.DialContext(ctx, protoType, registryAddress)
			if conn != nil {
				_ = conn.Close()
			}
			if err == nil {
				return nil
			}
			if time.Since(last) > 60*time.Second {
				logrus.Infof("Waiting for liveness probe: %s:%s", protoType, registryAddress)
				last = time.Now()
			}
			// Sleep to not overflow network
			<- time.After(idleSleep)
		}
	}
}

func parseKV(kv, kvsep string) (string, string) {
	keyValue := strings.Split(kv, kvsep)
	if len(keyValue) != 2 {
		keyValue = []string{"", ""}
	}
	return strings.Trim(keyValue[0], " "), strings.Trim(keyValue[1], " ")
}

// ParseKVStringToMap parses the input string
func ParseKVStringToMap(input, sep, kvsep string) map[string]string {
	result := map[string]string{}
	pairs := strings.Split(input, sep)
	for _, pair := range pairs {
		k, v := parseKV(pair, kvsep)
		result[k] = v
	}
	return result
}

// initJaeger returns an instance of Jaeger Tracer that samples 100% of traces and logs all spans to stdout.
func InitJaeger(service string) (opentracing.Tracer, io.Closer) {
	jaegerHost := os.Getenv("JAEGER_SERVICE_HOST")
	jaegerPort := os.Getenv("JAEGER_SERVICE_PORT_JAEGER")
	jaegerHostPort := fmt.Sprintf("%s:%s", jaegerHost, jaegerPort)

	logrus.Infof("Using Jaeger host/port: %s", jaegerHostPort)

	cfg := &config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: jaegerHostPort,
		},
	}

	hostname, err := os.Hostname()
	var serviceName string
	if err == nil {
		serviceName = fmt.Sprintf("%s@%s", service, hostname)
	} else {
		serviceName = service
	}

	tracer, closer, err := cfg.New(serviceName, config.Logger(jaeger.StdLogger))
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
}

type NsUrl struct {
	NsName string
	Intf   string
	Params url.Values
}

func parseNsUrl(urlString string) (*NsUrl, error) {
	result := &NsUrl{}

	url, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}
	path := strings.Split(url.Path, "/")
	if len(path) > 2 {
		return nil, fmt.Errorf("Invalid nsurl format")
	}
	if len(path) == 2 {
		if len(path[1]) > 15 {
			return nil, fmt.Errorf("Interface part cannot exceed 15 characters")
		}
		result.Intf = path[1]
	}
	result.NsName = path[0]
	result.Params = url.Query()
	return result, nil
}

func ParseAnnotationValue(value string) ([]*NsUrl, error) {
	var result []*NsUrl
	urls := strings.Split(value, ",")
	for _, u := range urls {
		nsurl, err := parseNsUrl(u)
		if err != nil {
			return nil, err
		}
		result = append(result, nsurl)
	}
	return result, nil
}
