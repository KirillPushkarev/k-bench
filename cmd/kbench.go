/*
Copyright 2019-2020 VMware, Inc.

SPDX-License-Identifier: Apache-2.0

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
	"encoding/json"
	"flag"
	"fmt"
	"k-bench/pkg/prometheus"
	"k-bench/util"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// Uncomment the following line to load the oidc plugin (only required to authenticate with oidc to your kubernetes cluster).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"sort"
	"time"
)

var defaultConfigPath = "./config/default/config.json"
var defaultOutDirPath = "."

func main() {
	var kubeconfig *string
	if conf, exists := os.LookupEnv("KUBECONFIG"); exists {
		kubeconfig = &conf
	} else if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"),
			"(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	var outDirPath *string

	// Provide the user input option to run a single config file, or all config files under a directory
	// The config file or directory should be under the current working directory
	benchmarkConfigPath := flag.String("benchconfig", defaultConfigPath, "(optional) benchmark config file")
	outDirPath = flag.String("outdir", defaultOutDirPath, "(optional) output directory for results, defaults to current directory")

	flag.Parse()

	benchmarkConfigs := readBenchmarkConfigs(benchmarkConfigPath)

	logFile, err := os.OpenFile(filepath.Join(*outDirPath, "kbench.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}

	defer logFile.Close()

	log.SetOutput(logFile)
	formatter := new(log.TextFormatter)
	formatter.FullTimestamp = true
	formatter.TimestampFormat = "2006-01-02T15:04:05.000"
	log.SetFormatter(formatter)

	log.Info("Starting kbench...")
	fmt.Printf("Starting benchmark, writing logs to " + filepath.Join(*outDirPath+"/kbench.log") + "... \n")

	testConfigs, configWithPrometheus := parseBenchmarkConfigs(benchmarkConfigs)

	k8sConfig, kubeerr := clientcmd.BuildConfigFromFlags("", *kubeconfig)

	if kubeerr != nil {
		fmt.Printf("Kube config file %v not valid, benchmark exited. \n", *kubeconfig)
		os.Exit(1)
		//panic(err)
	}

	if configWithPrometheus != nil {
		client, _ := kubernetes.NewForConfig(k8sConfig)
		dynClient, _ := dynamic.NewForConfig(k8sConfig)
		pc := prometheus.NewPrometheusController(client, &dynClient, k8sConfig, configWithPrometheus)
		pc.EnablePrometheus()
	}

	sortBenchmarkConfigsByWorkloadSize(testConfigs)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Info("Terminating the run after receiving SIGTERM signal.")
		util.Finalize()
		os.Exit(1)
	}()

	// Run each workload (specified by its config file) one after another in the sorted order
	for _, testConfig := range testConfigs {
		fmt.Printf("Running workload, please check kbench log for details... \n")
		util.Run(k8sConfig, testConfig, outDirPath)
		time.Sleep(time.Duration(testConfig.SleepTimeAfterRun) * time.Millisecond)
	}

	log.Info("Benchmark run completed.")
	fmt.Printf("Completed running benchmark, exit. \n")

	return
}

func readBenchmarkConfigs(benchmarkConfigPath *string) []string {
	benchmarkConfigFileInfo, err := os.Stat(*benchmarkConfigPath)
	if err != nil {
		log.Fatal(err)
	}

	var benchmarkConfigs []string
	if benchmarkConfigFileInfo.Mode().IsDir() {
		configDir := *benchmarkConfigPath
		f, err := os.Open(configDir)
		if err != nil {
			log.Fatal(err)
		}
		files, err := f.Readdir(-1)
		f.Close()
		if err != nil {
			log.Fatal(err)
		}
		for _, file := range files {
			if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
				benchmarkConfigs = append(benchmarkConfigs, configDir+"/"+file.Name())
			}
		}
	} else {
		benchmarkConfigs = append(benchmarkConfigs, *benchmarkConfigPath)
	}

	return benchmarkConfigs
}

func parseBenchmarkConfigs(benchmarkConfigs []string) ([]util.TestConfig, *util.TestConfig) {
	var testConfigs []util.TestConfig
	var configWithPrometheus *util.TestConfig

	for _, benchmarkConfigFile := range benchmarkConfigs {
		configFile, err := os.OpenFile(benchmarkConfigFile, os.O_RDWR, 0666)
		if err != nil {
			fmt.Printf("Can not open benchmark config file %v, benchmark exited. \n",
				benchmarkConfigFile)
			os.Exit(1)
		}

		configFile.Close()

		decoder := json.NewDecoder(configFile)
		testConfig := util.TestConfig{}
		err = decoder.Decode(&testConfig)

		if err != nil {
			fmt.Printf("Can not parse benchmark json config file, error: \n %v \n", err)
			log.Errorf("Can not parse benchmark json config file, error: %v", err)
			log.Info("Benchmark exited.")
			os.Exit(1)
		}

		if len(testConfig.PrometheusManifestPaths) != 0 {
			configWithPrometheus = &testConfig
		}
		testConfigs = append(testConfigs, testConfig)
	}

	return testConfigs, configWithPrometheus
}

func sortBenchmarkConfigsByWorkloadSize(benchmarkConfigs []util.TestConfig) {
	// Sort the config files by the lightness of workload, from light to heavy.
	// This is determined by the maximum number of pods in the config files
	sort.Slice(benchmarkConfigs, func(i, j int) bool {
		if len(benchmarkConfigs[i].Operations) != 0 && len(benchmarkConfigs[j].Operations) != 0 {
			iNumPodsMax := 0
			jNumPodsMax := 0
			for _, op := range benchmarkConfigs[i].Operations {
				if op.Pod.Count > iNumPodsMax {
					iNumPodsMax = op.Pod.Count
				}
			}
			for _, op := range benchmarkConfigs[j].Operations {
				if op.Pod.Count > jNumPodsMax {
					jNumPodsMax = op.Pod.Count
				}
			}
			return iNumPodsMax < jNumPodsMax
		}
		if len(benchmarkConfigs[i].Operations) == 0 {
			return true
		}
		return false
	})
}
