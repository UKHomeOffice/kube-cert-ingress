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
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type kubeCertIngress struct {
	// the last list of backend
	backends []string
	// the kubernetes client
	client kubernetes.Interface
	// the configuration of the service
	config *Config
}

const (
	// acmeWebChallengeEndpoint is the ACME web challenge URL
	acmeWebChallengeEndpoint = "/.well-known/acme-challenge"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
}

// newKubeCertIngressController creates a new ingress controller
func newKubeCertIngressController(config *Config) (*kubeCertIngress, error) {
	if config.Verbose {
		log.SetLevel(log.DebugLevel)
	}

	log.WithFields(log.Fields{
		"class":       config.KubeCertClass,
		"class-label": config.KubeCertClassLabel,
		"gitsha":      GitSHA,
		"provider":    config.KubeCertProviderAnnotation,
		"version":     Version,
	}).Info("starting the kube-cert-ingress controller")

	return &kubeCertIngress{
		backends: []string{},
		config:   config,
	}, nil
}

// serviceProcessor is resposible for handling the watching and firing off
// creation of ingress resources
func (k *kubeCertIngress) serviceProcessor() error {
	// @step: create a kubernetes client
	if k.client == nil {
		client, err := k.getKubernetesClient()
		if err != nil {
			return err
		}
		k.client = client
	}

	timer := time.NewTicker(k.config.Interval)

	for {
		select {
		case <-timer.C:
			log.Debug("performing an synchronization of ingresses")
			if err := k.synchronize(); err != nil {
				log.WithFields(log.Fields{
					"error": err.Error(),
				}).Error("unable to synchronize the ingresses")
			}
		}
	}
}

// synchronize is responsible for sycning the ingresses and ensure we have a
func (k *kubeCertIngress) synchronize() error {
	namespaces, err := k.client.Core().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("unable to get namespace list: %s", err)
	}

	var backends []string
	// @step: interate list and grab the ingresses
	for _, namespace := range namespaces.Items {
		// @check we are not listing our namespace
		if namespace.Name == k.config.Namespace {
			continue
		}
		// @step: get a list of ingresses in the namespace
		list, err := k.client.Extensions().Ingresses(namespace.Name).List(metav1.ListOptions{})
		if err != nil {
			log.WithFields(log.Fields{
				"namespace": namespace.Name,
				"error":     err.Error(),
			}).Error("unable to retrieve the ingresses in namespace")

			continue
		}

		log.WithFields(log.Fields{
			"ingresses": len(list.Items),
			"namespace": namespace.Name,
		}).Debug("inspectioning the namespace for ingresses")

		// @step: iterate the list of ingresses in the namespace
		for i, _ := range list.Items {
			// @check if the ingress supposed to be handled
			found := k.isHandled(&list.Items[i])
			if !found {
				log.WithFields(log.Fields{
					"name":      list.Items[i].Name,
					"namespace": namespace.Name,
				}).Debug("skipping namespace, ingress is not being handled")

				continue
			}

			// @step: get a list of tls hostname that need certificates
			hostnames := getIngressTLSHostnames(&list.Items[i])
			if len(hostnames) <= 0 {
				log.WithFields(log.Fields{
					"name":      list.Items[i].Name,
					"namespace": namespace.Name,
				}).Warn("ingress does not contain any tls hosts")

				continue
			}

			backends = append(backends, hostnames...)
		}
	}

	// we have a list of TLS hostname from ingresses which are requesting http provider.
	// we just need to update our ingress resource the backends
	sort.Strings(backends)

	changes := diff(k.backends, backends)
	if len(changes) == 0 {
		return nil
	}

	log.WithFields(log.Fields{
		"changes": len(changes),
		"names":   strings.Join(changes, ","),
	}).Infof("the above ingresses have changed")

	// @step: update the ingress with the backends
	if err := k.updateIngress(k.buildIngress(backends)); err != nil {
		return err
	}

	log.Info("successfully updated the ingress resource")
	k.backends = backends

	return nil
}

// updateIngress is responsible for updating the ingress for use
func (k *kubeCertIngress) updateIngress(ingress *extensions.Ingress) error {
	log.WithFields(log.Fields{
		"name": k.config.IngressName,
	}).Debug("attempting to update the ingress resource")

	namespace := k.config.Namespace
	name := k.config.IngressName

	var create bool
	_, err := k.client.Extensions().Ingresses(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			create = true
		}
	}
	switch create {
	case true:
		_, err = k.client.Extensions().Ingresses(namespace).Create(ingress)
	default:
		_, err = k.client.Extensions().Ingresses(namespace).Update(ingress)
	}

	return err
}

// isHandled checks if this ingress needs to be handled
func (k *kubeCertIngress) isHandled(ingress *extensions.Ingress) bool {
	// @check if the kube-cert-manager is enabled and we are using a web / sni
	class, found := ingress.GetLabels()[k.config.KubeCertClassLabel]
	if !found || class != k.config.KubeCertClass {
		return false
	}
	// @check there is a provider
	provider, found := ingress.GetAnnotations()[k.config.KubeCertProviderAnnotation]
	if !found {
		return false
	}
	// @check the provider is either http or tls
	if provider != "http" {
		return false
	}

	return true
}

// getIngressTLSHostnames returns a list of the hostnames from the TLS section
func getIngressTLSHostnames(ingress *extensions.Ingress) []string {
	var list []string
	for _, x := range ingress.Spec.TLS {
		list = append(list, x.Hosts...)
	}

	return list
}

// buildIngress is responsible for creating a ingress resource to handle the acme web hook
func (k *kubeCertIngress) buildIngress(hostnames []string) *extensions.Ingress {
	ingress := &extensions.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k.config.IngressName,
			Namespace: k.config.Namespace,
		},
		Spec: extensions.IngressSpec{},
	}
	for _, x := range hostnames {
		ingress.Spec.Rules = append(ingress.Spec.Rules, k.buildIngressBackend(x))
	}

	return ingress
}

// buildIngressBackend is reponsible for building a backend for us
func (k *kubeCertIngress) buildIngressBackend(hostname string) extensions.IngressRule {
	return extensions.IngressRule{
		Host: hostname,
		IngressRuleValue: extensions.IngressRuleValue{
			HTTP: &extensions.HTTPIngressRuleValue{
				Paths: []extensions.HTTPIngressPath{
					{
						Path: acmeWebChallengeEndpoint,
						Backend: extensions.IngressBackend{
							ServiceName: k.config.KubeCertService,
							ServicePort: intstr.FromInt(k.config.KubeCertServicePort),
						},
					},
				},
			},
		},
	}
}

// getKubernetesClient returns a kubernetes api client for us
func (k *kubeCertIngress) getKubernetesClient() (kubernetes.Interface, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

// diff checks for the difference between string slices
func diff(slice1 []string, slice2 []string) []string {
	var diff []string
	for i := 0; i < 2; i++ {
		for _, s1 := range slice1 {
			found := false
			for _, s2 := range slice2 {
				if s1 == s2 {
					found = true
					break
				}
			}
			if !found {
				diff = append(diff, s1)
			}
		}
		if i == 0 {
			slice1, slice2 = slice2, slice1
		}
	}

	return diff
}
