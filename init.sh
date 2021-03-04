#!/bin/bash

mkdir -p spec/{cloud_configs,deployments,credhub_variables,exodus,runtime_configs}

pushd spec
go mod init $(bosh int ../kit.yml --path /code | sed 's@https://@@g')/spec

ginkgo bootstrap

cat << EOF > spec_test.go
package spec_test

import (
	"path/filepath"
	"runtime"

	. "github.com/genesis-community/testkit/testing"
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
)

var _ = Describe("Interal Kit", func() {
	BeforeSuite(func() {
		_, filename, _, _ := runtime.Caller(0)
		KitDir, _ = filepath.Abs(filepath.Join(filepath.Dir(filename), "../"))
	})

	// Add tests by adding more blocks below.
	Test(Environment{
		Name:          "baseline",
		CloudConfig:   "aws",
	})

	// For more usage examples reference the spec dir of testkit itself
	// https://github.com/genesis-community/testkit/tree/master/internal/spec
	// Test(Environment{
	//      Name:          "ops-override",
	//      CloudConfig:   "aws",
	//      RuntimeConfig: "dns",
	//      CPI:           "aws",
	//      Exodus:        "test-exodus",
	//      CredhubVars: "secret",
	//      Ops: []string{
	//              "test-ops-override",
	//      },
	//      OutputMatchers: OutputMatchers{
	//              GenesisAddSecrets: ContainSubstring("this-does-not-exist"),
	//              GenesisCheck:      ContainSubstring("this-does-not-exist"),
	//              GenesisManifest:   ContainSubstring("this-does-not-exist"),
	//      },
	// })
})
EOF

cat << EOF > deployments/baseline.yml
kit:
  name: dev
  features: []

genesis:
  env:      baseline
EOF

cat << EOF > cloud_configs/aws.yml
azs:
- name: z1
  cloud_properties:
    availability_zone: [z1, z2, z3]
- name: z2
  cloud_properties:
    availability_zone: [z1, z2, z3]
- name: z3
  cloud_properties:
    availability_zone: [z1, z2, z3]

vm_types:
- name: default
  cloud_properties:
    instance_type: m5.large
    ephemeral_disk: {size: 25_000}
- name: large
  cloud_properties:
    instance_type: m5.xlarge
    ephemeral_disk: {size: 50_000}

disk_types:
- name: default
  disk_size: 3000
- name: jumpbox
  disk_size: 50_000

networks:
- name: default
  type: manual
  subnets:
  - range: 172.31.0.0/16
    gateway: 172.31.0.1
    azs: [z1, z2, z3]
    dns: [8.8.8.8]
    reserved: [ 172.31.0.1 - 172.31.0.15 ]
    static: [ 172.31.0.16 - 172.31.0.30 ]
    cloud_properties:
      subnet: foo-subnet
- name: vip
  type: vip

compilation:
  workers: 5
  reuse_compilation_vms: true
  az: z1
  vm_type: default
  network: default
EOF
