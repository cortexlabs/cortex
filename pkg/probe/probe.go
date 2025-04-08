/*
Copyright 2022 Cortex Labs, Inc.

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
	"sync"
	"time"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"go.uber.org/zap"
	kcore "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	_defaultInitialDelaySeconds = int32(1)
	_defaultTimeoutSeconds      = int32(1)
	_defaultPeriodSeconds       = int32(1)
	_defaultSuccessThreshold    = int32(1)
	_defaultFailureThreshold    = int32(1)
)

type Probe struct {
	*kcore.Probe
	sync.RWMutex
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
			ProbeHandler: kcore.ProbeHandler{
				TCPSocket: &kcore.TCPSocketAction{
					Port: intstr.FromString(targetURL.Port()),
					Host: targetURL.Hostname(),
				},
			},
			InitialDelaySeconds: _defaultInitialDelaySeconds,
			TimeoutSeconds:      _defaultTimeoutSeconds,
			PeriodSeconds:       _defaultPeriodSeconds,
			SuccessThreshold:    _defaultSuccessThreshold,
			FailureThreshold:    _defaultFailureThreshold,
		},
		logger: logger,
	}
}

func (p *Probe) StartProbing() chan struct{} {
	stop := make(chan struct{})

	time.AfterFunc(time.Duration(p.InitialDelaySeconds)*time.Second, func() {
		ticker := time.NewTicker(time.Duration(p.PeriodSeconds) * time.Second)

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

				p.Lock()

				if successCount >= p.SuccessThreshold {
					p.healthy = true
				} else if failureCount >= p.FailureThreshold {
					p.healthy = false
				}
				p.hasRunOnce = true

				p.Unlock()
			}
		}
	})

	return stop
}

func (p *Probe) IsHealthy() bool {
	p.RLock()
	defer p.RUnlock()

	return p.healthy
}

func (p *Probe) HasRunOnce() bool {
	p.RLock()
	defer p.RUnlock()

	return p.hasRunOnce
}

func AreProbesHealthy(probes []*Probe) bool {
	for _, probe := range probes {
		if probe == nil {
			continue
		}
		if !probe.IsHealthy() {
			return false
		}
	}
	return true
}

func (p *Probe) probeContainer() bool {
	var err error
	var probeType string

	switch {
	case p.HTTPGet != nil:
		err = p.httpProbe()
		probeType = "http"
	case p.TCPSocket != nil:
		err = p.tcpProbe()
		probeType = "tcp"
	case p.Exec != nil:
		// Should never be reachable.
		p.logger.Error("exec probe not supported")
		return false
	default:
		p.logger.Warn("no probe found")
		return false
	}

	if err != nil {
		p.logger.Warn(errors.Wrapf(err, "%s probe to user container failed", probeType))
		return false
	}
	return true
}

func (p *Probe) httpProbe() error {
	// to mimic k8s probe functionality
	targetHost := p.HTTPGet.Host
	if p.HTTPGet.Host == "" {
		targetHost = "localhost"
	}

	targetURL := s.EnsurePrefix(
		net.JoinHostPort(targetHost, p.HTTPGet.Port.String())+s.EnsurePrefix(p.HTTPGet.Path, "/"),
		"http://",
	)

	httpClient := &http.Client{
		Timeout: time.Duration(p.TimeoutSeconds) * time.Second,
	}
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return err
	}

	req.Header.Add(consts.UserAgentKey, consts.KubeProbeUserAgentPrefix)

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
	// to mimic k8s probe functionality
	targetHost := p.TCPSocket.Host
	if p.TCPSocket.Host == "" {
		targetHost = "localhost"
	}

	timeout := time.Duration(p.TimeoutSeconds) * time.Second
	address := net.JoinHostPort(targetHost, p.TCPSocket.Port.String())
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}
