// Copyright © 2026 Joseph Wright <joseph@cloudboss.co>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package deps

var Terraform = Dependency{
	Name:    "terraform",
	Version: "1.14.8",
	Format:  Zip,
	URLs: map[Platform]string{
		{"linux", "amd64"}: "https://releases.hashicorp.com/terraform/1.14.8/" +
			"terraform_1.14.8_linux_amd64.zip",
		{"darwin", "amd64"}: "https://releases.hashicorp.com/terraform/1.14.8/" +
			"terraform_1.14.8_darwin_amd64.zip",
		{"darwin", "arm64"}: "https://releases.hashicorp.com/terraform/1.14.8/" +
			"terraform_1.14.8_darwin_arm64.zip",
	},
	SHA256: map[Platform]string{
		{"linux", "amd64"}: "56a5d12f47cbc1c6bedb8f5426ae7d5df984d" +
			"1929572c24b56f4c82e9f9bf709",
		{"darwin", "amd64"}: "26dd7593d22e9d99ec09380f0869718f649be" +
			"7b7f954d888611335e6a84961f8",
		{"darwin", "arm64"}: "5593670a2d42323847bfb216db17c73a44df2" +
			"01a62f7587928bae16adeabba23",
	},
}

var Kubectl = Dependency{
	Name:    "kubectl",
	Version: "1.34.5",
	URLs: map[Platform]string{
		{"linux", "amd64"}: "https://dl.k8s.io/release/v1.34.5/" +
			"bin/linux/amd64/kubectl",
		{"darwin", "amd64"}: "https://dl.k8s.io/release/v1.34.5/" +
			"bin/darwin/amd64/kubectl",
		{"darwin", "arm64"}: "https://dl.k8s.io/release/v1.34.5/" +
			"bin/darwin/arm64/kubectl",
	},
	SHA256: map[Platform]string{
		{"linux", "amd64"}: "6a17dd8387783b3144a65535e38d02c351027" +
			"e9718ea34a6c360476cb26d28bb",
		{"darwin", "amd64"}: "61ccfd05992fcc1135818d25691e55997155e" +
			"914c98b704df7ddd542339a93cb",
		{"darwin", "arm64"}: "0c0b575db594e0f8842aa44e5d36b224bb9f8" +
			"bf4b92cd34c0547d29b61a0277f",
	},
}

var AWSIAMAuthenticator = Dependency{
	Name:    "aws-iam-authenticator",
	Version: "0.7.12",
	URLs: map[Platform]string{
		{"linux", "amd64"}: "https://github.com/kubernetes-sigs/" +
			"aws-iam-authenticator/releases/download/v0.7.12/" +
			"aws-iam-authenticator_0.7.12_linux_amd64",
		{"darwin", "amd64"}: "https://github.com/kubernetes-sigs/" +
			"aws-iam-authenticator/releases/download/v0.7.12/" +
			"aws-iam-authenticator_0.7.12_darwin_amd64",
		{"darwin", "arm64"}: "https://github.com/kubernetes-sigs/" +
			"aws-iam-authenticator/releases/download/v0.7.12/" +
			"aws-iam-authenticator_0.7.12_darwin_arm64",
	},
	SHA256: map[Platform]string{
		{"linux", "amd64"}: "73cca6175225ac72f4e0b8b23ca214043a980" +
			"97ce6047d159b1bb3abde1bfce5",
		{"darwin", "amd64"}: "065574810ebd8258f7019eea6f3bf429bd36a" +
			"663bbe774ace3429d8f4572ade9",
		{"darwin", "arm64"}: "2155d2a9d85e2aa1021c9c834b42bea07dbaf" +
			"e5601794dcf0cc02804eca50005",
	},
}

var All = []Dependency{Terraform, Kubectl, AWSIAMAuthenticator}
