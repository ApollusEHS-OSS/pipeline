// Copyright © 2020 Banzai Cloud
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

package workflow

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateNodeActivity_GenerateVMConfig(t *testing.T) {
	input := CreateNodeActivityInput{
		OrganizationID:   2,
		ClusterID:        269,
		SecretID:         "592cc302663c0755e5b121f8bda",
		ClusterName:      "vmware-test-638",
		ResourcePoolName: "resource-pool",
		FolderName:       "test",
		DatastoreName:    "DatastoreCluster",
		Node: Node{
			AdminUsername: "",
			VCPU:          2,
			RAM:           1024,
			Name:          "vmware-test-638-pool1-01",
			SSHPublicKey:  "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCcbjzbnsFLpteiglidLoYny7s93YjBq59oJEN no-reply@banzaicloud.com \n",
			UserDataScriptParams: map[string]string{
				"ClusterID":            "269",
				"ClusterName":          "vmware-test-638",
				"KubernetesMasterMode": "default",
				"KubernetesVersion":    "1.15.3",
				"NodePoolName":         "pool1",
				"OrgID":                "2",
				"PKEVersion":           "0.4.26",
				"PipelineToken":        "###token###",
				"PipelineURL":          "https://externalAddress/pipeline",
				"PipelineURLInsecure":  "false",
				"PublicAddress":        "192.168.33.29",
				"Taints":               "",
			},
			UserDataScriptTemplate: "#!/bin/sh\n#export HTTP_PROXY=\"{{ .HttpProxy }}\"\n#export HTTPS_PROXY=\"{{ .HttpsProxy }}\"\n#export NO_PROXY=\"{{ .NoProxy }}\"\n\nuntil curl -v https://banzaicloud.com/downloads/pke/pke-{{ .PKEVersion }} -o /usr/local/bin/pke; do sleep 10; done\nchmod +x /usr/local/bin/pke\nexport PATH=$PATH:/usr/local/bin/\n\nPRIVATE_IP=$(hostname -I | cut -d\" \" -f 1)\n\npke install worker --pipeline-url=\"{{ .PipelineURL }}\" \\\n--pipeline-insecure=\"{{ .PipelineURLInsecure }}\" \\\n--pipeline-token=\"{{ .PipelineToken }}\" \\\n--pipeline-org-id={{ .OrgID }} \\\n--pipeline-cluster-id={{ .ClusterID}} \\\n--pipeline-nodepool={{ .NodePoolName }} \\\n--taints={{ .Taints }} \\\n--kubernetes-cloud-provider=vsphere \\\n--kubernetes-api-server={{ .PublicAddress }}:6443 \\\n--kubernetes-infrastructure-cidr=$PRIVATE_IP/32 \\\n--kubernetes-version={{ .KubernetesVersion }} \\\n--kubernetes-pod-network-cidr=\"\"",
			TemplateName:           "centos-7-pke-202001171452",
			NodePoolName:           "pool1",
			Master:                 false,
		},
	}

	vmSpec, err := generateVMConfigs(input)
	require.NoError(t, err)

	expectedConfig := map[string]string{
		"disk.enableUUID":                        "true",
		"guestinfo.userdata.encoding":            "gzip+base64",
		"guestinfo.userdata":                     "H4sIAAAAAAAA/3yTb0/7NhDHn+dV3BIe/NDkhLTQ0WyZFv5MMEWlQLUNCQm58ZWaurY524VWvPifUkCgFvGglc/fz50uvu8ljTJBsMboibyPJo9CF7CYP3FC5tF51useMmuMytleHk2N85rP8RvEEjqkBd59sBOuHEYUdDMXRcTghUUAyS/ZWOrMTdszPltDHs5Go+Hd8Ori/5sy/kMbWHAV8M94g7j+HhlcfKVHAEF7qaAJpIAtYOq9dUWWjblecbl+hLQx80yYJ60MFy6zM2x/bC/dTzs9YAay4ChTpuFq3bud4e8gDDiFaCHfawONEUAznRsBvz5/kRABvPU5rEZn5U77X2xgbbPDq/N/q9Hp3fmw3Pnx/pTAzuEFmuCBiRhiYBPId1vazhCkdp4rBU+GZkjAmJUWldTIAqkyfv9efPZImqtKCELnsncqhtsIPmdJ7bAJhGW8nt+W7s0MdRknSbI+JUmyhRi6Z1KUnc37RgXnkdZar7+paiOwtVO59tSb6rnU3pVv0SyMkTR6dOzVvpbMQgqkcuHsFAm3OW4lWxuTyrzfSfPeYdrtpp1+0dvf727jUk+IO0+h8YGQNVJQufMxlKzb2c5ZIDlpdJmn+UH6RU1rBNPo2wG9FozjKDgk1y7F66p8cmME4NyU8eCnhuQKBZvh0hVtUYhbhRyHqqqqo+5gxY/zZdM5bcOT6rI6aq+ry+Nm/LAaa/d3bT3KeyVFbW708jfX7948HD0e9M0/pwPQhhFatfxrYxXgVrd75YIwBVR1Xf6o6noXBhfD6vr6v5OiquvoZwAAAP//9s3hwz4EAAA=",
		"guestinfo.banzaicloud-pipeline-managed": "true",
		"guestinfo.banzaicloud-cluster":          "vmware-test-638-pool1-01",
		"guestinfo.banzaicloud-nodepool":         "pool1",
	}

	if len(expectedConfig) != len(vmSpec.ExtraConfig) {
		t.Errorf("expected config size is %v, actual is %v", len(expectedConfig), len(vmSpec.ExtraConfig))
	}

	for _, config := range vmSpec.ExtraConfig {
		key := config.GetOptionValue().Key
		value := config.GetOptionValue().Value
		expectedValue, ok := expectedConfig[key]
		if !ok {
			t.Errorf("expected config key %s not found", key)
		} else if value != expectedValue {
			t.Errorf("expected config value for key %s: %s\n actual value: %s ", key, value, expectedValue)
		}
	}
}
