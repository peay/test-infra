/*
Copyright 2016 The Kubernetes Authors.

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

package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/Sirupsen/logrus"

	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/jobs"
	"k8s.io/test-infra/prow/kube"
	"k8s.io/test-infra/prow/plugins"

	_ "k8s.io/test-infra/prow/plugins/cla"
	_ "k8s.io/test-infra/prow/plugins/close"
	_ "k8s.io/test-infra/prow/plugins/lgtm"
	_ "k8s.io/test-infra/prow/plugins/releasenote"
	_ "k8s.io/test-infra/prow/plugins/trigger"
)

var (
	port = flag.Int("port", 8888, "Port to listen on.")

	jobConfig    = flag.String("job-config", "/etc/jobs/presubmit", "Path to job config file.")
	pluginConfig = flag.String("plugin-config", "/etc/plugins/plugins", "Path to plugin config file.")

	local = flag.Bool("local", false, "Run locally for testing purposes only. Does not require secret files.")

	webhookSecretFile = flag.String("hmac-secret-file", "/etc/webhook/hmac", "Path to the file containing the GitHub HMAC secret.")
	githubTokenFile   = flag.String("github-token-file", "/etc/github/oauth", "Path to the file containing the GitHub OAuth secret.")
)

func main() {
	flag.Parse()

	var webhookSecret []byte
	var githubClient *github.Client
	var kubeClient *kube.Client
	if *local {
		logrus.Warning("Running in local mode for dev only.")

		logrus.Print("HMAC Secret: abcde12345")
		webhookSecret = []byte("abcde12345")

		githubClient = github.NewFakeClient()
		githubClient.Logger = logrus.StandardLogger()

		kubeClient = kube.NewFakeClient()
		kubeClient.Logger = logrus.StandardLogger()
	} else {
		logrus.SetFormatter(&logrus.JSONFormatter{})

		// Ignore SIGTERM so that we don't drop hooks when the pod is removed.
		// We'll get SIGTERM first and then SIGKILL after our graceful termination
		// deadline.
		signal.Ignore(syscall.SIGTERM)

		webhookSecretRaw, err := ioutil.ReadFile(*webhookSecretFile)
		if err != nil {
			logrus.WithError(err).Fatal("Could not read webhook secret file.")
		}
		webhookSecret = bytes.TrimSpace(webhookSecretRaw)

		oauthSecretRaw, err := ioutil.ReadFile(*githubTokenFile)
		if err != nil {
			logrus.WithError(err).Fatal("Could not read oauth secret file.")
		}
		oauthSecret := string(bytes.TrimSpace(oauthSecretRaw))

		dry, err := strconv.ParseBool(os.Getenv("DRY_RUN"))
		if err != nil {
			logrus.WithError(err).Fatal("Failed to parse DRY_RUN environment variable.")
		}

		if dry {
			githubClient = github.NewDryRunClient(oauthSecret)
		} else {
			githubClient = github.NewClient(oauthSecret)
		}

		kubeClient, err = kube.NewClientInCluster("default")
		if err != nil {
			logrus.WithError(err).Fatal("Error getting kube client.")
		}
	}

	jobAgent := &jobs.JobAgent{}
	if err := jobAgent.Start(*jobConfig); err != nil {
		logrus.WithError(err).Fatal("Error starting job agent.")
	}

	pluginAgent := &plugins.PluginAgent{
		PluginClient: plugins.PluginClient{
			GitHubClient: githubClient,
			KubeClient:   kubeClient,
			JobAgent:     jobAgent,
			Logger:       logrus.NewEntry(logrus.StandardLogger()),
		},
	}
	if err := pluginAgent.Start(*pluginConfig); err != nil {
		logrus.WithError(err).Fatal("Error starting plugins.")
	}

	server := &Server{
		HMACSecret: webhookSecret,
		Plugins:    pluginAgent,
	}

	// Return 200 on / for health checks.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})
	// For /hook, handle a webhook normally.
	http.Handle("/hook", server)
	logrus.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}
