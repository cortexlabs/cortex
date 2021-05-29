/*
Copyright 2021 Cortex Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package probe

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/proxy"
	"go.uber.org/zap"
	kcore "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	_initialDelaySeconds     = int32(1)
	_defaultTimeoutSeconds   = int32(1)
	_defaultPeriodSeconds    = int32(1)
	_defaultSuccessThreshold = int32(1)
	_defaultFailureThreshold = int32(1)
)

type Probe struct {
	*kcore.Probe
	logger     *zap.SugaredLogger
	healthy    bool
	hasRunOnce bool
}

func NewProbe(probe *kcore.Probe, logger *zap.SugaredLogger) *Probe {
	return &Probe{
		Probe:  probe,
		logger: logger,
	}
}

func NewDefaultProbe(target string, logger *zap.SugaredLogger) *Probe {
	targetURL, err := url.Parse(target)
	if err != nil {
		panic(fmt.Sprintf("failed to parse target URL: %v", err))
	}

	return &Probe{
		Probe: &kcore.Probe{
			Handler: kcore.Handler{
				TCPSocket: &kcore.TCPSocketAction{
					Port: intstr.FromString(targetURL.Port()),
					Host: targetURL.Hostname(),
				},
			},
			InitialDelaySeconds: _initialDelaySeconds,
			TimeoutSeconds:      _defaultTimeoutSeconds,
			PeriodSeconds:       _defaultPeriodSeconds,
			SuccessThreshold:    _defaultSuccessThreshold,
			FailureThreshold:    _defaultFailureThreshold,
		},
		logger: logger,
	}
}

func (p *Probe) StartProbing() chan bool {
	stop := make(chan bool)

	ticker := time.NewTicker(time.Duration(p.PeriodSeconds) * time.Second)
	time.AfterFunc(time.Duration(p.InitialDelaySeconds)*time.Second, func() {
		successCount := int32(0)
		failureCount := int32(0)

		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				healthy := p.probeContainer()
				if healthy {
					successCount++
					failureCount = 0
				} else {
					failureCount++
					successCount = 0
				}

				if successCount >= p.SuccessThreshold {
					p.healthy = true
				} else if failureCount >= p.FailureThreshold {
					p.healthy = false
				}
				p.hasRunOnce = true
			}
		}
	})

	return stop
}

func (p *Probe) IsHealthy() bool {
	return p.healthy
}

func (p *Probe) HasRunOnce() bool {
	return p.hasRunOnce
}

func (p *Probe) probeContainer() bool {
	var err error

	switch {
	case p.HTTPGet != nil:
		err = p.httpProbe()
	case p.TCPSocket != nil:
		err = p.tcpProbe()
	case p.Exec != nil:
		// Should never be reachable.
		p.logger.Error("exec probe not supported")
		return false
	default:
		p.logger.Warn("no probe found")
		return false
	}

	if err != nil {
		p.logger.Warn(err)
		return false
	}
	return true
}

func (p *Probe) httpProbe() error {
	targetURL := s.EnsurePrefix(
		net.JoinHostPort(p.HTTPGet.Host, p.HTTPGet.Port.String())+s.EnsurePrefix(p.HTTPGet.Path, "/"),
		"http://",
	)

	httpClient := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return err
	}

	req.Header.Add(proxy.UserAgentKey, proxy.KubeProbeUserAgentPrefix)

	for _, header := range p.HTTPGet.HTTPHeaders {
		req.Header.Add(header.Name, header.Value)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		// Ensure body is both read _and_ closed so it can be reused for keep-alive.
		// No point handling errors, connection just won't be reused.
		_, _ = io.Copy(ioutil.Discard, res.Body)
		_ = res.Body.Close()
	}()

	// response status code between 200-399 indicates success
	if !(res.StatusCode >= 200 && res.StatusCode < 400) {
		return fmt.Errorf("HTTP probe did not respond Ready, got status code: %d", res.StatusCode)
	}

	return nil
}

func (p *Probe) tcpProbe() error {
	timeout := time.Duration(p.TimeoutSeconds) * time.Second
	address := net.JoinHostPort(p.TCPSocket.Host, p.TCPSocket.Port.String())
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}
