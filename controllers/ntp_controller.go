/*
Copyright 2020 Wim Henderickx.

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

package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/openconfig/gnmi/proto/gnmi"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	srlinuxv1alpha1 "github.com/srl-wim/srlinux-k8s-operator/api/v1alpha1"
	gnmiclient "github.com/srl-wim/srlinux-k8s-operator/pkg/gnmic"
)

// NtpReconciler reconciles a Ntp object
type NtpReconciler struct {
	client.Client
	GnmiClient *gnmiclient.GnmiClient
	Log        logr.Logger
	Scheme     *runtime.Scheme
}

// +kubebuilder:rbac:groups=srlinux.henderiw.be,resources=ntps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=srlinux.henderiw.be,resources=ntps/status,verbs=get;update;patch

func (r *NtpReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("ntp", req.NamespacedName)

	// your logic here
	log.Info("reconciling SRLinux NTP")

	var ntp srlinuxv1alpha1.Ntp
	if err := r.Get(ctx, req.NamespacedName, &ntp); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	for _, ntp := range ntp.Spec.Server {
		log.Info(ntp.Address)
	}

	//var path []string
	//var value []string
	//path = append(path, "/system/ntp/admin-state")
	//value = append(value, "disable")
	//setInput := &gnmiclient.SetCmdInput{
	//	UpdatePaths:  path,
	//	UpdateValues: value,
	//}
	// setReq, err := r.GnmiClient.CreateSetRequest(setInput)
	// if err != nil {
	// 	return ctrl.Result{}, client.IgnoreNotFound(err)
	// }

	path := "/system/ntp"
	gnmiPath, err := gnmiclient.ParsePath(strings.TrimSpace(path))
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	specBytes, _ := json.Marshal(ntp.Spec)
	fmt.Printf("bytes: %s \n", specBytes)
	value := new(gnmi.TypedValue)
	value.Value = &gnmi.TypedValue_JsonIetfVal{
		JsonIetfVal: bytes.Trim(specBytes, " \r\n\t"),
	}

	gnmiPrefix, err := gnmiclient.CreatePrefix("", r.GnmiClient.Target)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	setReq := &gnmi.SetRequest{
		Prefix:  gnmiPrefix,
		Delete:  make([]*gnmi.Path, 0, 0),
		Replace: make([]*gnmi.Update, 0),
		Update:  make([]*gnmi.Update, 0),
	}

	setReq.Update = append(setReq.Update, &gnmi.Update{
		Path: gnmiPath,
		Val:  value,
	})

	_, err = r.GnmiClient.Set(ctx, setReq)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	//log.Info(resp.GetResponse())

	return ctrl.Result{}, nil
}

func (r *NtpReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&srlinuxv1alpha1.Ntp{}).
		Complete(r)
}
