/*
Copyright 2017 Home Office All rights reserved.

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
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli"
)

var (
	// Version is the controller version
	Version = "v0.0.2"
	// GitSHA is the build sha
	GitSHA = "unset"
)

func main() {
	app := &cli.App{
		Name:    "kube-cert-ingress",
		Author:  "Rohith Jayawardene",
		Email:   "gambol99@gmail.com",
		Usage:   "is a service used to create a ingress for web / sni ingress for kube-cert-manager",
		Version: fmt.Sprintf("%s (git+sha: %s)", Version, GitSHA),

		OnUsageError: func(context *cli.Context, err error, isSubcommand bool) error {
			fmt.Fprintf(os.Stderr, "[error] invalid options, %s\n", err)
			return err
		},

		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "namespace",
				Usage:  "namespace where we create the ingress resources `NAME`",
				EnvVar: "KUBE_NAMESPACE",
				Value:  "kube-certificates",
			},
			cli.StringFlag{
				Name:   "kube-cert-service",
				Usage:  "the name of the kubernetes service where kube-cert-manager handles web `NAME`",
				EnvVar: "KUBE_CERT_SERVICE",
				Value:  "kube-cert-manager",
			},
			cli.IntFlag{
				Name:   "kube-cert-service-port",
				Usage:  "the service port where kube-cert-manager is listening on `PORT`",
				EnvVar: "KUBE_CERT_SERVICE_PORT",
				Value:  8080,
			},
			cli.StringFlag{
				Name:   "ingress-name",
				Usage:  "the name of ingress where we will create the acme webhooks `NAME`",
				EnvVar: "INGRESS_NAME",
				Value:  "kube-cert-webhooks",
			},
			cli.DurationFlag{
				Name:   "interval",
				Usage:  "the service port where kube-cert-manager is listening on `PORT`",
				EnvVar: "KUBE_CERT_SERVICE_PORT",
				Value:  time.Second * 10,
			},
			cli.StringFlag{
				Name:   "nginx-class",
				Usage:  "the nginx class to apply on the webhooks ingress `ANNOTATION`",
				EnvVar: "NGINX_CLASS",
				Value:  "nginx-external",
			},
			cli.StringFlag{
				Name:   "kube-cert-label",
				Usage:  "the label of the kube-cert-manager `ANNOTATION`",
				EnvVar: "KUBE_CERT_ANNOTATION",
				Value:  "stable.k8s.psg.io/kcm.class",
			},
			cli.StringFlag{
				Name:   "kube-cert-class",
				Usage:  "the annotation value of the of  `ANNOTATION`",
				EnvVar: "KUBE_CERT_CLASS",
				Value:  "default",
			},
			cli.StringFlag{
				Name:   "kube-cert-provider-annotation",
				Usage:  "the annotation of the kube-cert-manager `ANNOTATION`",
				EnvVar: "KUBE_CERT_PROVIDER_ANNOTATION",
				Value:  "stable.k8s.psg.io/kcm.provider",
			},
			cli.BoolFlag{
				Name:   "enable-metrics",
				Usage:  "indicates you wish to enable prometheus metrics `BOOL`",
				EnvVar: "ENABLE_METRICS",
			},
			cli.BoolFlag{
				Name:   "enable-events",
				Usage:  "indicates you wish to log kubernetes events `BOOL`",
				EnvVar: "ENABLE_EVENTS",
			},
			cli.BoolTFlag{
				Name:   "verbose",
				Usage:  "enable verbose logging `BOOL`",
				EnvVar: "VERBOSE",
			},
		},

		Action: func(cx *cli.Context) error {
			svc, err := newKubeCertIngressController(&Config{
				EnableEvents:               cx.Bool("enable-events"),
				IngressClass:               cx.String("ingress-class"),
				IngressName:                cx.String("ingress-name"),
				Interval:                   cx.Duration("interval"),
				KubeCertClass:              cx.String("kube-cert-class"),
				KubeCertClassLabel:         cx.String("kube-cert-label"),
				KubeCertProviderAnnotation: cx.String("kube-cert-provider-annotation"),
				KubeCertService:            cx.String("kube-cert-service"),
				KubeCertServicePort:        cx.Int("kube-cert-service-port"),
				Namespace:                  cx.String("namespace"),
				Verbose:                    cx.Bool("verbose"),
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "[error] unable to create controller: %s\n", err)
				os.Exit(1)
			}

			if err := svc.serviceProcessor(); err != nil {
				fmt.Fprintf(os.Stderr, "[error] unable to start the service processor: %s\n", err)
				os.Exit(1)
			}

			return nil
		},
	}

	app.Run(os.Args)
}
