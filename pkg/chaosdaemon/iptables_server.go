// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package chaosdaemon

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"

	pb "github.com/pingcap/chaos-mesh/pkg/chaosdaemon/pb"
)

const (
	iptablesCmd              = "iptables"
	iptablesBadRuleErr       = "Bad rule (does a matching rule exist in that chain?)."
	iptablesIpSetNotExistErr = "doesn't exist.\n\nTry `iptables -h' or 'iptables --help' for more information.\n"
)

func (s *Server) FlushIptables(ctx context.Context, req *pb.IpTablesRequest) (*empty.Empty, error) {
	pid, err := s.crClient.GetPidFromContainerID(ctx, req.ContainerId)
	if err != nil {
		log.Error(err, "error while getting PID")
		return nil, err
	}

	nsPath := GenNetnsPath(pid)

	rule := req.Rule

	format := ""

	switch rule.Direction {
	case pb.Rule_INPUT:
		format = "%s INPUT -m set --match-set %s src -j DROP -w 5"
	case pb.Rule_OUTPUT:
		format = "%s OUTPUT -m set --match-set %s dst -j DROP -w 5"
	default:
		return nil, fmt.Errorf("unknown rule direction")
	}

	action := ""
	switch rule.Action {
	case pb.Rule_ADD:
		action = "-A"
	case pb.Rule_DELETE:
		action = "-D"
	}

	command := fmt.Sprintf(format, action, rule.Set)

	if rule.Action == pb.Rule_DELETE {
		log.Info("deleting iptables rules")
		cmd := withNetNS(ctx, nsPath, iptablesCmd, strings.Split(command, " ")...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			output := string(out)
			if !(strings.Contains(output, iptablesBadRuleErr) || strings.Contains(output, iptablesIpSetNotExistErr)) {
				log.Error(err, "iptables error")
				return nil, err
			}
		}
	} else {
		cmd := withNetNS(ctx, nsPath, iptablesCmd, strings.Split(command, " ")...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			output := string(out)
			log.Info("run command failed", "command", fmt.Sprintf("%s %s", iptablesCmd, command), "stdout", output)
			return nil, err
		}
	}

	return &empty.Empty{}, nil
}
