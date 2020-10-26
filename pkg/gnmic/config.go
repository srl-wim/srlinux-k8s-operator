// Copyright (c) Nokia Corporation.
// Licensed under the MIT License.

package gnmiclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/gobuffalo/envy"
	"github.com/google/gnxi/utils/xpath"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/openconfig/gnmi/proto/gnmi"
	pb "github.com/openconfig/gnmi/proto/gnmi"
)

// GnmiClient holds the state of the GNMI configuration
type GnmiClient struct {
	Username   string
	Password   string
	Proxy      bool
	NoTLS      bool
	TLSCA      string
	TLSCert    string
	TLSKey     string
	SkipVerify bool
	Insecure   bool
	Encoding   string
	Timeout    time.Duration
	Target     string
	MaxMsgSize int
	Client     gnmi.GNMIClient
}

func NewGnmiClient() *GnmiClient {
	return new(GnmiClient)
}

// ParseEnvironment loads a sibling `.env` file then looks through all environment
func (g *GnmiClient) ParseEnvironment() error {
	g.Username = envy.Get("SRL_USERNAME", "admin")
	g.Password = envy.Get("SRL_PASSWORD", "admin")
	g.Proxy = false
	g.NoTLS = false
	g.TLSCA = envy.Get("SRL_TLSCA", "")
	g.TLSCert = envy.Get("SRL_TLSCERT", "")
	g.TLSKey = envy.Get("SRL_TLSKEY", "")
	g.SkipVerify = true
	g.Insecure = false
	g.Encoding = envy.Get("SRL_ENCODING", "JSON_IETF")
	g.Timeout = 30 * time.Second
	g.Target = envy.Get("SRL_ENCODING", "172.19.19.2:57400")
	g.MaxMsgSize = 512 * 1024 * 1024

	return nil
}

// NewTLS //
func (g *GnmiClient) newTLS() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		Renegotiation:      tls.RenegotiateNever,
		InsecureSkipVerify: g.SkipVerify,
	}
	err := g.loadCerts(tlsConfig)
	if err != nil {
		return nil, err
	}
	return tlsConfig, nil
}

// loadCertificates loads certificates from file.
func (g *GnmiClient) loadCerts(tlscfg *tls.Config) error {
	if g.TLSCert != "" && g.TLSKey != "" {
		certificate, err := tls.LoadX509KeyPair(g.TLSCert, g.TLSKey)
		if err != nil {
			return err
		}
		tlscfg.Certificates = []tls.Certificate{certificate}
		tlscfg.BuildNameToCertificate()
	}
	if g.TLSCA != "" {
		certPool := x509.NewCertPool()
		caFile, err := ioutil.ReadFile(g.TLSCA)
		if err != nil {
			return err
		}
		if ok := certPool.AppendCertsFromPEM(caFile); !ok {
			return errors.New("failed to append certificate")
		}
		tlscfg.RootCAs = certPool
	}
	return nil
}

func (g *GnmiClient) Initialize() error {
	var err error

	opts := []grpc.DialOption{}

	if g.Insecure {
		opts = append(opts, grpc.WithInsecure())
	} else {
		tlsConfig, err := g.newTLS()
		if err != nil {
			return err
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	}
	ctx, cancel := context.WithCancel(context.Background())
	timeoutCtx, cancel := context.WithTimeout(ctx, g.Timeout)
	defer cancel()
	conn, err := grpc.DialContext(timeoutCtx, g.Target, opts...)
	if err != nil {
		return err
	}
	g.Client = gnmi.NewGNMIClient(conn)
	return nil

}

func buildPbUpdateList(pathValuePairs []string) ([]*pb.Update, error) {
	var pbUpdateList []*pb.Update
	for _, item := range pathValuePairs {
		pathValuePair := strings.SplitN(item, ":", 2)
		// TODO (leguo): check if any path attribute contains ':'
		if len(pathValuePair) != 2 || len(pathValuePair[1]) == 0 {
			return nil, fmt.Errorf("invalid path-value pair: %v", item)
		}
		pbPath, err := xpath.ToGNMIPath(pathValuePair[0])
		if err != nil {
			return nil, fmt.Errorf("error in parsing xpath %q to gnmi path", pathValuePair[0])
		}
		var pbVal *pb.TypedValue
		if pathValuePair[1][0] == '@' {
			jsonFile := pathValuePair[1][1:]
			jsonConfig, err := ioutil.ReadFile(jsonFile)
			if err != nil {
				return nil, fmt.Errorf("cannot read data from file %v", jsonFile)
			}
			jsonConfig = bytes.Trim(jsonConfig, " \r\n\t")
			pbVal = &pb.TypedValue{
				Value: &pb.TypedValue_JsonIetfVal{
					JsonIetfVal: jsonConfig,
				},
			}
		} else {
			if strVal, err := strconv.Unquote(pathValuePair[1]); err == nil {
				pbVal = &pb.TypedValue{
					Value: &pb.TypedValue_StringVal{
						StringVal: strVal,
					},
				}
			} else {
				if intVal, err := strconv.ParseInt(pathValuePair[1], 10, 64); err == nil {
					pbVal = &pb.TypedValue{
						Value: &pb.TypedValue_IntVal{
							IntVal: intVal,
						},
					}
				} else if floatVal, err := strconv.ParseFloat(pathValuePair[1], 32); err == nil {
					pbVal = &pb.TypedValue{
						Value: &pb.TypedValue_FloatVal{
							FloatVal: float32(floatVal),
						},
					}
				} else if boolVal, err := strconv.ParseBool(pathValuePair[1]); err == nil {
					pbVal = &pb.TypedValue{
						Value: &pb.TypedValue_BoolVal{
							BoolVal: boolVal,
						},
					}
				} else {
					pbVal = &pb.TypedValue{
						Value: &pb.TypedValue_StringVal{
							StringVal: pathValuePair[1],
						},
					}
				}
			}
		}
		pbUpdateList = append(pbUpdateList, &pb.Update{Path: pbPath, Val: pbVal})
	}
	return pbUpdateList, nil
}
