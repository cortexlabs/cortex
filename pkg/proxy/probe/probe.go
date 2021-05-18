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
	"github.com/pkg/errors"
	"go.uber.org/zap"
	kcore "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	_defaultInitialDelaySeconds = 1
	_defaultTimeoutSeconds      = 3
	_defaultPeriodSeconds       = 1
	_defaultSuccessThreshold    = 1
	_defaultFailureThreshold    = 5
)

type Probe struct {
	*kcore.Probe
	count  int32
	logger *zap.SugaredLogger
}

func NewProbe(probe *kcore.Probe, logger *zap.SugaredLogger) *Probe {
	return &Probe{
		Probe:  probe,
		logger: logger,
	}
}

func NewDefaultProbe(logger *zap.SugaredLogger, target string) *Probe {
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
			InitialDelaySeconds: _defaultInitialDelaySeconds,
			TimeoutSeconds:      _defaultTimeoutSeconds,
			PeriodSeconds:       _defaultPeriodSeconds,
			SuccessThreshold:    _defaultSuccessThreshold,
			FailureThreshold:    _defaultFailureThreshold,
		},
		logger: logger,
	}
}

func (p *Probe) ProbeContainer() bool {
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

func (p *Probe) doProbe(probe func(time.Duration) error) error {
	timeout := time.Duration(p.TimeoutSeconds) * time.Second
	retryInterval := time.Duration(p.PeriodSeconds) * time.Second

	var failCount int
	pollErr := wait.PollImmediate(retryInterval, timeout, func() (bool, error) {
		if err := probe(timeout); err != nil {
			// Reset count of consecutive successes to zero.
			p.count = 0
			failCount++

			if failCount >= int(p.FailureThreshold) {
				return false, errors.Wrapf(err, "probe failure exceeded (failureThreshold = %d)", p.FailureThreshold)
			}

			return false, nil
		}

		p.count++

		// Return success if count of consecutive successes is equal to or greater
		// than the probe's SuccessThreshold.
		return p.count >= p.SuccessThreshold, nil
	})

	return pollErr
}

func (p *Probe) httpProbe() error {
	return p.doProbe(func(timeout time.Duration) error {
		targetURL := s.EnsurePrefix(
			net.JoinHostPort(p.HTTPGet.Host, p.HTTPGet.Port.String())+s.EnsurePrefix(p.HTTPGet.Path, "/"),
			"http://",
		)

		httpClient := &http.Client{}
		req, err := http.NewRequest(http.MethodGet, targetURL, nil)
		if err != nil {
			return err
		}

		res, err := httpClient.Do(req)
		if err != nil {
			return err
		}

		req.Header.Add(proxy.UserAgentKey, proxy.KubeProbeUserAgentPrefix)

		for _, header := range p.HTTPGet.HTTPHeaders {
			req.Header.Add(header.Name, header.Value)
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
	})
}

func (p *Probe) tcpProbe() error {
	return p.doProbe(func(timeout time.Duration) error {
		address := net.JoinHostPort(p.TCPSocket.Host, p.TCPSocket.Port.String())
		conn, err := net.DialTimeout("tcp", address, timeout)
		if err != nil {
			return err
		}
		_ = conn.Close()
		return nil
	})
}
