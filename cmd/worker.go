// Copyright 2018 Legrin, LLC
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/seecis/sauron/internal/machinery"
	"log"
	"github.com/seecis/sauron/internal/dataaccess"
	"github.com/spf13/viper"
	"fmt"
	"net/http"
	"net/http/httputil"
	"github.com/davecgh/go-spew/spew"
)

// workerCmd represents the worker command
var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Spins up a worker sauron to rule on",
	// Todo: expand documentation
	Long: `Workers are capable of running scheduled jobs`,
	Run: func(cmd *cobra.Command, args []string) {
		ew := ExtractionWorker{
			reportService:    dataaccess.NewMSSQLReportService(true, false),
			extractorService: dataaccess.NewMsSqlExtractorService(true, false),
		}

		v := viper.GetString("machinery-broker")
		master := machinery.NewMachineryWithBrokerAddress(v)
		w := master.NewWorker("worker", 1)
		master.RegisterTask("extract", ew.Extract)
		err := w.Launch()
		// todo: maybe add a worker scheduling mechanism?
		// Todo: maybe switch to actors?????
		if err != nil {
			log.Fatal(err)
		}
	},
}

type ExtractionWorker struct {
	reportService    dataaccess.ReportService
	extractorService dataaccess.ExtractorService
}

func (ew *ExtractionWorker) Extract(url string, extractorId, reportId string) error {
	extractor, err := ew.extractorService.Get(extractorId)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Get("http://localhost:8092/new?url="+url)
	if err != nil {
		return err
	}
	d, _ := httputil.DumpResponse(res, true)
	fmt.Println(string(d))

	fields, err := extractor.Extract(res.Body)

	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}

	spew.Dump(fields)
	err = ew.reportService.WriteAsReport(reportId, fields)
	return err
}

var machineryBrokerAddress string

func init() {
	rootCmd.AddCommand(workerCmd)

	//Todo: maybe add another scheduling mechanism
	workerCmd.Flags().StringVar(&machineryBrokerAddress,
		"machinery-broker",
		"redis://localhost:6379",
		"Provide address for machinery")

	viper.BindPFlag("machinery-broker", workerCmd.Flags().Lookup("machinery-broker"))
}
