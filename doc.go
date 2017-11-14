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

import "time"

// Config is the configuration for the service
type Config struct {
	// EnableEvents indicates we should create events
	EnableEvents bool `yaml:"enable-events" json:"enable-events"`
	// IngressName is the name of the ingress we create
	IngressName string `yaml:"ingress-name" json:"ingress-name"`
	// Intervai is the service processor interval
	Interval time.Duration `yaml:"interval" json:"interval"`
	// KubeCertClassLabel is the kube-cert-manager class annotation
	KubeCertClassLabel string `yaml:"kube-cert-label" json:"kube-cert-label"`
	// KubeCertClass is the value of the class we are handling
	KubeCertClass string `yaml:"kube-cert-class" json:"kube-cert-class"`
	// KubeCertProviderAnnotation is the annotation for the provider
	KubeCertProviderAnnotation string `yaml:"kube-cert-provider-annotation" json:"kube-cert-provider-annotation"`
	// KubeCertService is the kubernetes service
	KubeCertService string `yaml:"kube-cert-service" json:"kube-cert-service"`
	// KubeCertServicePort is the kubernetes service
	KubeCertServicePort int `yaml:"kube-cert-service-port" json:"kube-cert-service-port"`
	// Namespace is the namespace we create the hooks in
	Namespace string `yaml:"namespace" json:"namespace"`
	// Verbose logging
	Verbose bool `yaml:"verbose" json:"verbose"`
}
