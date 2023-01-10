// Copyright 2023 The kubegems.io Authors
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

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/spf13/cobra"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/kube"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func main() {
	configflags := genericclioptions.ConfigFlags{
		KubeConfig: pointer.String(""),
		Context:    pointer.String(""),
	}
	cmd := &cobra.Command{
		Use: os.Args[0],
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
			defer cancel()
			cfg, err := configflags.ToRESTConfig()
			if err != nil {
				return err
			}
			log.Infof("using cluster: %s", cfg.Host)
			return Run(ctx, cfg)
		},
	}
	flags := cmd.Flags()
	configflags.AddFlags(flags)
	if err := cmd.Execute(); err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}

func Run(ctx context.Context, cfg *rest.Config) error {
	db, err := setupDB(ctx, cfg)
	if err != nil {
		return err
	}
	scopes := []models.PromqlTplScope{}
	if err := db.DB().Preload("Resources.Rules").Find(&scopes).Error; err != nil {
		return err
	}
	bts, _ := yaml.Marshal(scopes)
	if err := os.WriteFile("config/promql_tpl.yaml", bts, 0644); err != nil {
		return err
	}

	tpls := []models.MonitorDashboardTpl{}
	if err := db.DB().Find(&tpls).Error; err != nil {
		return err
	}
	for _, tpl := range tpls {
		bts, _ := yaml.Marshal(tpl)
		file, err := os.OpenFile(fmt.Sprintf("config/dashboards/%s.yaml", strings.ToLower(tpl.Name)), os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			return err
		}
		if _, err = file.Write(bts); err != nil {
			return err
		}
		file.Close()
	}
	return nil
}

func setupDB(ctx context.Context, cfg *rest.Config) (*database.Database, error) {
	cli, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, err
	}
	selector := labels.SelectorFromSet(labels.Set{"app.kubernetes.io/name": "mysql"}).String()
	listenport, err := kube.PortForward(ctx, cfg, "kubegems", selector, 3306)
	if err != nil {
		return nil, err
	}
	// find mysql password
	secret := &v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "kubegems-mysql", Namespace: "kubegems"}}
	if err := cli.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
		return nil, err
	}
	mysqlrootpassword := string(secret.Data["mysql-root-password"])
	return database.NewDatabase(&database.Options{
		Addr:      fmt.Sprintf("localhost:%d", listenport),
		Username:  "root",
		Password:  mysqlrootpassword,
		Database:  "kubegems",
		Collation: "utf8mb4_unicode_ci",
	})
}
