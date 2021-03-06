// Copyright 2017-2019 Lei Ni (nilei81@gmail.com) and other contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"reflect"
	"testing"
)

func ExampleNodeHostConfig() {
	nhc := NodeHostConfig{
		WALDir:         "/data/wal",
		NodeHostDir:    "/data/dragonboat-data",
		RTTMillisecond: 200,
		// RaftAddress is the public address that will be used by others to contact
		// this NodeHost instance.
		RaftAddress: "node01.raft.company.com:5012",
	}
	_ = nhc
}

func checkValidAddress(t *testing.T, addr string) {
	if !IsValidAddress(addr) {
		t.Errorf("valid addr %s considreed as invalid", addr)
	}
}

func checkInvalidAddress(t *testing.T, addr string) {
	if IsValidAddress(addr) {
		t.Errorf("invalid addr %s considered as valid", addr)
	}
}

func TestIsValidAddress(t *testing.T) {
	va := []string{
		"192.0.0.1:12345",
		"202.96.1.23:1234",
		"myhost:214",
		"0.0.0.0:12345",
		"node1.mydomain.com.cn:12345",
		"myhost.test:12345",
		"    myhost.test:12345 ",
	}
	for _, v := range va {
		checkValidAddress(t, v)
	}
	iva := []string{
		"192.168.0.1",
		"myhost",
		"192.168.0.1:",
		"192.168.0.1:0",
		"192.168.0.1:65536",
		"192.168.0.1:-1",
		":12345",
		":",
		"#$:%",
		"mytest:again",
		"myhost:",
		"345.168.0.1:12345",
		"192.345.0.1:12345",
		"192.168.345.1:12345",
		"192.168.1.345:12345",
		"192 .168.0.1:12345",
		"myhost :12345",
		"1host:12345",
		"",
		"    ",
	}
	for _, v := range iva {
		checkInvalidAddress(t, v)
	}
}

func TestWitnessNodeCanNotBeNonVoting(t *testing.T) {
	cfg := Config{IsWitness: true, IsNonVoting: true}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("witness node can not be an observer")
	}
}

func TestWitnessCanNotTakeSnapshot(t *testing.T) {
	cfg := Config{IsWitness: true, SnapshotEntries: 100}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("witness node can not take snapshot")
	}
}

func TestDefaultEngineConfig(t *testing.T) {
	nhc := &NodeHostConfig{}
	if err := nhc.Prepare(); err != nil {
		t.Errorf("prepare failed, %v", err)
	}
	ec := GetDefaultEngineConfig()
	if !reflect.DeepEqual(&nhc.Expert.Engine, &ec) {
		t.Errorf("default engine configure not set")
	}
}
